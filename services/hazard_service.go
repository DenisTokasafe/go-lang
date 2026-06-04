package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"latihan1/cmd/web/config"
	"latihan1/cmd/web/helpers"
	"latihan1/middlewares"
	"latihan1/models"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type HazardService struct {
	DB *gorm.DB
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
func parseUint(val string) (uint, error) {
	num, err := strconv.ParseUint(val, 10, 64)
	return uint(num), err
}
func parseDateTimeLocal(val string) (time.Time, error) {
	loc, err := time.LoadLocation("Asia/Makassar") // Waktu WITA sesuai project Anda
	if err != nil {
		return time.Time{}, err
	}

	// 1. Coba parse format dengan SPASI (Format default Flatpickr Anda saat ini)
	if t, err := time.ParseInLocation("2006-01-02 15:04", val, loc); err == nil {
		return t, nil
	}

	// 2. Fallback: Coba parse format dengan 'T' (Standar HTML5 datetime-local)
	return time.ParseInLocation("2006-01-02T15:04", val, loc)
}

func NewHazardService(
	db *gorm.DB,
) *HazardService {

	return &HazardService{
		DB: db,
	}
}

type FileDTO struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	IsImage bool   `json:"is_image"`
}

type CorrectiveActionDTO struct {
	ID             uint   `json:"id"`
	FollowupAction string `json:"followup_action"`
	Type           string `json:"type"`

	DepartmentTerkaitID *uint `json:"department_terkait_id"`
	ContractorTerkaitID *uint `json:"contractor_terkait_id"`
	PicTerkaitID        *uint `json:"pic_terkait_id"`

	Pics []models.User `json:"pics"`

	DueDate     string `json:"due_date"`
	CompletedOn string `json:"completed_on"`
}

type HazardIndexResult struct {
	Hazards         []models.Hazard
	TotalRows       int64
	TotalPages      int
	CurrentPage     int
	Categories      []models.EventCategory
	Locations       []models.Location
	ScatOptions     []models.ScatOption
	RiskAssessments []models.RiskAssessmentCode
}

