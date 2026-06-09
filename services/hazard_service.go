package services

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"gorm.io/gorm/clause"
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
type HazardEditData struct {
	Hazard               models.Hazard
	CanReopen            bool
	IsModerator          bool
	CorrectiveAction     models.CorrectiveActionHazard
	CorrectiveActionsDTO []CorrectiveActionDTO
	DescDocs             []FileDTO
	CorrectiveDocs       []FileDTO
	WorkType             string
	StatusArea           string

	// Master Data
	EventCategories []models.EventCategory
	Locations       []models.Location
	RiskMatrices    []models.RiskMatrix
	Departments     []models.Department
	Contractors     []models.Contractor
	Users           []models.User
	ScatOptions     []models.ScatOption
	Consequences    []models.RiskConsequence
	Likelihoods     []models.RiskLikelihood
	Assessments     []models.RiskAssessmentCode
	Matrices        []models.RiskMatrix
	AllTypes        []models.EventCategory
	AllPics         []models.User
}

type HazardDisplay struct {
	models.Hazard
	CanAccess bool
}

type HazardIndexResult struct {
	Hazards         []HazardDisplay
	TotalRows       int64
	TotalPages      int
	CurrentPage     int
	Categories      []models.EventCategory
	Locations       []models.Location
	ScatOptions     []models.ScatOption
	RiskAssessments []models.RiskAssessmentCode
}

var ErrUnauthorizedHazard = errors.New("anda tidak memiliki akses untuk mengubah laporan ini")

