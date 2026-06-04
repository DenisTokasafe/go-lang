package controllers

import (
	// Sesuaikan dengan nama module di go.mod Anda
	"encoding/json"
	"fmt"
	"html/template"
	"latihan1/middlewares"
	"latihan1/models"
	"latihan1/services"
	"latihan1/utils"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type HazardController struct {
	DB      *gorm.DB
	Service *services.HazardService
	Render  func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (hc *HazardController) Index(w http.ResponseWriter, r *http.Request) {

	search := strings.TrimSpace(
		r.URL.Query().Get("search"),
	)

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	filterCategory := r.URL.Query().Get("category")
	filterLocation := r.URL.Query().Get("location")
	filterRisk := r.URL.Query().Get("risk")
	filterScat := r.URL.Query().Get("scat")

	page, err := strconv.Atoi(
		r.URL.Query().Get("page"),
	)

	if err != nil || page < 1 {
		page = 1
	}

	// =========================
	// GET DATA FROM SERVICE
	// =========================
	result, err := hc.Service.GetHazards(
		search,
		startDate,
		endDate,
		filterCategory,
		filterLocation,
		filterRisk,
		filterScat,
		page,
	)

	if err != nil {

		log.Println(
			"GET HAZARDS ERROR:",
			err,
		)

		http.Error(
			w,
			"Failed load hazards",
			http.StatusInternalServerError,
		)

		return
	}

	// =========================
	// VIEW DATA
	// =========================
	data := map[string]interface{}{
		"Title": "Hazard Reports",

		"Hazards": result.Hazards,

		"Search":         search,
		"FilterCategory": filterCategory,
		"FilterLocation": filterLocation,
		"FilterRisk":     filterRisk,
		"FilterScat":     filterScat,

		"Categories":      result.Categories,
		"ScatOptions":     result.ScatOptions,
		"RiskAssessments": result.RiskAssessments,
		"Location":        result.Locations,

		"StartDate": startDate,
		"EndDate":   endDate,

		"CurrentPage": result.CurrentPage,
		"TotalPages":  result.TotalPages,
		"TotalRows":   result.TotalRows,
	}

	hc.Render(
		w,
		r,
		"/hazard_report/index.gohtml",
		data,
	)
}
func (hc *HazardController) Create(w http.ResponseWriter, r *http.Request) {

	// Ambil stringnya dulu
	rawID := r.FormValue("parent_id")

	// Baru konversi ke angka (uint)
	eventCategoryID, _ := strconv.ParseUint(rawID, 10, 32)

	// Sekarang fmt.Printf akan memunculkan Type: uint64
	fmt.Printf("Value: %v | Type: %T\n", eventCategoryID, eventCategoryID)

	// Di dalam function yang merender form (misal: Create atau Edit)
	var eventCategories []models.EventCategory
	var locations []models.Location
	var riskMatrices []models.RiskMatrix
	var departments []models.Department
	var contractors []models.Contractor
	var users []models.User
	var scatOptions []models.ScatOption
	var consequences []models.RiskConsequence
	var likelihoods []models.RiskLikelihood
	var assessments []models.RiskAssessmentCode

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize := 25 // Biasanya matrix butuh view lebih banyak (5x5 = 25)
	offset := (page - 1) * pageSize

	var matrices []models.RiskMatrix
	var totalRows int64

	// Ambil semua data referensi
	hc.DB.Order("name asc").Find(&eventCategories)
	hc.DB.Order("name asc").Limit(50).Find(&locations)
	hc.DB.Order("risk_consequence_id asc").Limit(50).Find(&riskMatrices)
	hc.DB.Order("name asc").Limit(50).Find(&departments)
	hc.DB.Order("name asc").Limit(50).Find(&contractors)
	hc.DB.Order("name asc").Limit(50).Find(&users)
	hc.DB.
		Where("type IN ?", []string{"unsafe_act", "personal_factor"}).
		Order("FIELD(type, 'unsafe_act', 'personal_factor'), code asc").
		Find(&scatOptions)
	hc.DB.Where("parent_id IS NULL AND category_group = ? AND code LIKE ?", "lead", "%HZD%").
		Order("name asc").Find(&eventCategories)
	hc.DB.Order("severity_level DESC").Find(&consequences)
	hc.DB.Order("sequence ASC").Find(&likelihoods)
	hc.DB.Find(&assessments)
	// 3. Filter Search (Bisa cari berdasarkan nama consequence atau likelihood)
	dbQuery := hc.DB.Model(&models.RiskMatrix{}).
		Preload("RiskConsequence").
		Preload("RiskLikelihood").
		Preload("RiskAssessment").
		Joins("JOIN risk_consequences ON risk_consequences.id = risk_matrices.risk_consequence_id").
		Joins("JOIN risk_likelihoods ON risk_likelihoods.id = risk_matrices.risk_likelihood_id").
		Joins("JOIN risk_assessment_codes ON risk_assessment_codes.id = risk_matrices.risk_assessment_id")

	// 3. Filter Search (Bisa cari berdasarkan nama consequence atau likelihood)

	// 4. Hitung Total & Ambil Data
	dbQuery.Count(&totalRows)
	dbQuery.Limit(pageSize).Offset(offset).
		Order("risk_consequences.severity_level DESC, risk_likelihoods.sequence ASC").
		Find(&matrices)
	totalPages := int((totalRows + int64(pageSize) - 1) / int64(pageSize))
	// 2. Ambil SEMUA Sub-Kategori (Anak dari kategori HZD)
	// Kita ambil semua dulu, nanti Alpine.js yang memfilter berdasarkan parent_id

	data := map[string]interface{}{
		"Title":           "Tambah Laporan Hazard",
		"Matrices":        matrices,
		"Consequences":    consequences, // Untuk header kolom grid
		"Likelihoods":     likelihoods,  // Untuk row baris grid
		"Assessments":     assessments,  // Untuk pilihan di modal
		"EventCategories": eventCategories,
		"Locations":       locations,
		"RiskMatrices":    riskMatrices,
		"Departments":     departments,
		"Contractors":     contractors,
		"Users":           users, // Untuk Report By (Internal)
		"ScatOptions":     scatOptions,
		"TotalRows":       totalRows,
		"TotalPages":      totalPages,                            // Untuk pilihan Unsafe Act/Personal Factor
		"CurrentTime":     time.Now().Format("2006-01-02T15:04"), // Default input datetime-local
	}

	hc.Render(w, r, "/hazard_report/create.gohtml", data)
}
func (hc *HazardController) Store(w http.ResponseWriter, r *http.Request) {
	if hc.Service == nil {
		http.Error(w, "hazard service is nil", http.StatusInternalServerError)
		return
	}
	hazard, err := hc.Service.CreateWithFiles(r)
	if err != nil {
		log.Println("Store error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hc.setFlash(w, "Data Hazard berhasil dibuat!!!", "success")
	http.Redirect(w, r, "/hazard/edit/"+strconv.Itoa(int(hazard.ID)), http.StatusSeeOther)
}
func (hc *HazardController) Edit(w http.ResponseWriter, r *http.Request) {

	id := r.PathValue("id")

	var hazard models.Hazard

	// =========================
	// GET HAZARD
	// =========================
	err := hc.DB.
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
		First(&hazard, id).Error

	if err != nil {
		http.NotFound(w, r)
		return
	}

	// =========================
	// DTO + JSON HELPER (ADD HERE)
	// =========================

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

		DepartmentTerkaitID *uint         `json:"department_terkait_id"`
		ContractorTerkaitID *uint         `json:"contractor_terkait_id"`
		PicTerkaitID        *uint         `json:"pic_terkait_id"`
		Pics                []models.User `json:"pics"`
		DueDate             string        `json:"due_date"`
		CompletedOn         string        `json:"completed_on"`
	}

	toFileDTO := func(docs []models.Documentation) []FileDTO {
		var result []FileDTO

		for _, doc := range docs {

			if doc.FileURL == "" {
				continue
			}

			ext := strings.ToLower(filepath.Ext(doc.FileURL))

			isImage := ext == ".jpg" ||
				ext == ".jpeg" ||
				ext == ".png" ||
				ext == ".webp" ||
				ext == ".gif"

			result = append(result, FileDTO{
				ID:      doc.ID,
				Name:    filepath.Base(doc.FileURL),
				URL:     doc.FileURL,
				IsImage: isImage,
			})
		}

		return result
	}

	toJSON := func(v interface{}) template.JS {
		b, _ := json.Marshal(v)
		return template.JS(b)
	}

	// =========================
	// MASTER DATA
	// =========================
	var eventCategories []models.EventCategory
	var locations []models.Location
	var riskMatrices []models.RiskMatrix
	var departments []models.Department
	var contractors []models.Contractor
	var users []models.User
	var scatOptions []models.ScatOption
	var consequences []models.RiskConsequence
	var likelihoods []models.RiskLikelihood
	var assessments []models.RiskAssessmentCode
	var matrices []models.RiskMatrix

	hc.DB.
		Where("parent_id IS NULL AND category_group = ? AND code LIKE ?", "lead", "%HZD%").
		Order("name asc").
		Find(&eventCategories)

	hc.DB.
		Order("risk_consequence_id asc").
		Find(&riskMatrices)
	if hazard.LocationID != 0 {
		// Konversi uint ke string
		locIDStr := strconv.FormatUint(uint64(hazard.LocationID), 10)

		hc.DB.Where("id = ? OR id NOT IN (?)", hazard.LocationID, hazard.LocationID).
			Order("CASE WHEN id = " + locIDStr + " THEN 0 ELSE 1 END, name asc").
			Limit(50).
			Find(&locations)
	} else {
		hc.DB.Order("name asc").Limit(50).Find(&locations)
	}
	// ==========================================
	// 2. DEPARTMENTS (Menggunakan Pointer *uint)
	// ==========================================
	if hazard.DepartmentID != nil && *hazard.DepartmentID != 0 {
		// Ambil nilai asli dari pointer menggunakan *
		deptIDStr := strconv.FormatUint(uint64(*hazard.DepartmentID), 10)

		hc.DB.Where("id = ? OR id NOT IN (?)", *hazard.DepartmentID, *hazard.DepartmentID).
			Order("CASE WHEN id = " + deptIDStr + " THEN 0 ELSE 1 END, name asc").
			Limit(50).
			Find(&departments)
	} else {
		hc.DB.Order("name asc").Limit(50).Find(&departments)
	}

	// ==========================================
	// 3. CONTRACTORS (Menggunakan Pointer *uint)
	// ==========================================
	if hazard.ContractorID != nil && *hazard.ContractorID != 0 {
		// Ambil nilai asli dari pointer menggunakan *
		contIDStr := strconv.FormatUint(uint64(*hazard.ContractorID), 10)

		hc.DB.Where("id = ? OR id NOT IN (?)", *hazard.ContractorID, *hazard.ContractorID).
			Order("CASE WHEN id = " + contIDStr + " THEN 0 ELSE 1 END, name asc").
			Limit(50).
			Find(&contractors)
	} else {
		hc.DB.Order("name asc").Limit(50).Find(&contractors)
	}

	// ==========================================
	// 4. USERS / ReportBy (Menggunakan Pointer *uint)
	// ==========================================
	if hazard.ReportByID != nil && *hazard.ReportByID != 0 {
		// Ambil nilai asli dari pointer menggunakan *
		userIDStr := strconv.FormatUint(uint64(*hazard.ReportByID), 10)

		hc.DB.Where("id = ? OR id NOT IN (?)", *hazard.ReportByID, *hazard.ReportByID).
			Order("CASE WHEN id = " + userIDStr + " THEN 0 ELSE 1 END, name asc").
			Limit(50).
			Find(&users)
	} else {
		hc.DB.Order("name asc").Limit(50).Find(&users)
	}

	hc.DB.
		Where("type IN ?", []string{
			"unsafe_act",
			"personal_factor",
		}).
		Order("FIELD(type, 'unsafe_act', 'personal_factor'), code asc").
		Find(&scatOptions)

	hc.DB.
		Order("severity_level DESC").
		Find(&consequences)

	hc.DB.
		Order("sequence ASC").
		Find(&likelihoods)

	hc.DB.Find(&assessments)

	hc.DB.
		Preload("RiskConsequence").
		Preload("RiskLikelihood").
		Preload("RiskAssessment").
		Joins("JOIN risk_consequences ON risk_consequences.id = risk_matrices.risk_consequence_id").
		Joins("JOIN risk_likelihoods ON risk_likelihoods.id = risk_matrices.risk_likelihood_id").
		Joins("JOIN risk_assessment_codes ON risk_assessment_codes.id = risk_matrices.risk_assessment_id").
		Order("risk_consequences.severity_level DESC, risk_likelihoods.sequence ASC").
		Find(&matrices)

	// =========================
	// LOAD EVENT TYPES
	// =========================
	var allTypes []models.EventCategory

	if hazard.EventCategory.ParentID != nil {

		hc.DB.
			Where("parent_id = ?", *hazard.EventCategory.ParentID).
			Order("name asc").
			Find(&allTypes)

	}

	// =========================
	// LOAD PICS
	// =========================
	var allPics []models.User

	// =========================
	// LOAD DOCUMENTATIONS
	// =========================
	var descDocs []models.Documentation
	var correctiveDocs []models.Documentation

	for _, item := range hazard.Documentations {

		switch item.DocType {

		case "desc":
			descDocs = append(descDocs, item.Documentation)

		case "corrective":
			correctiveDocs = append(correctiveDocs, item.Documentation)
		}
	}

	// =========================
	// DETERMINE WORK TYPE
	// =========================
	workType := "department"

	if hazard.ContractorID != nil {
		workType = "contractor"
	}
	if workType == "department" && hazard.DepartmentID != nil {

		hc.DB.
			Where(
				"department_id = ? AND is_pic = ?",
				*hazard.DepartmentID,
				true,
			).
			Order("name asc").Limit(50).
			Find(&allPics)

	}

	if workType == "contractor" && hazard.ContractorID != nil {

		hc.DB.
			Where(
				"contractor_id = ? AND is_pic = ?",
				*hazard.ContractorID,
				true,
			).
			Order("name asc").Limit(50).
			Find(&allPics)

	}

	// =========================
	// CORRECTIVE ACTION DATA (support multiple)
	// =========================
	var correctiveAction models.CorrectiveActionHazard

	statusArea := "aman"

	var correctiveActionsDTO []CorrectiveActionDTO

	if len(hazard.CorrectiveActions) > 0 {

		statusArea = "sementara"

		// keep compatibility: first corrective action
		correctiveAction = hazard.CorrectiveActions[0]

		// build DTOs for all corrective actions
		for _, ca := range hazard.CorrectiveActions {

			caType := "department"

			var pics []models.User

			if ca.ContractorTerkaitID != nil {

				caType = "contractor"

				hc.DB.
					Where(
						"contractor_id = ? AND is_pic = ?",
						*ca.ContractorTerkaitID,
						true,
					).
					Order("name asc").Limit(50).
					Find(&pics)

			}

			if ca.DepartmentTerkaitID != nil {

				hc.DB.
					Where(
						"department_id = ? AND is_pic = ?",
						*ca.DepartmentTerkaitID,
						true,
					).
					Order("name asc").Limit(50).
					Find(&pics)

			}

			due := ""
			completed := ""

			if ca.DueDate != nil {
				due = ca.DueDate.Format("2006-01-02")
			}

			if ca.CompletedOn != nil {
				completed = ca.CompletedOn.Format("2006-01-02")
			}

			correctiveActionsDTO = append(
				correctiveActionsDTO,
				CorrectiveActionDTO{
					ID:             ca.ID,
					FollowupAction: ca.FollowupAction,
					Type:           caType,

					DepartmentTerkaitID: ca.DepartmentTerkaitID,
					ContractorTerkaitID: ca.ContractorTerkaitID,
					PicTerkaitID:        ca.PicTerkaitID,

					DueDate:     due,
					CompletedOn: completed,

					Pics: pics,
				},
			)
		}

	}

	// =========================
	// CURRENT USER
	// =========================
	userRaw, ok := r.Context().
		Value(middlewares.AuthUserKey).(models.User)

	if !ok {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// reload lengkap relasi user
	var currentUser models.User

	hc.DB.
		Preload("ModeratedCategories").
		First(&currentUser, userRaw.ID)

	// =========================
	// CHECK PERMISSION REOPEN
	// =========================
	canReopen := false

	// PIC hazard
	if currentUser.ID == hazard.PicID {
		canReopen = true
	}

	// Moderator kategori hazard
	for _, cat := range currentUser.ModeratedCategories {

		// parent category moderator
		if hazard.EventCategory.ParentID != nil {

			if cat.ID == *hazard.EventCategory.ParentID {
				canReopen = true
				break
			}
		}

		// direct category moderator
		if cat.ID == hazard.EventCategoryID {
			canReopen = true
			break
		}
	}

	// =========================
	// VIEW DATA
	// =========================
	data := map[string]interface{}{
		"Title":                "Edit Hazard",
		"CanReopen":            canReopen,
		"Hazard":               hazard,
		"CorrectiveAction":     correctiveAction,
		"CorrectiveActions":    toJSON(correctiveActionsDTO),
		"CorrectiveActionsRaw": hazard.CorrectiveActions,
		"Matrices":             matrices,
		"Consequences":         consequences,
		"Likelihoods":          likelihoods,
		"Assessments":          assessments,
		"Audits":               hazard.Audits,
		"EventCategories":      eventCategories,
		"Locations":            locations,
		"RiskMatrices":         riskMatrices,
		"Departments":          departments,
		"Contractors":          contractors,
		"Users":                users,
		"ScatOptions":          scatOptions,

		// dynamic select
		"AllTypes": allTypes,
		"AllPics":  allPics,
		// =========================
		// FIXED: JSON SAFE FOR ALPINE
		// =========================
		"DescDocs":       toJSON(toFileDTO(descDocs)),
		"CorrectiveDocs": toJSON(toFileDTO(correctiveDocs)),

		"WorkType":   workType,
		"StatusArea": statusArea,
	}

	hc.Render(
		w,
		r,
		"/hazard_report/edit.gohtml",
		data,
	)
}
func (hc *HazardController) Update(w http.ResponseWriter, r *http.Request) {

	// ========================
	// VALIDASI METHOD
	// ========================
	if r.Method != http.MethodPost {

		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)

		return
	}

	// ========================
	// VALIDASI SERVICE
	// ========================
	if hc.Service == nil {

		http.Error(
			w,
			"service nil",
			http.StatusInternalServerError,
		)

		return
	}

	// ========================
	// GET PARAM ID
	// ========================
	idStr := r.PathValue("id")

	id, err := strconv.ParseUint(
		idStr,
		10,
		64,
	)

	if err != nil {

		http.Error(
			w,
			"invalid id",
			http.StatusBadRequest,
		)

		return
	}

	// ========================
	// UPDATE DATA
	// ========================
	hazard, err := hc.Service.UpdateWithFiles(
		uint(id),
		r,
	)

	if err != nil {

		log.Printf(
			"Hazard Update Error ID %s: %v\n",
			idStr,
			err,
		)

		hc.setFlash(
			w,
			err.Error(),
			"error",
		)

		http.Redirect(
			w,
			r,
			"/hazard/edit/"+idStr,
			http.StatusSeeOther,
		)

		return
	}

	// ========================
	// SUCCESS
	// ========================
	hc.setFlash(
		w,
		"Data Hazard berhasil diupdate!",
		"success",
	)

	http.Redirect(
		w,
		r,
		"/hazard/edit/"+strconv.Itoa(
			int(hazard.ID),
		),
		http.StatusSeeOther,
	)
}
func (hc *HazardController) UpdateStatus(w http.ResponseWriter, r *http.Request) {

	id := r.PathValue("id")

	status := r.FormValue("status")

	err := hc.DB.
		Model(&models.Hazard{}).
		Where("id = ?", id).
		UpdateColumn("status", status).Error

	if err != nil {

		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)

		return
	}
	hc.setFlash(w, "Laporan Hazard Dibuka!", "success")

	http.Redirect(
		w,
		r,
		"/hazard/edit/"+id,
		http.StatusSeeOther,
	)
}
func (hc *HazardController) SyncField(w http.ResponseWriter, r *http.Request) {

	fieldName := r.FormValue("name")
	value := r.FormValue("value")

	resp := map[string]interface{}{
		"status": "success",
	}

	switch fieldName {

	// =========================
	// EVENT TYPES
	// =========================
	case "parent_id", "event_category_id":

		if value == "" || value == "0" {

			resp["types"] = []models.EventCategory{}

		} else {

			var filteredTypes []models.EventCategory

			err := hc.DB.
				Where("parent_id = ?", value).
				Order("name asc").
				Find(&filteredTypes).Error

			if err != nil {

				http.Error(
					w,
					"Database error",
					http.StatusInternalServerError,
				)

				return
			}

			resp["types"] = filteredTypes
		}

	// =========================
	// PICS - DEPARTMENT
	// =========================
	case "department_id":

		var pics []models.User

		if value != "" && value != "0" {

			hc.DB.
				Where(
					"department_id = ? AND is_pic = ?",
					value,
					true,
				).
				Order("name asc").
				Limit(50).
				Find(&pics)

		}

		resp["pics"] = pics
	// =========================
	// PICS - DEPARTMENT TERKAIT
	// =========================
	case "department_terkait_id":

		var pics_terkait []models.User

		if value != "" && value != "0" {

			hc.DB.
				Where(
					"department_id = ? AND is_pic = ?",
					value,
					true,
				).
				Order("name asc").
				Limit(50).
				Find(&pics_terkait)

		}

		resp["pics_terkait"] = pics_terkait

	// =========================
	// PICS - CONTRACTOR
	// =========================
	case "contractor_id":

		var pics []models.User

		if value != "" && value != "0" {

			hc.DB.
				Where(
					"contractor_id = ? AND is_pic = ?",
					value,
					true,
				).
				Order("name asc").
				Limit(50).
				Find(&pics)

		}

		resp["pics"] = pics
	// =========================
	// PICS - CONTRACTOR TERKAIT
	// =========================
	case "contractor_terkait_id":

		var pics_terkait []models.User

		if value != "" && value != "0" {

			hc.DB.
				Where(
					"contractor_id = ? AND is_pic = ?",
					value,
					true,
				).
				Order("name asc").
				Limit(50).
				Find(&pics_terkait)

		}

		resp["pics_terkait"] = pics_terkait
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	json.NewEncoder(w).Encode(resp)
}
func (hc *HazardController) setFlash(w http.ResponseWriter, msg string, msgType string) {
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg",
		Value: msg,
		Path:  "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: msgType,
		Path:  "/",
	})
}
func GetMatrixFunc() template.FuncMap {
	return template.FuncMap{

		"getMatrixColor": func(matrices []models.RiskMatrix, consID, likeID uint) string {
			for _, m := range matrices {
				if m.RiskConsequenceID == consID && m.RiskLikelihoodID == likeID {
					return m.RiskAssessment.Colour
				}
			}
			return "#ffffff" // Default putih jika tidak ada mapping
		},
		"getMatrixName": func(matrices []models.RiskMatrix, consID, likeID uint) string {
			for _, m := range matrices {
				if m.RiskConsequenceID == consID && m.RiskLikelihoodID == likeID {
					return m.RiskAssessment.Name
				}
			}
			return "-" // Default jika kosong
		},
		"firstChar": func(s string) string {
			if s == "" {
				return ""
			}
			return string([]rune(s)[0])
		},
	}
}

// PENCARIAN USER
func (hc *HazardController) Search(w http.ResponseWriter, r *http.Request) {
	// models.User{} memberi tahu GORM untuk mencari di tabel users
	utils.GlobalSearch(hc.DB, w, r, models.User{}, "name")
}

// PENCARIAN LAIN (Contoh: Unit Kerja / Departemen)
// PENCARIAN LOKASI
func (hc *HazardController) SearchLocation(w http.ResponseWriter, r *http.Request) {
	// models.Location{} memberi tahu GORM untuk mencari di tabel locations
	utils.GlobalSearch(hc.DB, w, r, models.Location{}, "name")
}