func (s *HazardService) GetHazards(
	search string,
	startDate string,
	endDate string,
	filterCategory string,
	filterLocation string,
	filterRisk string,
	filterScat string,
	page int,
) (HazardIndexResult, error) {

	if page < 1 {
		page = 1
	}

	pageSize := 10
	offset := (page - 1) * pageSize

	// =========================
	// BASE QUERY
	// =========================
	dbQuery := s.DB.
		Model(&models.Hazard{}).
		Preload("EventCategory").
		Preload("EventCategory.Parent").
		Preload("RiskMatrix").
		Preload("RiskMatrix.RiskAssessment").
		Preload("ScatOption").
		Preload("Location").
		Preload("ReportBy").
		Preload("Department").
		Preload("Contractor").
		Preload("PIC").
		Preload("Documentations.Documentation").
		Preload("CorrectiveActions").
		Preload("Audits").
		Order("tanggal_waktu DESC")

	// =========================
	// SEARCH FILTER
	// =========================
	if search != "" {

		searchLike := "%" + search + "%"

		dbQuery = dbQuery.
			Joins(`
				LEFT JOIN event_categories 
				ON event_categories.id = hazards.event_category_id
			`).
			Joins(`
				LEFT JOIN locations 
				ON locations.id = hazards.location_id
			`).
			Joins(`
				LEFT JOIN users report_by_user
				ON report_by_user.id = hazards.report_by_id
			`).
			Joins(`
				LEFT JOIN users pic_user
				ON pic_user.id = hazards.pic_id
			`).
			Where(`
				hazards.deskripsi LIKE ?
				OR hazards.corrective_action LIKE ?
				OR hazards.reporter_manual LIKE ?
				OR event_categories.name LIKE ?
				OR locations.name LIKE ?
				OR report_by_user.name LIKE ?
				OR pic_user.name LIKE ?
			`,
				searchLike,
				searchLike,
				searchLike,
				searchLike,
				searchLike,
				searchLike,
				searchLike,
			)
	}

	// =========================
	// DATE FILTER
	// =========================
	if startDate != "" {

		dbQuery = dbQuery.Where(
			"DATE(tanggal_waktu) >= ?",
			startDate,
		)
	}

	if endDate != "" {

		dbQuery = dbQuery.Where(
			"DATE(tanggal_waktu) <= ?",
			endDate,
		)
	}

	// =========================
	// CATEGORY FILTER
	// =========================
	if filterCategory != "" {

		dbQuery = dbQuery.Where(
			"hazards.event_category_id = ?",
			filterCategory,
		)
	}

	// =========================
	// LOCATION FILTER
	// =========================
	if filterLocation != "" {

		dbQuery = dbQuery.Where(
			"hazards.location_id = ?",
			filterLocation,
		)
	}

	// =========================
	// SCAT FILTER
	// =========================
	if filterScat != "" {

		dbQuery = dbQuery.Where(
			"hazards.scat_option_id = ?",
			filterScat,
		)
	}

	// =========================
	// RISK FILTER
	// =========================
	if filterRisk != "" {

		dbQuery = dbQuery.Joins(`
			LEFT JOIN risk_matrices
			ON risk_matrices.id = hazards.risk_matrix_id
		`).Where(
			"risk_matrices.risk_assessment_id = ?",
			filterRisk,
		)
	}

	// =========================
	// COUNT TOTAL
	// =========================
	var totalRows int64

	err := dbQuery.Count(&totalRows).Error

	if err != nil {
		return HazardIndexResult{}, err
	}

	// =========================
	// GET DATA
	// =========================
	var hazards []models.Hazard

	err = dbQuery.
		Limit(pageSize).
		Offset(offset).
		Find(&hazards).Error

	if err != nil {
		return HazardIndexResult{}, err
	}

	// =========================
	// CATEGORIES
	// =========================
	var categories []models.EventCategory

	subQuery := s.DB.
		Model(&models.Hazard{}).
		Select("DISTINCT event_category_id")

	err = s.DB.
		Model(&models.EventCategory{}).
		Where("id IN (?)", subQuery).
		Order("name ASC").
		Find(&categories).Error

	if err != nil {
		return HazardIndexResult{}, err
	}

	// =========================
	// LOCATIONS
	// =========================
	var locations []models.Location

	err = s.DB.
		Model(&models.Location{}).
		Distinct().
		Joins(`
			JOIN hazards
			ON hazards.location_id = locations.id
		`).
		Where("hazards.deleted_at IS NULL").
		Order("locations.name ASC").
		Find(&locations).Error

	if err != nil {
		return HazardIndexResult{}, err
	}

	// =========================
	// SCAT OPTIONS
	// =========================
	var scatOptions []models.ScatOption

	err = s.DB.
		Model(&models.ScatOption{}).
		Distinct().
		Joins(`
			JOIN hazards
			ON hazards.scat_option_id = scat_options.id
		`).
		Where("hazards.deleted_at IS NULL").
		Order("scat_options.name ASC").
		Find(&scatOptions).Error

	if err != nil {
		return HazardIndexResult{}, err
	}

	// =========================
	// RISK ASSESSMENTS
	// =========================
	var riskAssessments []models.RiskAssessmentCode

	err = s.DB.
		Order("name asc").
		Find(&riskAssessments).Error

	if err != nil {
		return HazardIndexResult{}, err
	}

	// =========================
	// PAGINATION
	// =========================
	totalPages := int(
		(totalRows + int64(pageSize) - 1) / int64(pageSize),
	)

	return HazardIndexResult{
		Hazards:         hazards,
		TotalRows:       totalRows,
		TotalPages:      totalPages,
		CurrentPage:     page,
		Categories:      categories,
		Locations:       locations,
		ScatOptions:     scatOptions,
		RiskAssessments: riskAssessments,
	}, nil
}
func (s *HazardService) uploadFiles(r *http.Request, fieldName string) ([]string, error) {
	// Pastikan ParseMultipartForm sudah dipanggil sebelumnya di handler,
	// namun pengecekan ini tetap bagus untuk safety.
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, fmt.Errorf("multipart form tidak tersedia atau belum di-parse")
	}

	files, ok := r.MultipartForm.File[fieldName]
	if !ok || len(files) == 0 {
		return nil, nil // Tidak ada file yang diupload di field ini, return nil (bukan error)
	}

	folder := "./public/uploads/hazards"
	if err := os.MkdirAll(folder, 0755); err != nil {
		return nil, fmt.Errorf("gagal membuat direktori: %v", err)
	}

	var results []string

	// Tambahkan index 'i' untuk mencegah bentrok nama file
	for i, header := range files {
		// Pindahkan logika ke fungsi terpisah agar return error bisa ditangkap
		err := func() error {
			// Cek ukuran file sebelum membukanya (2 MB limit)
			if header.Size > 2<<20 {
				return fmt.Errorf("ukuran file %s melebihi batas 2MB", header.Filename)
			}

			file, err := header.Open()
			if err != nil {
				return fmt.Errorf("gagal membuka file %s: %v", header.Filename, err)
			}
			defer file.Close()

			ext := filepath.Ext(header.Filename)
			// Tambahkan index (i) ke nama file agar pasti unik meskipun di-looping di nanosecond yang sama
			safeName := fmt.Sprintf("%d_%d%s", time.Now().UnixNano(), i, ext)
			path := filepath.Join(folder, safeName)

			dst, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("gagal menyimpan file %s: %v", header.Filename, err)
			}
			defer dst.Close()

			if _, err := io.Copy(dst, file); err != nil {
				return fmt.Errorf("gagal menyalin data file %s: %v", header.Filename, err)
			}

			results = append(results, "/public/uploads/hazards/"+safeName)
			return nil
		}()

		// Jika terjadi error pada satu file, kita hentikan seluruh proses
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}
func buildCorrectiveActionAudit(
	actions []models.CorrectiveActionHazard,
) []map[string]interface{} {

	var result []map[string]interface{}

	for _, a := range actions {

		result = append(result, map[string]interface{}{
			"followup_action": a.FollowupAction,

			"department": func() string {
				if a.DepartmentTerkait != nil {
					return a.DepartmentTerkait.Name
				}
				return ""
			}(),

			"contractor": func() string {
				if a.ContractorTerkait != nil {
					return a.ContractorTerkait.Name
				}
				return ""
			}(),

			"pic": func() string {
				if a.PICTerkait != nil {
					return a.PICTerkait.Name
				}
				return ""
			}(),

			"due_date":     a.DueDate,
			"completed_on": a.CompletedOn,
		})
	}

	return result
}
func buildHazardAuditData(h models.Hazard) map[string]interface{} {

	return map[string]interface{}{
		"event_category": h.EventCategory.Name,
		"risk_matrix": fmt.Sprintf(
			"%s × %s (%s)",
			h.RiskMatrix.RiskConsequence.Name,
			h.RiskMatrix.RiskLikelihood.Name,
			h.RiskMatrix.RiskAssessment.Name,
		),

		"report_by": func() string {
			if h.ReportBy != nil {
				return h.ReportBy.Name
			}
			return ""
		}(),

		"department": func() string {
			if h.Department != nil {
				return h.Department.Name
			}
			return ""
		}(),

		"contractor": func() string {
			if h.Contractor != nil {
				return h.Contractor.Name
			}
			return ""
		}(),

		"pic": h.PIC.Name,
		"corrective_actions": buildCorrectiveActionAudit(
			h.CorrectiveActions,
		),
		"description":       h.Deskripsi,
		"corrective_action": h.CorrectiveAction,
		"location_specific": h.LocationSpecific,
		"status":            h.Status,
	}
}
func (s *HazardService) CreateWithFiles(r *http.Request) (models.Hazard, error) {

	// ========================
	// 1. PARSE MULTIPART FORM
	// ========================
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		return models.Hazard{}, fmt.Errorf("gagal parse multipart: %w", err)
	}

	if r.MultipartForm == nil {
		return models.Hazard{}, fmt.Errorf("multipart form kosong (frontend tidak mengirim file dengan benar)")
	}

	// ========================
	// USER LOGIN
	// ========================
	var changedBy uint

	if val := r.Context().Value(middlewares.AuthUserKey); val != nil {

		if user, ok := val.(models.User); ok {
			changedBy = user.ID
		}
	}

	// ========================
	// 2. GET FORM VALUES
	// ========================
	eventCategoryIDStr := r.FormValue("event_type_id")
	scatOptionIDStr := r.FormValue("scat_option_id")
	locationIDStr := r.FormValue("location_id")
	departmentIDStr := r.FormValue("department_id")
	contractorIDStr := r.FormValue("contractor_id")
	picIDStr := r.FormValue("pic_id")
	description := r.FormValue("description")
	correctiveAction := r.FormValue("corrective_action")
	locationSpecific := r.FormValue("location_specific")
	reportByIDStr := r.FormValue("report_by_id")
	reportManual := r.FormValue("reporter_manual")
	eventDateStr := r.FormValue("event_date")
	riskMatrixIDStr := r.FormValue("risk_matrix_id")

	// ========================
	// CORRECTIVE_ACTION
	// ========================

	departmentTerkaitIDStr := r.FormValue("department_terkait_id")
	contractorTerkaitIDStr := r.FormValue("contractor_terkait_id")
	picTerkaitIDStr := r.FormValue("pic_terkait_id")

	dueDateLanjutanStr := r.FormValue("due_date_lanjutan")
	completionDateLanjutanStr := r.FormValue("completion_date_lanjutan")

	followUpAction := r.FormValue("followup_action")

	// ========================
	// 3. VALIDASI WAJIB
	// ========================
	required := map[string]string{
		"event_type_id":  eventCategoryIDStr,
		"scat_option_id": scatOptionIDStr,
		"location_id":    locationIDStr,
		"pic_id":         picIDStr,
		"event_date":     eventDateStr,
		"risk_matrix_id": riskMatrixIDStr,
	}

	for k, v := range required {
		if v == "" {
			return models.Hazard{}, fmt.Errorf("field %s wajib diisi", k)
		}
	}

	// ========================
	// 4. PARSE DATA
	// ========================
	eventCategoryID, err := parseUint(eventCategoryIDStr)
	if err != nil {
		return models.Hazard{}, fmt.Errorf("event_type_id tidak valid")
	}

	scatOptionID, err := parseUint(scatOptionIDStr)
	if err != nil {
		return models.Hazard{}, fmt.Errorf("scat_option_id tidak valid")
	}

	locationID, err := parseUint(locationIDStr)
	if err != nil {
		return models.Hazard{}, fmt.Errorf("location_id tidak valid")
	}

	riskMatrixID, err := parseUint(riskMatrixIDStr)
	if err != nil {
		return models.Hazard{}, fmt.Errorf("risk_matrix_id tidak valid")
	}

	picID, err := parseUint(picIDStr)
	if err != nil {
		return models.Hazard{}, fmt.Errorf("pic_id tidak valid")
	}

	fmt.Println("date", eventDateStr)
	eventDate, err := parseDateTimeLocal(eventDateStr)
	if err != nil {
		return models.Hazard{}, fmt.Errorf("event_date tidak valid")
	}

	if eventDate.After(time.Now()) {
		return models.Hazard{}, fmt.Errorf("tanggal tidak boleh melebihi waktu sekarang")
	}

	var dueDate *time.Time

	if dueDateLanjutanStr != "" {

		parsedDueDate, err := time.Parse(
			"2006-01-02",
			dueDateLanjutanStr,
		)

		if err != nil {
			return models.Hazard{}, fmt.Errorf("due_date_lanjutan tidak valid")
		}

		dueDate = &parsedDueDate
	}

	var completedOn *time.Time

	if completionDateLanjutanStr != "" {

		parsedCompletedOn, err := time.Parse(
			"2006-01-02",
			completionDateLanjutanStr,
		)

		if err != nil {
			return models.Hazard{}, fmt.Errorf("completion_date_lanjutan tidak valid")
		}

		completedOn = &parsedCompletedOn
	}

	// ========================
	// OPTIONAL RELATION
	// ========================
	var reportByID *uint

	if reportManual == "" && reportByIDStr != "" {
		if id, err := parseUint(reportByIDStr); err == nil {
			reportByID = &id
		}
	}

	var departmentID *uint

	if departmentIDStr != "" {
		if id, err := parseUint(departmentIDStr); err == nil {
			departmentID = &id
		}
	}

	var contractorID *uint

	if contractorIDStr != "" {
		if id, err := parseUint(contractorIDStr); err == nil {
			contractorID = &id
		}

	}

	var departmentTerkaitID *uint

	if departmentTerkaitIDStr != "" {
		if id, err := parseUint(departmentTerkaitIDStr); err == nil {
			departmentTerkaitID = &id
		}
	}

	var contractorTerkaitID *uint

	if contractorTerkaitIDStr != "" {
		if id, err := parseUint(contractorTerkaitIDStr); err == nil {
			contractorTerkaitID = &id
		}
	}

	var picTerkaitID *uint

	if picTerkaitIDStr != "" {
		if id, err := parseUint(picTerkaitIDStr); err == nil {
			picTerkaitID = &id
		}
	}

	// ========================
	// 5. START TRANSACTION
	// ========================
	if s.DB == nil {
		return models.Hazard{}, fmt.Errorf("database connection is nil")
	}

	tx := s.DB.Begin()

	if tx.Error != nil {
		return models.Hazard{}, tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	var status models.HazardStatus

	if picTerkaitID != nil {
		status = models.HazardStatusSubmit
	} else {
		status = models.HazardStatusClosed
	}
	// ========================
	// 6. CREATE HAZARD
	// ========================
	hazard := models.Hazard{
		EventCategoryID:  eventCategoryID,
		ScatOptionID:     scatOptionID,
		LocationID:       locationID,
		LocationSpecific: locationSpecific,
		PicID:            picID,
		Deskripsi:        description,
		CorrectiveAction: correctiveAction,
		TanggalWaktu:     eventDate,
		ReportByID:       reportByID,
		ReporterManual:   reportManual,
		RiskMatrixID:     riskMatrixID,
		DepartmentID:     departmentID,
		ContractorID:     contractorID,
		Status:           status,
	}

	if err := tx.Create(&hazard).Error; err != nil {
		tx.Rollback()
		return models.Hazard{}, err
	}

	// ========================
	// CREATE CORRECTIVE ACTION
	// ========================
	if followUpAction != "" {

		correctiveActionHazard := models.CorrectiveActionHazard{
			HazardID:            hazard.ID,
			FollowupAction:      followUpAction,
			DepartmentTerkaitID: departmentTerkaitID,
			ContractorTerkaitID: contractorTerkaitID,
			PicTerkaitID:        picTerkaitID,
			DueDate:             dueDate,
			CompletedOn:         completedOn,
		}

		if err := tx.Create(&correctiveActionHazard).Error; err != nil {
			tx.Rollback()
			return models.Hazard{}, err
		}
	}

	// ========================
	// AUDIT CREATE
	// ========================
	audit := models.HazardAudit{
		HazardID:  hazard.ID,
		Action:    "CREATE",
		After:     toJSON(hazard),
		ChangedBy: changedBy,
		ChangedAt: time.Now(),
	}

	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		return models.Hazard{}, err
	}

	// ========================
	// 7. HANDLE FILE UPLOAD
	// ========================
	descFiles, err := s.uploadFiles(r, "dokumentasi_desc")
	if err != nil {
		tx.Rollback()
		return models.Hazard{}, err
	}

	correctiveFiles, err := s.uploadFiles(r, "dokumentasi_corrective")
	if err != nil {
		tx.Rollback()
		return models.Hazard{}, err
	}

	// ========================
	// 8. Moderator Notifikasi
	// ========================
	var category models.EventCategory
	if err := tx.First(&category, hazard.EventCategoryID).Error; err != nil {
		tx.Rollback()
		return models.Hazard{}, err
	}

	// Jika ParentID kosong, jangan kirim notifikasi
	if category.ParentID == nil {
		fmt.Println("ParentID kosong, notifikasi moderator dilewati")
	} else {
		parentID := *category.ParentID

		var moderators []models.User
		tx.Model(&models.User{}).
			Joins("JOIN user_event_categories uec ON uec.user_id = users.id").
			Where("uec.event_category_id = ?", parentID).
			Find(&moderators)

		// di sini baru lakukan insert ke tabel notifikasi / kirim pesan
		fmt.Println("Kirim data ke moderator:", moderators)
	}

	// ========================
	// 9. SAVE DOCUMENTATION
	// ========================
	saveDocs := func(files []string, docType string) error {

		for _, fileURL := range files {

			doc := models.Documentation{
				FileURL: fileURL,
			}

			if err := tx.Create(&doc).Error; err != nil {
				return err
			}

			pivot := models.HazardDocumentation{
				HazardID:        hazard.ID,
				DocumentationID: doc.ID,
				DocType:         docType,
			}

			if err := tx.Create(&pivot).Error; err != nil {
				return err
			}
		}

		return nil
	}

	if err := saveDocs(descFiles, "desc"); err != nil {
		tx.Rollback()
		return models.Hazard{}, err
	}

	if err := saveDocs(correctiveFiles, "corrective"); err != nil {
		tx.Rollback()
		return models.Hazard{}, err
	}

	// ========================
	// 9. COMMIT
	// ========================
	if err := tx.Commit().Error; err != nil {
		return models.Hazard{}, err
	}

	// ========================
	// 10. RETURN
	// ========================
	return hazard, nil
}