func (s *HazardService) GetHazards(
	currentUser models.User, // <-- 1. Tambahkan parameter currentUser di sini
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
		dbQuery = dbQuery.Where("DATE(tanggal_waktu) >= ?", startDate)
	}

	if endDate != "" {
		dbQuery = dbQuery.Where("DATE(tanggal_waktu) <= ?", endDate)
	}

	// =========================
	// CATEGORY FILTER
	// =========================
	if filterCategory != "" {
		dbQuery = dbQuery.Where("hazards.event_category_id = ?", filterCategory)
	}

	// =========================
	// LOCATION FILTER
	// =========================
	if filterLocation != "" {
		dbQuery = dbQuery.Where("hazards.location_id = ?", filterLocation)
	}

	// =========================
	// SCAT FILTER
	// =========================
	if filterScat != "" {
		dbQuery = dbQuery.Where("hazards.scat_option_id = ?", filterScat)
	}

	// =========================
	// RISK FILTER
	// =========================
	if filterRisk != "" {
		dbQuery = dbQuery.Joins(`
            LEFT JOIN risk_matrices
            ON risk_matrices.id = hazards.risk_matrix_id
        `).Where("risk_matrices.risk_assessment_id = ?", filterRisk)
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

	// ==========================================
	// 2. KONTROL AKSES (DIPROSES SETELAH FIND)
	// ==========================================
	var fullUser models.User
	if err := s.DB.Preload("ModeratedCategories").First(&fullUser, currentUser.ID).Error; err != nil {
		return HazardIndexResult{}, fmt.Errorf("gagal memverifikasi data user: %w", err)
	}

	var displayItems []HazardDisplay
	for _, item := range hazards {
		canAccess := false

		// A. Cek jika user adalah PIC
		if fullUser.ID == item.PicID {
			canAccess = true
		}

		// B. Cek jika user adalah Pelapor
		if item.ReportByID != nil && fullUser.ID == *item.ReportByID {
			canAccess = true
		}

		// C. Cek jika user adalah Moderator
		if isModerator, _ := s.checkPermissions(&item, fullUser); isModerator {
			canAccess = true
		}

		displayItems = append(displayItems, HazardDisplay{
			Hazard:    item,
			CanAccess: canAccess,
		})
	}

	// =========================
	// CATEGORIES OPTIONS
	// =========================
	var categories []models.EventCategory
	subQuery := s.DB.Model(&models.Hazard{}).Select("DISTINCT event_category_id")
	err = s.DB.
		Model(&models.EventCategory{}).
		Where("id IN (?)", subQuery).
		Order("name ASC").
		Find(&categories).Error

	if err != nil {
		return HazardIndexResult{}, err
	}

	// =========================
	// LOCATIONS OPTIONS
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
	// RISK ASSESSMENTS OPTIONS
	// =========================
	var riskAssessments []models.RiskAssessmentCode
	err = s.DB.Order("name asc").Find(&riskAssessments).Error
	if err != nil {
		return HazardIndexResult{}, err
	}

	// =========================
	// PAGINATION MATH
	// =========================
	totalPages := int((totalRows + int64(pageSize) - 1) / int64(pageSize))

	// 3. Kembalikan displayItems yang sudah berisi informasi CanAccess
	return HazardIndexResult{
		Hazards:         displayItems,
		TotalRows:       totalRows,
		TotalPages:      totalPages,
		CurrentPage:     page,
		Categories:      categories,
		Locations:       locations,
		ScatOptions:     scatOptions,
		RiskAssessments: riskAssessments,
	}, nil
}
func (s *HazardService) GetHazardEditData(hazardID string, currentUserID uint) (*HazardEditData, error) {
	var result HazardEditData
	var hazard models.Hazard

	// 1. GET HAZARD
	err := s.DB.
		Preload("EventCategory").
		Preload("RiskMatrix").
		Preload("ScatOption").
		Preload("Location").
		Preload("ReportBy").
		Preload("Department").
		Preload("Contractor").
		Preload("PIC").
		Preload("Documentations.Documentation").
		Preload("Audits").
		Preload("Audits.User").
		Preload("CorrectiveActions").
		Preload("CorrectiveActions.DepartmentTerkait").
		Preload("CorrectiveActions.ContractorTerkait").
		Preload("CorrectiveActions.PICTerkait").
		First(&hazard, hazardID).Error

	if err != nil {
		return nil, err
	}
	result.Hazard = hazard

	// ==========================================
	// 2. CHECK PERMISSION & POLICY GUARD (IDOR PROTECTION)
	// ==========================================
	var currentUser models.User
	if err := s.DB.Preload("ModeratedCategories").First(&currentUser, currentUserID).Error; err != nil {
		return nil, fmt.Errorf("gagal memverifikasi data user: %w", err)
	}

	result.CanReopen = false
	result.IsModerator = false
	canAccess := false // Flag penentu apakah user boleh mengedit halaman ini

	// A. Cek jika user adalah PIC
	if currentUser.ID == hazard.PicID {
		result.CanReopen = true
		canAccess = true
	}

	// B. Cek jika user adalah Pelapor (Reporter)
	if hazard.ReportByID != nil && currentUser.ID == *hazard.ReportByID {
		canAccess = true
		// result.CanReopen = true // Buka baris ini jika pelapor juga diizinkan reopen
	}

	// C. Cek jika user adalah Moderator
	for _, cat := range currentUser.ModeratedCategories {
		if hazard.EventCategory.ParentID != nil && cat.ID == *hazard.EventCategory.ParentID {
			result.CanReopen = true
			result.IsModerator = true
			canAccess = true
			break
		}
		if cat.ID == hazard.EventCategoryID {
			result.CanReopen = true
			result.IsModerator = true
			canAccess = true
			break
		}
	}

	// [CRUCIAL] KUNCI PINTU DI SINI: Jika tidak lolos salah satu syarat di atas, langsung TENDANG!
	if !canAccess {
		return nil, ErrUnauthorizedHazard
	}

	// ==========================================
	// 3. DOCUMENTATIONS (Hanya dieksekusi jika lolos guard)
	// ==========================================
	var descDocs, correctiveDocs []models.Documentation
	for _, item := range hazard.Documentations {
		switch item.DocType {
		case "desc":
			descDocs = append(descDocs, item.Documentation)
		case "corrective":
			correctiveDocs = append(correctiveDocs, item.Documentation)
		}
	}
	result.DescDocs = s.toFileDTO(descDocs)
	result.CorrectiveDocs = s.toFileDTO(correctiveDocs)

	// 4. DETERMINE WORK TYPE & LOAD ALL PICS
	result.WorkType = "department"
	if hazard.ContractorID != nil {
		result.WorkType = "contractor"
	}

	if result.WorkType == "department" && hazard.DepartmentID != nil {
		s.DB.Where("department_id = ? AND is_pic = ?", *hazard.DepartmentID, true).
			Order("name asc").Limit(50).Find(&result.AllPics)
	} else if result.WorkType == "contractor" && hazard.ContractorID != nil {
		s.DB.Where("contractor_id = ? AND is_pic = ?", *hazard.ContractorID, true).
			Order("name asc").Limit(50).Find(&result.AllPics)
	}

	// 5. CORRECTIVE ACTIONS DTO
	result.StatusArea = "aman"
	if len(hazard.CorrectiveActions) > 0 {
		result.StatusArea = "sementara"
		result.CorrectiveAction = hazard.CorrectiveActions[0]

		for _, ca := range hazard.CorrectiveActions {
			caType := "department"
			var pics []models.User

			if ca.ContractorTerkaitID != nil {
				caType = "contractor"
				s.DB.Where("contractor_id = ? AND is_pic = ?", *ca.ContractorTerkaitID, true).
					Order("name asc").Limit(50).Find(&pics)
			} else if ca.DepartmentTerkaitID != nil {
				s.DB.Where("department_id = ? AND is_pic = ?", *ca.DepartmentTerkaitID, true).
					Order("name asc").Limit(50).Find(&pics)
			}

			due, completed := "", ""
			if ca.DueDate != nil {
				due = ca.DueDate.Format("2006-01-02")
			}
			if ca.CompletedOn != nil {
				completed = ca.CompletedOn.Format("2006-01-02")
			}

			result.CorrectiveActionsDTO = append(result.CorrectiveActionsDTO, CorrectiveActionDTO{
				ID:                  ca.ID,
				FollowupAction:      ca.FollowupAction,
				Type:                caType,
				DepartmentTerkaitID: ca.DepartmentTerkaitID,
				ContractorTerkaitID: ca.ContractorTerkaitID,
				PicTerkaitID:        ca.PicTerkaitID,
				DueDate:             due,
				CompletedOn:         completed,
				Pics:                pics,
			})
		}
	}

	// 6. MASTER DATA QUERIES
	s.DB.Where("parent_id IS NULL AND category_group = ? AND code LIKE ?", "lead", "%HZD%").
		Order("name asc").Find(&result.EventCategories)

	s.DB.Order("risk_consequence_id asc").Find(&result.RiskMatrices)

	// Location
	if hazard.LocationID != 0 {
		locIDStr := strconv.FormatUint(uint64(hazard.LocationID), 10)
		s.DB.Where("id = ? OR id NOT IN (?)", hazard.LocationID, hazard.LocationID).
			Order("CASE WHEN id = " + locIDStr + " THEN 0 ELSE 1 END, name asc").
			Limit(50).Find(&result.Locations)
	} else {
		s.DB.Order("name asc").Limit(50).Find(&result.Locations)
	}

	// Department
	if hazard.DepartmentID != nil && *hazard.DepartmentID != 0 {
		deptIDStr := strconv.FormatUint(uint64(*hazard.DepartmentID), 10)
		s.DB.Where("id = ? OR id NOT IN (?)", *hazard.DepartmentID, *hazard.DepartmentID).
			Order("CASE WHEN id = " + deptIDStr + " THEN 0 ELSE 1 END, name asc").
			Limit(50).Find(&result.Departments)
	} else {
		s.DB.Order("name asc").Limit(50).Find(&result.Departments)
	}

	// Contractor
	if hazard.ContractorID != nil && *hazard.ContractorID != 0 {
		contIDStr := strconv.FormatUint(uint64(*hazard.ContractorID), 10)
		s.DB.Where("id = ? OR id NOT IN (?)", *hazard.ContractorID, *hazard.ContractorID).
			Order("CASE WHEN id = " + contIDStr + " THEN 0 ELSE 1 END, name asc").
			Limit(50).Find(&result.Contractors)
	} else {
		s.DB.Order("name asc").Limit(50).Find(&result.Contractors)
	}

	// User / ReportBy
	if hazard.ReportByID != nil && *hazard.ReportByID != 0 {
		userIDStr := strconv.FormatUint(uint64(*hazard.ReportByID), 10)
		s.DB.Where("id = ? OR id NOT IN (?)", *hazard.ReportByID, *hazard.ReportByID).
			Order("CASE WHEN id = " + userIDStr + " THEN 0 ELSE 1 END, name asc").
			Limit(50).Find(&result.Users)
	} else {
		s.DB.Order("name asc").Limit(50).Find(&result.Users)
	}

	s.DB.Where("type IN ?", []string{"unsafe_act", "personal_factor"}).
		Order("FIELD(type, 'unsafe_act', 'personal_factor'), code asc").Find(&result.ScatOptions)

	s.DB.Order("severity_level DESC").Find(&result.Consequences)
	s.DB.Order("sequence ASC").Find(&result.Likelihoods)
	s.DB.Find(&result.Assessments)

	s.DB.Preload("RiskConsequence").Preload("RiskLikelihood").Preload("RiskAssessment").
		Joins("JOIN risk_consequences ON risk_consequences.id = risk_matrices.risk_consequence_id").
		Joins("JOIN risk_likelihoods ON risk_likelihoods.id = risk_matrices.risk_likelihood_id").
		Joins("JOIN risk_assessment_codes ON risk_assessment_codes.id = risk_matrices.risk_assessment_id").
		Order("risk_consequences.severity_level DESC, risk_likelihoods.sequence ASC").Find(&result.Matrices)

	if hazard.EventCategory.ParentID != nil {
		s.DB.Where("parent_id = ?", *hazard.EventCategory.ParentID).
			Order("name asc").Find(&result.AllTypes)
	}

	return &result, nil
}

