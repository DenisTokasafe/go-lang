package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"latihan1/middlewares"
	"latihan1/models"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	loc, err := time.LoadLocation("Asia/Makassar") // sesuaikan
	if err != nil {
		return time.Time{}, err
	}

	return time.ParseInLocation("2006-01-02T15:04", val, loc)
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

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, fmt.Errorf("multipart form tidak tersedia")
	}

	files, ok := r.MultipartForm.File[fieldName]
	if !ok || len(files) == 0 {
		return nil, nil
	}

	folder := "./public/uploads/hazards"

	if err := os.MkdirAll(folder, 0755); err != nil {
		return nil, err
	}

	var results []string

	for _, header := range files {

		file, err := header.Open()
		if err != nil {
			continue
		}

		func() {
			defer file.Close()

			if header.Size > 2<<20 {
				return
			}

			ext := filepath.Ext(header.Filename)
			safeName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
			path := filepath.Join(folder, safeName)

			dst, err := os.Create(path)
			if err != nil {
				return
			}
			defer dst.Close()

			io.Copy(dst, file)

			results = append(results, "/public/uploads/hazards/"+safeName)
		}()
	}

	return results, nil
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

type CorrectiveActionPayload struct {
	ID uint

	FollowupAction string
	Type           string

	DepartmentTerkaitID *uint
	ContractorTerkaitID *uint
	PicTerkaitID        *uint

	DueDate     *time.Time
	CompletedOn *time.Time
}