func (s *HazardService) UpdateWithFiles(id uint, r *http.Request) (models.Hazard, error) {

	// ========================
	// PARSE MULTIPART
	// ========================
	if err := r.ParseMultipartForm(10 << 20); err != nil {

		return models.Hazard{},
			fmt.Errorf(
				"gagal parse multipart: %w",
				err,
			)
	}

	if r.MultipartForm == nil {

		return models.Hazard{},
			fmt.Errorf("multipart form kosong")
	}

	// ========================
	// USER LOGIN
	// ========================
	var changedBy uint

	if val := r.Context().
		Value(middlewares.AuthUserKey); val != nil {

		if user, ok := val.(models.User); ok {

			changedBy = user.ID
		}
	}

	// ========================
	// GET OLD HAZARD
	// ========================
	var oldHazard models.Hazard

	if err := s.DB.
		Preload("EventCategory").
		Preload("RiskMatrix").
		Preload("RiskMatrix.RiskConsequence").
		Preload("RiskMatrix.RiskLikelihood").
		Preload("RiskMatrix.RiskAssessment").
		Preload("ReportBy").
		Preload("Department").
		Preload("Contractor").
		Preload("PIC").
		Preload("Location").
		Preload("Documentations.Documentation").
		Preload("CorrectiveActions").
		Preload("CorrectiveActions.DepartmentTerkait").
		Preload("CorrectiveActions.ContractorTerkait").
		Preload("CorrectiveActions.PICTerkait").
		First(&oldHazard, id).Error; err != nil {

		return models.Hazard{},
			fmt.Errorf(
				"hazard lama tidak ditemukan: %w",
				err,
			)
	}

	before := toJSON(
		buildHazardAuditData(oldHazard),
	)

	// ========================
	// GET CURRENT HAZARD
	// ========================
	var hazard models.Hazard

	if err := s.DB.
		Preload("Documentations.Documentation").
		Preload("ReportBy").
		Preload("Location").
		Preload("EventCategory").
		Preload("RiskMatrix").
		First(&hazard, id).Error; err != nil {

		return models.Hazard{},
			fmt.Errorf("hazard tidak ditemukan")
	}

	// ========================
	// FORM VALUES
	// ========================
	eventCategoryIDStr := r.FormValue("event_type_id")
	scatOptionIDStr := r.FormValue("scat_option_id")
	locationIDStr := r.FormValue("location_id")

	departmentIDStr := r.FormValue("department_id")
	contractorIDStr := r.FormValue("contractor_id")

	picIDStr := r.FormValue("pic_id")

	description := r.FormValue("description")
	correctiveAction := r.FormValue("corrective_action")

	locationSpecific := r.FormValue("location_specific")

	reportByIDStr := r.FormValue("report_by_id")

	reportManual := r.FormValue("reporter_manual")

	eventDateStr := r.FormValue("event_date")
	riskMatrixIDStr := r.FormValue("risk_matrix_id")

	deletedFiles := r.MultipartForm.
		Value["deleted_files[]"]

	// ========================
	// VALIDASI REQUIRED
	// ========================
	required := map[string]string{
		"event_type_id":  eventCategoryIDStr,
		"scat_option_id": scatOptionIDStr,
		"location_id":    locationIDStr,
		"pic_id":         picIDStr,
		"event_date":     eventDateStr,
		"risk_matrix_id": riskMatrixIDStr,
	}

	for k, v := range required {

		if strings.TrimSpace(v) == "" {

			return models.Hazard{},
				fmt.Errorf(
					"field %s wajib diisi",
					k,
				)
		}
	}

	// ========================
	// PARSE DATA
	// ========================
	eventCategoryID, err := parseUint(
		eventCategoryIDStr,
	)

	if err != nil {

		return models.Hazard{},
			fmt.Errorf(
				"event_type_id tidak valid",
			)
	}

	scatOptionID, err := parseUint(
		scatOptionIDStr,
	)

	if err != nil {

		return models.Hazard{},
			fmt.Errorf(
				"scat_option_id tidak valid",
			)
	}

	locationID, err := parseUint(
		locationIDStr,
	)

	if err != nil {

		return models.Hazard{},
			fmt.Errorf(
				"location_id tidak valid",
			)
	}

	riskMatrixID, err := parseUint(
		riskMatrixIDStr,
	)

	if err != nil {

		return models.Hazard{},
			fmt.Errorf(
				"risk_matrix_id tidak valid",
			)
	}

	picID, err := parseUint(
		picIDStr,
	)

	if err != nil {

		return models.Hazard{},
			fmt.Errorf(
				"pic_id tidak valid",
			)
	}

	eventDate, err := parseDateTimeLocal(
		eventDateStr,
	)

	if err != nil {

		return models.Hazard{},
			fmt.Errorf(
				"event_date tidak valid",
			)
	}

	// ========================
	// OPTIONAL POINTERS
	// ========================
	fmt.Printf("departmentIDStr: %#v\n", departmentIDStr)
	fmt.Printf("contractorIDStr: %#v\n", contractorIDStr)
	// helper validasi id kosong/null
	isValidID := func(v string) bool {

		v = strings.ToLower(
			strings.TrimSpace(v),
		)

		invalidValues := map[string]bool{
			"":                true,
			"0":               true,
			"null":            true,
			"undefined":       true,
			"nan":             true,
			"false":           true,
			"[object object]": true,
			"<nil>":           true,
		}

		return !invalidValues[v]
	}

	// ========================
	// REPORT BY
	// ========================
	var reportByID *uint

	if reportManual == "" &&
		isValidID(reportByIDStr) {

		id, err := parseUint(
			reportByIDStr,
		)

		if err != nil {

			return models.Hazard{},
				fmt.Errorf(
					"report_by_id tidak valid",
				)
		}

		reportByID = &id
		fmt.Printf(
			"report_by_id dipanggil: %d\n",
			*reportByID,
		)
	}

	// ========================
	// DEPARTMENT / CONTRACTOR
	// ========================
	var departmentID *uint
	var contractorID *uint

	departmentIDStr = strings.TrimSpace(
		departmentIDStr,
	)

	contractorIDStr = strings.TrimSpace(
		contractorIDStr,
	)

	// ========================
	// TIDAK BOLEH BERSAMAAN
	// ========================
	if isValidID(departmentIDStr) &&
		isValidID(contractorIDStr) {

		return models.Hazard{},
			fmt.Errorf(
				"department dan contractor tidak boleh dipilih bersamaan",
			)
	}

	// ========================
	// DEPARTMENT
	// ========================
	if isValidID(departmentIDStr) {

		id, err := parseUint(
			departmentIDStr,
		)

		if err != nil {

			return models.Hazard{},
				fmt.Errorf(
					"department_id tidak valid",
				)
		}

		departmentID = &id

		// contractor wajib null
		contractorID = nil
	}

	// ========================
	// CONTRACTOR
	// ========================
	if isValidID(contractorIDStr) {

		id, err := parseUint(
			contractorIDStr,
		)

		if err != nil {

			return models.Hazard{},
				fmt.Errorf(
					"contractor_id tidak valid",
				)
		}

		contractorID = &id

		// department wajib null
		departmentID = nil
	}

	// ========================
	// START TX
	// ========================
	tx := s.DB.Begin()

	if tx.Error != nil {

		return models.Hazard{},
			tx.Error
	}

	committed := false

	defer func() {

		if !committed {

			tx.Rollback()
		}
	}()

	// ========================
	// CORRECTIVE ACTIONS
	// ========================
	followupActions := r.MultipartForm.
		Value["followup_action[]"]

	types := r.MultipartForm.
		Value["type[]"]

	departmentTerkaitIDs := r.MultipartForm.
		Value["department_terkait_id[]"]

	contractorTerkIDs := r.MultipartForm.
		Value["contractor_terkait_id[]"]

	picTerkIDs := r.MultipartForm.
		Value["pic_terkait_id[]"]

	dueDatesCorrective := r.MultipartForm.
		Value["due_date[]"]

	completedDatesCorrective := r.MultipartForm.
		Value["completed_on[]"]

	// ========================
	// STATUS LOGIC
	// ========================
	var status models.HazardStatus

	hasAnyAction := false
	allCompleted := true

	for i := range followupActions {

		if strings.TrimSpace(
			followupActions[i],
		) == "" {

			continue
		}

		hasAnyAction = true

		if i >= len(completedDatesCorrective) ||
			strings.TrimSpace(
				completedDatesCorrective[i],
			) == "" {

			allCompleted = false
		}
	}

	if !hasAnyAction {

		status = models.HazardStatusSubmit

	} else if allCompleted {

		status = models.HazardStatusClosed

	} else {

		status = models.HazardStatusInProgress
	}

	// ========================
	// DELETE OLD ACTIONS
	// ========================
	if err := tx.
		Where(
			"hazard_id = ?",
			hazard.ID,
		).
		Delete(
			&models.CorrectiveActionHazard{},
		).Error; err != nil {

		return models.Hazard{}, err
	}

	// ========================
	// UPDATE HAZARD
	// ========================
	updateData := map[string]interface{}{
		"event_category_id": eventCategoryID,
		"scat_option_id":    scatOptionID,

		"location_id":       locationID,
		"location_specific": locationSpecific,

		"pic_id": picID,

		"deskripsi":         description,
		"corrective_action": correctiveAction,

		"tanggal_waktu": eventDate,

		"risk_matrix_id": riskMatrixID,
		"status":         status,

		"department_id": departmentID,
		"contractor_id": contractorID,
	}

	// ========================
	// REPORTER LOGIC
	// ========================
	if strings.TrimSpace(reportManual) != "" {

		// manual reporter
		updateData["reporter_manual"] = reportManual
		updateData["report_by_id"] = nil

	} else {

		// pilih user reporter
		updateData["reporter_manual"] = ""
		updateData["report_by_id"] = reportByID
	}

	// ========================
	// UPDATE DATABASE
	// ========================
	if err := tx.
		Model(&models.Hazard{}).
		Where("id = ?", hazard.ID).
		Updates(updateData).Error; err != nil {

		return models.Hazard{}, err
	}

	fmt.Printf("reportByIDStr: %s\n", reportByIDStr)

	if reportByID != nil {

		fmt.Printf("reportByID value: %d\n", *reportByID)

	} else {

		fmt.Println("reportByID nil")
	}

	fmt.Printf("reportManual: %s\n", reportManual)

	// ========================
	// SAVE ACTIONS
	// ========================
	for i := range followupActions {

		if strings.TrimSpace(
			followupActions[i],
		) == "" {

			continue
		}

		action := models.CorrectiveActionHazard{
			HazardID:       hazard.ID,
			FollowupAction: followupActions[i],
		}

		actionType := ""

		if i < len(types) {

			actionType = types[i]
		}

		// DEPARTMENT
		if actionType == "department" {

			if i < len(
				departmentTerkaitIDs,
			) {

				idStr := strings.TrimSpace(
					departmentTerkaitIDs[i],
				)

				if idStr != "" {

					id, err := parseUint(idStr)

					if err != nil {

						return models.Hazard{},
							fmt.Errorf(
								"department terkait tidak valid",
							)
					}

					action.DepartmentTerkaitID = &id
				}
			}
		}

		// CONTRACTOR
		if actionType == "contractor" {

			if i < len(
				contractorTerkIDs,
			) {

				idStr := strings.TrimSpace(
					contractorTerkIDs[i],
				)

				if idStr != "" {

					id, err := parseUint(idStr)

					if err != nil {

						return models.Hazard{},
							fmt.Errorf(
								"contractor terkait tidak valid",
							)
					}

					action.ContractorTerkaitID = &id
				}
			}
		}

		// PIC
		if i < len(picTerkIDs) {

			idStr := strings.TrimSpace(
				picTerkIDs[i],
			)

			if idStr != "" {

				id, err := parseUint(idStr)

				if err != nil {

					return models.Hazard{},
						fmt.Errorf(
							"pic terkait tidak valid",
						)
				}

				action.PicTerkaitID = &id
			}
		}

		// DUE DATE
		if i < len(dueDatesCorrective) {

			dateStr := strings.TrimSpace(
				dueDatesCorrective[i],
			)

			if dateStr != "" {

				t, err := time.Parse(
					"2006-01-02",
					dateStr,
				)

				if err != nil {

					return models.Hazard{},
						fmt.Errorf(
							"due_date tidak valid",
						)
				}

				action.DueDate = &t
			}
		}

		// COMPLETED ON
		if i < len(
			completedDatesCorrective,
		) {

			dateStr := strings.TrimSpace(
				completedDatesCorrective[i],
			)

			if dateStr != "" {

				t, err := time.Parse(
					"2006-01-02",
					dateStr,
				)

				if err != nil {

					return models.Hazard{},
						fmt.Errorf(
							"completed_on tidak valid",
						)
				}

				action.CompletedOn = &t
			}
		}

		if err := tx.
			Create(&action).Error; err != nil {

			return models.Hazard{}, err
		}

		fmt.Printf(
			"SUCCESS INSERT ACTION ID: %d\n",
			action.ID,
		)
	}

	// ========================
	// DELETE FILES
	// ========================
	var filesToDeletePhysical []string // <--- TAMBAHAN: Variabel penampung

	if len(deletedFiles) > 0 {

		for _, idStr := range deletedFiles {

			docID, err := strconv.ParseUint(
				idStr,
				10,
				64,
			)

			if err != nil {
				continue
			}

			// <--- TAMBAHAN: 1. Ambil data URL file SEBELUM datanya dihapus
			var doc models.Documentation
			if err := tx.First(&doc, docID).Error; err == nil {
				filesToDeletePhysical = append(filesToDeletePhysical, doc.FileURL)
			}

			// 2. Hapus relasi pivot
			if err := tx.
				Where(
					"documentation_id = ? AND hazard_id = ?",
					docID,
					hazard.ID,
				).
				Delete(
					&models.HazardDocumentation{},
				).Error; err != nil {

				return models.Hazard{}, err
			}

			// 3. Hapus data dokumen utamanya
			if err := tx.
				Delete(
					&models.Documentation{},
					docID,
				).Error; err != nil {

				return models.Hazard{}, err
			}
		}
	}

	// ========================
	// UPLOAD FILES
	// ========================
	descFiles, err := s.uploadFiles(
		r,
		"dokumentasi_desc",
	)

	if err != nil {

		return models.Hazard{}, err
	}

	correctiveFiles, err := s.uploadFiles(
		r,
		"dokumentasi_corrective",
	)

	if err != nil {

		return models.Hazard{}, err
	}

	// ========================
	// SAVE DOCUMENTS
	// ========================
	saveDocs := func(
		files []string,
		docType string,
	) error {

		for _, fileURL := range files {

			doc := models.Documentation{
				FileURL: fileURL,
			}

			if err := tx.
				Create(&doc).Error; err != nil {

				return err
			}

			pivot := models.HazardDocumentation{
				HazardID:        hazard.ID,
				DocumentationID: doc.ID,
				DocType:         docType,
			}

			if err := tx.
				Create(&pivot).Error; err != nil {

				return err
			}
		}

		return nil
	}

	if err := saveDocs(
		descFiles,
		"desc",
	); err != nil {

		return models.Hazard{}, err
	}

	if err := saveDocs(
		correctiveFiles,
		"corrective",
	); err != nil {

		return models.Hazard{}, err
	}

	// ========================
	// UPDATED DATA
	// ========================
	var updatedHazard models.Hazard

	if err := tx.
		Preload("EventCategory").
		Preload("RiskMatrix").
		Preload("RiskMatrix.RiskConsequence").
		Preload("RiskMatrix.RiskLikelihood").
		Preload("RiskMatrix.RiskAssessment").
		Preload("ReportBy").
		Preload("Department").
		Preload("Contractor").
		Preload("PIC").
		Preload("Location").
		Preload("Documentations.Documentation").
		Preload("CorrectiveActions").
		Preload("CorrectiveActions.DepartmentTerkait").
		Preload("CorrectiveActions.ContractorTerkait").
		Preload("CorrectiveActions.PICTerkait").
		First(
			&updatedHazard,
			hazard.ID,
		).Error; err != nil {

		return models.Hazard{}, err
	}

	after := toJSON(
		buildHazardAuditData(updatedHazard),
	)

	// ========================
	// SAVE AUDIT
	// ========================
	audit := models.HazardAudit{
		HazardID:  hazard.ID,
		Action:    "UPDATE",
		Before:    before,
		After:     after,
		ChangedBy: changedBy,
		ChangedAt: time.Now(),
	}

	if err := tx.
		Create(&audit).Error; err != nil {

		return models.Hazard{}, err
	}

	// ========================
	// COMMIT
	// ========================
	if err := tx.Commit().Error; err != nil {

		return models.Hazard{}, err
	}

	committed = true

	// ========================
	// HAPUS FILE FISIK (SETELAH COMMIT SUKSES)
	// ========================
	// Kita hapus filenya sekarang karena database sudah dipastikan aman.
	// Anda bisa menggunakan goroutine di sini agar tidak memblokir response
	go func(files []string) {
		for _, fileURL := range files {
			// Gunakan helper function deletePhysicalFile yang kita bahas sebelumnya
			// Anda mungkin perlu membuat function helper tersendiri di service ini
			deletePhysicalFile(fileURL)
		}
	}(filesToDeletePhysical)

	// ========================
	// SEND EMAIL
	// ========================
	go s.sendUpdateNotification(
		updatedHazard,
	)

	return updatedHazard, nil
}
func FileURL(path string) string {
	if path == "" {
		return ""
	}
	return "/" + path
}
func prettyJSON(input string) string {

	var out bytes.Buffer

	err := json.Indent(&out, []byte(input), "", "  ")
	if err != nil {
		return input
	}

	return out.String()
}