// Helper Internal Service
func (s *HazardService) toFileDTO(docs []models.Documentation) []FileDTO {
	var result []FileDTO
	for _, doc := range docs {
		if doc.FileURL == "" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(doc.FileURL))
		isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" || ext == ".gif"

		result = append(result, FileDTO{
			ID:      doc.ID,
			Name:    filepath.Base(doc.FileURL),
			URL:     doc.FileURL,
			IsImage: isImage,
		})
	}
	return result
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

// internal/services/hazard_service.go

// checkPermissions adalah helper untuk menentukan apakah user bisa reopen atau adalah seorang moderator.
// Fungsi ini mengembalikan (isModerator, canReopen)
// checkPermissions adalah helper untuk menentukan apakah user bisa reopen atau adalah seorang moderator.
// Mengembalikan: (isModerator, canReopen)
func (s *HazardService) checkPermissions(hazard *models.Hazard, currentUser models.User) (bool, bool) {
	canReopen := false
	isModerator := false

	// Proteksi 1: Pastikan user yang dicek valid (tidak kosong)
	if currentUser.ID == 0 {
		return false, false
	}

	// 1. Cek apakah user adalah PIC Hazard (Boleh Reopen, tapi BUKAN moderator)
	if currentUser.ID == hazard.PicID {
		canReopen = true
	}

	// 2. Cek apakah user adalah Moderator Kategori Hazard
	// Loop daftar kategori yang dimoderasi oleh user saat ini
	for _, cat := range currentUser.ModeratedCategories {

		// A. Cek jika user memoderasi 'Direct Category' (Sub-kategori yang langsung dipilih di hazard)
		if cat.ID == hazard.EventCategoryID {
			canReopen = true
			isModerator = true
			break // Berhenti looping karena status moderator sudah dikonfirmasi
		}

		// B. Cek jika user memoderasi 'Parent Category' (Kategori Utama)
		// Karena EventCategory adalah struct (bukan pointer di dalam Hazard), kita bisa langsung cek ParentID
		if hazard.EventCategory.ParentID != nil && cat.ID == *hazard.EventCategory.ParentID {
			canReopen = true
			isModerator = true
			break
		}
	}

	return isModerator, canReopen
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

	// =======================================================
	// 🔥 PROSES TRANSLASI OTOMATIS (Dilakukan sebelum transaksi DB)
	// =======================================================
	var descriptionEn string
	var correctiveActionEn string

	// Gunakan helper translate Anda (contoh: helpers.TranslateText)
	// Kita buat logic fallback: Jika API Google gagal, isi dengan teks asli agar tidak memblokir simpan data.
	if description != "" {
		if trans, err := helpers.TranslateText(description, "en"); err == nil {
			descriptionEn = trans
		} else {
			log.Printf("Warning: Gagal translate deskripsi: %v", err)
			descriptionEn = description
		}
	}

	if correctiveAction != "" {
		if trans, err := helpers.TranslateText(correctiveAction, "en"); err == nil {
			correctiveActionEn = trans
		} else {
			log.Printf("Warning: Gagal translate corrective action: %v", err)
			correctiveActionEn = correctiveAction
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

	ref, err := s.GenerateRefNumber(tx)
	if err != nil {
		tx.Rollback()
		return models.Hazard{}, err
	}
	// ========================
	// 6. CREATE HAZARD
	// ========================
	hazard := models.Hazard{
		RefNumber:          ref,
		EventCategoryID:    eventCategoryID,
		ScatOptionID:       scatOptionID,
		LocationID:         locationID,
		LocationSpecific:   locationSpecific,
		PicID:              picID,
		Deskripsi:          description,
		DeskripsiEn:        descriptionEn, // 🔥 Terisi Otomatis
		CorrectiveAction:   correctiveAction,
		CorrectiveActionEn: correctiveActionEn,
		TanggalWaktu:       eventDate,
		ReportByID:         reportByID,
		ReporterManual:     reportManual,
		RiskMatrixID:       riskMatrixID,
		DepartmentID:       departmentID,
		ContractorID:       contractorID,
		Status:             status,
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

	go s.sendUpdateNotification(hazard, false)
	return hazard, nil
}

func (s *HazardService) UpdateWithFiles(id uint, r *http.Request) (models.Hazard, error) {

	// ========================
	// PARSE MULTIPART
	// ========================
	if err := r.ParseMultipartForm(10 << 20); err != nil {
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

		return models.Hazard{}, fmt.Errorf("hazard lama tidak ditemukan: %w", err)
	}

	before := toJSON(buildHazardAuditData(oldHazard))

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
	workType := r.FormValue("work_type")

	picIDStr := r.FormValue("pic_id")

	description := r.FormValue("description")
	correctiveAction := r.FormValue("corrective_action")

	locationSpecific := r.FormValue("location_specific")

	reportByIDStr := r.FormValue("report_by_id")
	reportManual := r.FormValue("reporter_manual")

	eventDateStr := r.FormValue("event_date")
	riskMatrixIDStr := r.FormValue("risk_matrix_id")

	// Input Tambahan untuk Status Logic Baru
	isVerifiedForm := r.FormValue("is_verified") == "true"
	statusAreaForm := r.FormValue("status_area") // Berisi: "aman" atau "sementara"
	moderatorComment := r.FormValue("moderator_comment")
	deletedFiles := r.MultipartForm.Value["deleted_files[]"]

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
			return models.Hazard{}, fmt.Errorf("field %s wajib diisi", k)
		}
	}

	// ========================
	// PARSE DATA
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

	// ========================
	// OPTIONAL POINTERS
	// ========================
	fmt.Printf("departmentIDStr: %#v\n", departmentIDStr)
	fmt.Printf("contractorIDStr: %#v\n", contractorIDStr)

	isValidID := func(v string) bool {
		v = strings.ToLower(strings.TrimSpace(v))
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

	if reportManual == "" && isValidID(reportByIDStr) {
		id, err := parseUint(reportByIDStr)
		if err != nil {
			return models.Hazard{}, fmt.Errorf("report_by_id tidak valid")
		}
		reportByID = &id
		fmt.Printf("report_by_id dipanggil: %d\n", *reportByID)
	}

	// ========================
	// DEPARTMENT / CONTRACTOR
	// ========================
	var departmentID *uint
	var contractorID *uint

	departmentIDStr = strings.TrimSpace(departmentIDStr)
	contractorIDStr = strings.TrimSpace(contractorIDStr)

	if workType == "contractor" {
		departmentIDStr = ""
	} else if workType == "department" {
		contractorIDStr = ""
	}

	if isValidID(departmentIDStr) && isValidID(contractorIDStr) {
		return models.Hazard{}, fmt.Errorf("department dan contractor tidak boleh dipilih bersamaan")
	}

	if isValidID(departmentIDStr) {
		id, err := parseUint(departmentIDStr)
		if err != nil {
			return models.Hazard{}, fmt.Errorf("department_id tidak valid")
		}
		departmentID = &id
		contractorID = nil
	}

	if isValidID(contractorIDStr) {
		id, err := parseUint(contractorIDStr)
		if err != nil {
			return models.Hazard{}, fmt.Errorf("contractor_id tidak valid")
		}
		contractorID = &id
		departmentID = nil
	}

	// =======================================================
	// 🔥 OPTIMIZED TRANSLATION STATE (Dilakukan sebelum transaksi DB)
	// =======================================================
	var descriptionEn string
	var correctiveActionEn string
	var moderatorCommentEn string

	// 1. Cek Deskripsi
	if description != "" {
		if description == oldHazard.Deskripsi {
			descriptionEn = oldHazard.DeskripsiEn // Hemat API jika teks sama
		} else {
			if trans, err := helpers.TranslateText(description, "en"); err == nil {
				descriptionEn = trans
			} else {
				fmt.Printf("Warning: Gagal translate deskripsi: %v\n", err)
				descriptionEn = description
			}
		}
	}

	// 2. Cek Corrective Action
	if correctiveAction != "" {
		if correctiveAction == oldHazard.CorrectiveAction {
			correctiveActionEn = oldHazard.CorrectiveActionEn // Hemat API jika teks sama
		} else {
			if trans, err := helpers.TranslateText(correctiveAction, "en"); err == nil {
				correctiveActionEn = trans
			} else {
				fmt.Printf("Warning: Gagal translate corrective action: %v\n", err)
				correctiveActionEn = correctiveAction
			}
		}
	}

	// 3. Cek Komentar Moderator (Sekarang ikut ditranslate saat di-update)
	if moderatorComment != "" {
		if moderatorComment == oldHazard.ModeratorComment {
			moderatorCommentEn = oldHazard.ModeratorCommentEn // Hemat API jika teks sama
		} else {
			if trans, err := helpers.TranslateText(moderatorComment, "en"); err == nil {
				moderatorCommentEn = trans
			} else {
				fmt.Printf("Warning: Gagal translate moderator comment: %v\n", err)
				moderatorCommentEn = moderatorComment
			}
		}
	}

	// ========================
	// START TX
	// ========================
	tx := s.DB.Begin()
	if tx.Error != nil {
		return models.Hazard{}, tx.Error
	}

	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	// ========================
	// CORRECTIVE ACTIONS FORM ARRAYS
	// ========================
	followupActions := r.MultipartForm.Value["followup_action[]"]
	types := r.MultipartForm.Value["type[]"]
	departmentTerkaitIDs := r.MultipartForm.Value["department_terkait_id[]"]
	contractorTerkIDs := r.MultipartForm.Value["contractor_terkait_id[]"]
	picTerkIDs := r.MultipartForm.Value["pic_terkait_id[]"]
	dueDatesCorrective := r.MultipartForm.Value["due_date[]"]
	completedDatesCorrective := r.MultipartForm.Value["completed_on[]"]

	// ========================
	// STATUS LOGIC (SMART STATE MACHINE)
	// ========================
	var status models.HazardStatus

	hasAnyAction := false
	allCompleted := true

	// Evaluasi tindakan korektif jika status area membutuhkan penanganan lanjutan
	if statusAreaForm == "sementara" {
		for i := range followupActions {
			if strings.TrimSpace(followupActions[i]) == "" {
				continue
			}
			hasAnyAction = true

			// Jika ada tindakan tapi tanggal selesai kosong -> belum rampung
			if i >= len(completedDatesCorrective) || strings.TrimSpace(completedDatesCorrective[i]) == "" {
				allCompleted = false
			}
		}
	}

	// Penentuan Status Otomatis Berdasarkan Input Form
	if statusAreaForm == "aman" {
		// Bahaya langsung diselesaikan di tempat, otomatis masuk status menunggu verifikasi
		status = models.HazardStatusPending
	} else {
		// Jalur penanganan "sementara" (Corrective Action Berantai)
		if !hasAnyAction {
			status = models.HazardStatusSubmit
		} else if !allCompleted {
			status = models.HazardStatusInProgress
		} else {
			// Semua tindakan korektif sudah diisi 'Completed On' oleh PIC -> Naik ke Pending
			status = models.HazardStatusPending
		}
	}

	// Intervensi manual melalui tombol "Verifikasi & Close" di UI Auditor
	if isVerifiedForm && (statusAreaForm == "aman" || allCompleted) {
		status = models.HazardStatusClosed
	}

	// Safety Guard: Kunci status tetap Closed jika data diedit biasa tanpa alur re-open
	if oldHazard.Status == models.HazardStatusClosed && status != models.HazardStatusClosed {
		status = models.HazardStatusClosed
	}

	// ========================
	// DELETE OLD ACTIONS
	// ========================
	if err := tx.Where("hazard_id = ?", hazard.ID).Delete(&models.CorrectiveActionHazard{}).Error; err != nil {
		return models.Hazard{}, err
	}

	// ========================
	// UPDATE HAZARD MAP
	// ========================
	updateData := map[string]interface{}{
		"event_category_id":    eventCategoryID,
		"scat_option_id":       scatOptionID,
		"location_id":          locationID,
		"location_specific":    locationSpecific,
		"pic_id":               picID,
		"deskripsi":            description,
		"deskripsi_en":         descriptionEn, // 🔥 Terupdate otomatis
		"corrective_action":    correctiveAction,
		"corrective_action_en": correctiveActionEn, // 🔥 Terupdate otomatis
		"tanggal_waktu":        eventDate,
		"risk_matrix_id":       riskMatrixID,
		"status":               status, // Menggunakan status baru dari state machine
		"department_id":        departmentID,
		"contractor_id":        contractorID,
		"moderator_comment":    moderatorComment,
		"moderator_comment_en": moderatorCommentEn, // 🔥 Terupdate otomatis
	}

	// ========================
	// REPORTER LOGIC
	// ========================
	if strings.TrimSpace(reportManual) != "" {
		updateData["reporter_manual"] = reportManual
		updateData["report_by_id"] = nil
	} else {
		updateData["reporter_manual"] = ""
		updateData["report_by_id"] = reportByID
	}

	// ========================
	// UPDATE DATABASE
	// ========================
	if err := tx.Model(&models.Hazard{}).Where("id = ?", hazard.ID).Updates(updateData).Error; err != nil {
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
		if strings.TrimSpace(followupActions[i]) == "" {
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
		if actionType == "department" && i < len(departmentTerkaitIDs) {
			idStr := strings.TrimSpace(departmentTerkaitIDs[i])
			if idStr != "" {
				id, err := parseUint(idStr)
				if err != nil {
					return models.Hazard{}, fmt.Errorf("department terkait tidak valid")
				}
				action.DepartmentTerkaitID = &id
			}
		}

		// CONTRACTOR
		if actionType == "contractor" && i < len(contractorTerkIDs) {
			idStr := strings.TrimSpace(contractorTerkIDs[i])
			if idStr != "" {
				id, err := parseUint(idStr)
				if err != nil {
					return models.Hazard{}, fmt.Errorf("contractor terkait tidak valid")
				}
				action.ContractorTerkaitID = &id
			}
		}

		// PIC
		if i < len(picTerkIDs) {
			idStr := strings.TrimSpace(picTerkIDs[i])
			if idStr != "" {
				id, err := parseUint(idStr)
				if err != nil {
					return models.Hazard{}, fmt.Errorf("pic terkait tidak valid")
				}
				action.PicTerkaitID = &id
			}
		}

		// DUE DATE
		if i < len(dueDatesCorrective) {
			dateStr := strings.TrimSpace(dueDatesCorrective[i])
			if dateStr != "" {
				t, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					return models.Hazard{}, fmt.Errorf("due_date tidak valid")
				}
				action.DueDate = &t
			}
		}

		// COMPLETED ON
		if i < len(completedDatesCorrective) {
			dateStr := strings.TrimSpace(completedDatesCorrective[i])
			if dateStr != "" {
				t, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					return models.Hazard{}, fmt.Errorf("completed_on tidak valid")
				}
				action.CompletedOn = &t
			}
		}

		if err := tx.Create(&action).Error; err != nil {
			return models.Hazard{}, err
		}

		fmt.Printf("SUCCESS INSERT ACTION ID: %d\n", action.ID)
	}

	// ========================
	// DELETE FILES
	// ========================
	var filesToDeletePhysical []string

	if len(deletedFiles) > 0 {
		for _, idStr := range deletedFiles {
			docID, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				continue
			}

			var doc models.Documentation
			if err := tx.First(&doc, docID).Error; err == nil {
				filesToDeletePhysical = append(filesToDeletePhysical, doc.FileURL)
			}

			if err := tx.Where("documentation_id = ? AND hazard_id = ?", docID, hazard.ID).Delete(&models.HazardDocumentation{}).Error; err != nil {
				return models.Hazard{}, err
			}

			if err := tx.Delete(&models.Documentation{}, docID).Error; err != nil {
				return models.Hazard{}, err
			}
		}
	}

	// ========================
	// UPLOAD FILES
	// ========================
	descFiles, err := s.uploadFiles(r, "dokumentasi_desc")
	if err != nil {
		return models.Hazard{}, err
	}

	correctiveFiles, err := s.uploadFiles(r, "dokumentasi_corrective")
	if err != nil {
		return models.Hazard{}, err
	}

	// ========================
	// SAVE DOCUMENTS
	// ========================
	saveDocs := func(files []string, docType string) error {
		for _, fileURL := range files {
			doc := models.Documentation{FileURL: fileURL}
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
		return models.Hazard{}, err
	}

	if err := saveDocs(correctiveFiles, "corrective"); err != nil {
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
		First(&updatedHazard, hazard.ID).Error; err != nil {
		return models.Hazard{}, err
	}

	after := toJSON(buildHazardAuditData(updatedHazard))

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

	if err := tx.Create(&audit).Error; err != nil {
		return models.Hazard{}, err
	}

	// ========================
	// COMMIT TRANSACTION
	// ========================
	if err := tx.Commit().Error; err != nil {
		return models.Hazard{}, err
	}

	committed = true

	// ========================
	// HAPUS FILE FISIK (SETELAH COMMIT SUKSES)
	// ========================
	go func(files []string) {
		for _, fileURL := range files {
			deletePhysicalFile(fileURL)
		}
	}(filesToDeletePhysical)

	// ========================
	// SEND EMAIL (DETEKSI PERUBAHAN STATUS)
	// ========================
	go s.sendUpdateNotification(updatedHazard, true)
	return updatedHazard, nil
}

func (s *HazardService) DeleteHazard(id string, currentUser models.User) error {
	var hazard models.Hazard

	// 1. Load data hazard beserta EventCategory
	if err := s.DB.Preload("EventCategory").First(&hazard, id).Error; err != nil {
		return err
	}

	// 2. Load data user beserta ModeratedCategories
	var fullUser models.User
	if err := s.DB.Preload("ModeratedCategories").First(&fullUser, currentUser.ID).Error; err != nil {
		return fmt.Errorf("gagal memverifikasi data user")
	}

	// Cek Izin
	isModerator, _ := s.checkPermissions(&hazard, fullUser)
	if fullUser.ID != hazard.PicID && !isModerator {
		return fmt.Errorf("anda tidak memiliki izin untuk menghapus hazard ini")
	}

	// ==========================================
	// 3. EKSEKUSI HAPUS (PERBAIKAN DI SINI)
	// ==========================================
	// Gunakan Select(clause.Associations) agar GORM menghapus data di tabel:
	// hazard_audits, hazard_documentations, corrective_action_hazards,
	// dan menghapus relasi many-to-many di hazard_scat_options.
	if err := s.DB.Unscoped().Select(clause.Associations).Delete(&hazard).Error; err != nil {
		return err
	}

	return nil
}

func (s *HazardService) GetHazardsWithAccessControl(currentUser models.User) ([]HazardDisplay, error) {
	var hazards []models.Hazard

	// 1. Ambil semua data hazard beserta relasi pendukungnya
	if err := s.DB.Preload("EventCategory").Preload("ReportBy").Find(&hazards).Error; err != nil {
		return nil, err
	}

	// 2. Load ulang user dari DB untuk memastikan ModeratedCategories ikut terbawa
	var fullUser models.User
	if err := s.DB.Preload("ModeratedCategories").First(&fullUser, currentUser.ID).Error; err != nil {
		return nil, fmt.Errorf("gagal memverifikasi data user")
	}

	var displayItems []HazardDisplay

	// 3. Iterasi dan hitung izin akses untuk setiap item hazard
	for _, item := range hazards {
		canAccess := false

		// A. Cek jika user adalah PIC
		if fullUser.ID == item.PicID {
			canAccess = true
		}

		// B. Cek jika user adalah Pelapor
		if item.ReportByID != nil && fullUser.ID == *item.ReportByID {
			canAccess = true
		}

		// C. Cek jika user adalah Moderator (bisa langsung panggil unexported method)
		isModerator, _ := s.checkPermissions(&item, fullUser)
		if isModerator {
			canAccess = true
		}

		// Bungkus ke dalam struct baru
		displayItems = append(displayItems, HazardDisplay{
			Hazard:    item,
			CanAccess: canAccess,
		})
	}

	return displayItems, nil
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

func (s *HazardService) sendUpdateNotification(h models.Hazard, isUpdate bool) {
	// 1. Defer recover untuk mencegah aplikasi crash
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic email:", r)
		}
	}()

	// ==========================================================
	// RE-FETCH DATA (Mendapatkan relasi lengkap)
	// ==========================================================
	var fullHazard models.Hazard
	err := s.DB.
		Preload("EventCategory").
		Preload("RiskMatrix.RiskAssessment").
		Preload("ReportBy").
		Preload("Location").
		First(&fullHazard, h.ID).Error

	if err != nil {
		fmt.Println("Gagal memuat data lengkap untuk email:", err)
		return
	}

	// 2. Ambil Email
	picEmail, err := helpers.GetPICEmail(s.DB, fullHazard.PicID)
	if err != nil {
		fmt.Println("gagal ambil pic email:", err)
	}

	moderatorEmails, err := helpers.GetModeratorEmails(s.DB, fullHazard.EventCategoryID)
	if err != nil {
		fmt.Println("gagal ambil moderator email:", err)
	}

	// 3. Persiapkan Data
	namaPelapor := "Sistem / Anonim"
	if fullHazard.ReportBy != nil {
		namaPelapor = fullHazard.ReportBy.Name
	}

	locationName := "-"
	if fullHazard.Location.Name != "" {
		locationName = fullHazard.Location.Name
	}

	categoryName := "-"
	if fullHazard.EventCategory.Name != "" {
		categoryName = fullHazard.EventCategory.Name
	}

	riskLevel := "-"
	if fullHazard.RiskMatrix.RiskAssessment.Name != "" {
		riskLevel = fullHazard.RiskMatrix.RiskAssessment.Name
	}

	hazardURL := fmt.Sprintf("%s/hazard/edit/%d", config.AppURL(), fullHazard.ID)

	// 4. Generate HTML Table
	tableContent := fmt.Sprintf(`
        <table border="1" cellpadding="8" cellspacing="0" style="border-collapse: collapse; width: 100%%;">
            <tr><td><strong>Kategori</strong></td><td>%s</td></tr>
            <tr><td><strong>Lokasi</strong></td><td>%s</td></tr>
            <tr><td><strong>Deskripsi</strong></td><td>%s</td></tr>
            <tr><td><strong>Risk</strong></td><td>%s</td></tr>
            <tr><td><strong>Pelapor</strong></td><td>%s</td></tr>
            <tr><td><strong>Status</strong></td><td>%s</td></tr>
        </table>
        <p style="margin-top:20px;">
            <a href="%s" style="background:#2563eb; color:white; padding:12px 20px; text-decoration:none; border-radius:6px; display:inline-block; font-weight:bold;">
                Buka Laporan Hazard
            </a>
        </p>`,
		categoryName, locationName, fullHazard.Deskripsi, riskLevel, namaPelapor, fullHazard.Status, hazardURL,
	)

	// 5. Logic Greeting Dinamis
	moderatorGreeting := "Halo Moderator HSE, Terdapat hazard baru pada kategori yang Anda moderasi:"
	if isUpdate {
		moderatorGreeting = "Halo Moderator HSE, Terdapat update hazard pada kategori yang Anda moderasi:"
	}

	picGreeting := "Halo PIC, Anda ditunjuk sebagai PIC hazard baru berikut:"
	if isUpdate {
		picGreeting = "Halo PIC, Terdapat update pada hazard yang Anda tangani:"
	}

	// 6. Fungsi Helper untuk Kirim Email (IMPLEMENTASI LENGKAP)
	sendMail := func(to []string, subject, greeting string) {
		if len(to) == 0 || (len(to) == 1 && to[0] == "") {
			return
		}

		// tableContent digunakan di sini, sehingga error 'declared and not used' hilang
		fullHTML := fmt.Sprintf("<p>%s</p> %s", greeting, tableContent)

		err := config.SendEmail(
			to,
			subject,
			config.EmailTemplate("SENTRY Hazard System", "Notifikasi Hazard", fullHTML),
		)

		if err != nil {
			fmt.Printf("gagal kirim email ke %v: %v\n", to, err)
		}
	}

	// 7. Eksekusi Pengiriman
	if picEmail != "" {
		sendMail([]string{picEmail}, fmt.Sprintf("SENTRY PIC Hazard #%d", fullHazard.ID), picGreeting)
	}

	if len(moderatorEmails) > 0 {
		sendMail(moderatorEmails, fmt.Sprintf("SENTRY Moderator Hazard #%d", fullHazard.ID), moderatorGreeting)
	}
}

// Pastikan parameter menerima *gorm.DB agar bisa menerima 'tx' maupun 's.DB'
func (s *HazardService) GenerateRefNumber(db *gorm.DB) (string, error) {
	today := time.Now().Format("20060102")
	prefix := "HZ-" + today + "-"

	var lastHazard models.Hazard

	// Tambahkan .Clauses tepat di sini untuk mengunci baris data
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("ref_number LIKE ?", prefix+"%").
		Order("ref_number DESC").
		First(&lastHazard).
		Error

	// Penanganan error tetap sama
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	var refNumber string
	if lastHazard.ID == 0 {
		refNumber = prefix + "0001"
	} else {
		// Logika increment tetap sama
		lastNumber := lastHazard.RefNumber[len(prefix):]
		lastNumberInt, err := strconv.Atoi(lastNumber)
		if err != nil {
			return "", err
		}
		refNumber = fmt.Sprintf("%s%04d", prefix, lastNumberInt+1)
	}

	return refNumber, nil
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