func (s *HazardService) UpdateWithFiles(id uint, r *http.Request) (models.Hazard, error) {

	// ========================
	// PARSE MULTIPART
	// ========================
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		return models.Hazard{}, fmt.Errorf("gagal parse multipart: %w", err)
	}

	if r.MultipartForm == nil {
		return models.Hazard{}, fmt.Errorf("multipart form kosong")
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
	// STEP 1: AMBIL DATA LAMA
	// ========================
	var oldHazard models.Hazard

	s.DB.First(&oldHazard, id)

	// ========================
	// STEP 2: SNAPSHOT BEFORE
	// ========================
	before := toJSON(oldHazard)

	// ========================
	// GET EXISTING DATA
	// ========================
	var hazard models.Hazard

	if err := s.DB.
		Preload("Documentations.Documentation").
		First(&hazard, id).Error; err != nil {

		return models.Hazard{}, fmt.Errorf("hazard tidak ditemukan")
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

	deletedFiles := r.MultipartForm.Value["deleted_files[]"]

	// ========================
	// VALIDASI WAJIB
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
	// PARSE DATA
	// ========================
	eventCategoryID, _ := parseUint(eventCategoryIDStr)
	scatOptionID, _ := parseUint(scatOptionIDStr)
	locationID, _ := parseUint(locationIDStr)
	riskMatrixID, _ := parseUint(riskMatrixIDStr)
	picID, _ := parseUint(picIDStr)
	eventDate, _ := parseDateTimeLocal(eventDateStr)

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

	// ========================
	// TX START
	// ========================
	tx := s.DB.Begin()

	// ========================
	// UPDATE HAZARD
	// ========================
	hazard.EventCategoryID = eventCategoryID
	hazard.ScatOptionID = scatOptionID
	hazard.LocationID = locationID
	hazard.LocationSpecific = locationSpecific
	hazard.PicID = picID
	hazard.Deskripsi = description
	hazard.CorrectiveAction = correctiveAction
	hazard.TanggalWaktu = eventDate
	hazard.ReportByID = reportByID
	hazard.ReporterManual = reportManual
	hazard.RiskMatrixID = riskMatrixID
	hazard.DepartmentID = departmentID
	hazard.ContractorID = contractorID

	if err := tx.Save(&hazard).Error; err != nil {
		tx.Rollback()
		return models.Hazard{}, err
	}
	// ========================
	// PARSE MULTIPLE CORRECTIVE ACTIONS
	// ========================
	followUpActions := r.MultipartForm.Value["followup_action[]"]
	departmentTerkaitIDs := r.MultipartForm.Value["department_terkait_id[]"]
	contractorTerkaitIDs := r.MultipartForm.Value["contractor_terkait_id[]"]
	picTerkaitIDs := r.MultipartForm.Value["pic_terkait_id[]"]
	dueDates := r.MultipartForm.Value["due_date[]"]
	completedOns := r.MultipartForm.Value["completed_on[]"]

	// ========================
	// DELETE OLD CORRECTIVE ACTIONS
	// ========================
	if err := tx.
		Where("hazard_id = ?", hazard.ID).
		Delete(&models.CorrectiveActionHazard{}).Error; err != nil {
		tx.Rollback()
		return models.Hazard{}, err
	}

	// ========================
	// CREATE MULTIPLE CORRECTIVE ACTIONS
	// ========================
	maxIndex := len(followUpActions)
	if len(departmentTerkaitIDs) > maxIndex {
		maxIndex = len(departmentTerkaitIDs)
	}
	if len(picTerkaitIDs) > maxIndex {
		maxIndex = len(picTerkaitIDs)
	}

	for i := 0; i < maxIndex; i++ {

		// Get values safely
		followUp := ""
		if i < len(followUpActions) {
			followUp = followUpActions[i]
		}

		// Skip if followup action is empty
		if followUp == "" {
			continue
		}

		// Parse dates
		var actionDueDate *time.Time
		if i < len(dueDates) && dueDates[i] != "" {
			if parsed, err := time.Parse("2006-01-02", dueDates[i]); err == nil {
				actionDueDate = &parsed
			}
		}

		var actionCompletedOn *time.Time
		if i < len(completedOns) && completedOns[i] != "" {
			if parsed, err := time.Parse("2006-01-02", completedOns[i]); err == nil {
				actionCompletedOn = &parsed
			}
		}

		// Parse department/contractor
		var actionDeptID *uint
		if i < len(departmentTerkaitIDs) && departmentTerkaitIDs[i] != "" {
			if id, err := parseUint(departmentTerkaitIDs[i]); err == nil {
				actionDeptID = &id
			}
		}

		var actionContractorID *uint
		if i < len(contractorTerkaitIDs) && contractorTerkaitIDs[i] != "" {
			if id, err := parseUint(contractorTerkaitIDs[i]); err == nil {
				actionContractorID = &id
			}
		}

		// Parse PIC
		var actionPicID *uint
		if i < len(picTerkaitIDs) && picTerkaitIDs[i] != "" {
			if id, err := parseUint(picTerkaitIDs[i]); err == nil {
				actionPicID = &id
			}
		}

		// Create new corrective action
		corrective := models.CorrectiveActionHazard{
			HazardID:            hazard.ID,
			FollowupAction:      followUp,
			DepartmentTerkaitID: actionDeptID,
			ContractorTerkaitID: actionContractorID,
			PicTerkaitID:        actionPicID,
			DueDate:             actionDueDate,
			CompletedOn:         actionCompletedOn,
		}

		if err := tx.Create(&corrective).Error; err != nil {
			tx.Rollback()
			return models.Hazard{}, err
		}
	}
	// ========================
	// STEP 3: SNAPSHOT AFTER
	// ========================
	after := toJSON(hazard)

	// ========================
	// STEP 4: SAVE AUDIT
	// ========================
	audit := models.HazardAudit{
		HazardID:  hazard.ID,
		Action:    "UPDATE",
		Before:    before,
		After:     after,
		ChangedBy: changedBy,
		ChangedAt: time.Now(),
	}

	if err := tx.Create(&audit).Error; err != nil {
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
	// DELETE FILES REQUESTED
	// ========================
	if len(deletedFiles) > 0 {

		for _, idStr := range deletedFiles {

			docID, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				continue
			}

			tx.
				Where("documentation_id = ? AND hazard_id = ?", docID, hazard.ID).
				Delete(&models.HazardDocumentation{})

			tx.Delete(&models.Documentation{}, docID)
		}
	}

	// ========================
	// NEW FILE UPLOAD
	// ========================
	descFiles, _ := s.uploadFiles(r, "dokumentasi_desc")
	correctiveFiles, _ := s.uploadFiles(r, "dokumentasi_corrective")

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
	// COMMIT
	// ========================
	if err := tx.Commit().Error; err != nil {
		return models.Hazard{}, err
	}

	return hazard, nil
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