func (s *HazardService) sendUpdateNotification(
	h models.Hazard,
) {

	defer func() {

		if r := recover(); r != nil {

			fmt.Println(
				"panic email:",
				r,
			)
		}
	}()

	// ========================
	// GET PIC EMAIL
	// ========================
	picEmail, err := helpers.GetPICEmail(
		s.DB,
		h.PicID,
	)

	if err != nil {

		fmt.Println(
			"gagal ambil pic email:",
			err,
		)
	}

	// ========================
	// GET MODERATOR EMAILS
	// ========================
	moderatorEmails, err := helpers.GetModeratorEmails(s.DB, h.EventCategoryID)

	if err != nil {

		fmt.Println(
			"gagal ambil moderator email:",
			err,
		)
	}

	// ========================
	// REPORTER NAME
	// ========================
	namaPelapor := "Sistem / Anonim"

	if h.ReportBy != nil {

		namaPelapor = h.ReportBy.Name
	}

	// ========================
	// BASIC DATA
	// ========================
	locationName := "-"

	if h.Location.Name != "" {

		locationName = h.Location.Name
	}

	categoryName := "-"

	if h.EventCategory.Name != "" {

		categoryName = h.EventCategory.Name
	}

	riskLevel := "-"

	if h.RiskMatrix.
		RiskAssessment.Name != "" {

		riskLevel = h.RiskMatrix.
			RiskAssessment.Name
	}

	hazardURL := fmt.Sprintf(
		"%s/hazard/edit/%d",
		config.AppURL(),
		h.ID,
	)

	// ========================
	// EMAIL PIC
	// ========================
	if picEmail != "" {

		picHTML := fmt.Sprintf(`
		<p>Halo PIC,</p>

		<p>
			Anda ditunjuk sebagai PIC hazard berikut:
		</p>

		<table border="1"
			cellpadding="8"
			cellspacing="0">

			<tr>
				<td><strong>Kategori</strong></td>
				<td>%s</td>
			</tr>

			<tr>
				<td><strong>Lokasi</strong></td>
				<td>%s</td>
			</tr>

			<tr>
				<td><strong>Deskripsi</strong></td>
				<td>%s</td>
			</tr>

			<tr>
				<td><strong>Risk</strong></td>
				<td>%s</td>
			</tr>

			<tr>
				<td><strong>Pelapor</strong></td>
				<td>%s</td>
			</tr>

			<tr>
				<td><strong>Status</strong></td>
				<td>%s</td>
			</tr>

		</table>
		<p style="margin-top:20px;">
			<a href="%s"
				style="
					background:#2563eb;
					color:white;
					padding:12px 20px;
					text-decoration:none;
					border-radius:6px;
					display:inline-block;
					font-weight:bold;
				">
				Buka Laporan Hazard
			</a>
		</p>
		`,
			categoryName,
			locationName,
			h.Deskripsi,
			riskLevel,
			namaPelapor,
			h.Status,
			hazardURL,
		)

		err := config.SendEmail(
			[]string{picEmail},
			fmt.Sprintf(
				"SENTRY PIC Hazard #%d",
				h.ID,
			),
			config.EmailTemplate(
				"SENTRY Hazard PIC",
				"Penugasan PIC Hazard",
				picHTML,
			),
		)

		if err != nil {

			fmt.Println(
				"gagal kirim email PIC:",
				err,
			)
		}
	}

	// ========================
	// EMAIL MODERATOR
	// ========================
	if len(moderatorEmails) > 0 {

		moderatorHTML := fmt.Sprintf(`
		<p>Halo Moderator HSE,</p>

		<p>
			Terdapat update hazard pada kategori yang Anda moderasi.
		</p>

		<table border="1"
			cellpadding="8"
			cellspacing="0">

			<tr>
				<td><strong>Kategori</strong></td>
				<td>%s</td>
			</tr>

			<tr>
				<td><strong>Lokasi</strong></td>
				<td>%s</td>
			</tr>

			<tr>
				<td><strong>Deskripsi</strong></td>
				<td>%s</td>
			</tr>

			<tr>
				<td><strong>Risk</strong></td>
				<td>%s</td>
			</tr>

			<tr>
				<td><strong>Pelapor</strong></td>
				<td>%s</td>
			</tr>

			<tr>
				<td><strong>Status</strong></td>
				<td>%s</td>
			</tr>

		</table>
		<p style="margin-top:20px;">
			<a href="%s"
				style="
					background:#2563eb;
					color:white;
					padding:12px 20px;
					text-decoration:none;
					border-radius:6px;
					display:inline-block;
					font-weight:bold;
				">
				Buka Laporan Hazard
			</a>
		</p>
		`,
			categoryName,
			locationName,
			h.Deskripsi,
			riskLevel,
			namaPelapor,
			h.Status,
			hazardURL,
		)

		err := config.SendEmail(
			moderatorEmails,
			fmt.Sprintf(
				"SENTRY Moderator Hazard #%d",
				h.ID,
			),
			config.EmailTemplate(
				"SENTRY Hazard Moderator",
				"Update Hazard Moderator",
				moderatorHTML,
			),
		)

		if err != nil {

			fmt.Println(
				"gagal kirim email moderator:",
				err,
			)
		}
	}
}

func deletePhysicalFile(fileURL string) {
	if fileURL == "" {
		return
	}

	// Konversi dari URL database "/public/uploads/hazards/xxx.png"
	// Menjadi file path OS "./public/uploads/hazards/xxx.png"
	localPath := filepath.Clean("." + fileURL)

	err := os.Remove(localPath)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Gagal menghapus file fisik %s: %v", localPath, err)
	}
}
