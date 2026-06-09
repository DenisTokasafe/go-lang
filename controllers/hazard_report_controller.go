package controllers

import (
	// Sesuaikan dengan nama module di go.mod Anda
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"latihan1/cmd/web/helpers"
	"latihan1/middlewares"
	"latihan1/models"
	"latihan1/services"
	"latihan1/utils"
	"log"
	"net/http"
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
	// ==========================================
	// AMBIL CURRENT USER DARI CONTEXT (TAMBAHAN)
	// ==========================================
	userRaw, ok := r.Context().Value(middlewares.AuthUserKey).(models.User)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

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

	// ==========================================
	// GET DATA FROM SERVICE (USER RAW DISISIPKAN)
	// ==========================================
	result, err := hc.Service.GetHazards(
		userRaw, // <-- Parameter pertama sesuai update Service
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
	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":  lang,
		"Tr":    helpers.Translations[lang],
		"Title": "Hazard Reports",

		"Hazards": result.Hazards, // <-- Sekarang otomatis berisi []HazardDisplay

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

	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":            lang,
		"Tr":              helpers.Translations[lang],
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

	// 1. Ambil Current User dari Context
	userRaw, ok := r.Context().Value(middlewares.AuthUserKey).(models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Panggil Service untuk mengambil semua Data
	editData, err := hc.Service.GetHazardEditData(id, userRaw.ID)
	if err != nil {
		// 1. KONDISI: Data memang tidak ada di Database
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.NotFound(w, r)
			return
		}

		// 2. KONDISI: Data ada, tapi User TIDAK MEMILIKI AKSES (IDOR Prevention)
		// *Catatan: Sesuaikan 'services.ErrUnauthorizedHazard' dengan nama sentinel error di tempat Anda
		if errors.Is(err, services.ErrUnauthorizedHazard) {
			w.WriteHeader(http.StatusForbidden) // Sinyal 403 ke browser

			lang := "id"
			if cookie, err := r.Cookie("lang"); err == nil {
				lang = cookie.Value
			}

			// Pastikan key "Tr" dan "Lang" dikirim ke RenderTemplate
			data := map[string]interface{}{
				"Lang": lang,
				"Tr":   helpers.Translations[lang],
			}

			helpers.RenderTemplate(hc.DB, w, r, "errors/403.gohtml", data)
			return
		}

		// 3. KONDISI: Error sistem lainnya (DB down, query salah, dll)
		log.Println("EDIT HAZARD ERROR:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Helper JSON khusus view template
	toJSON := func(v interface{}) template.JS {
		b, _ := json.Marshal(v)
		return template.JS(b)
	}

	// 3. Siapkan View Data
	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":                 lang,
		"Tr":                   helpers.Translations[lang],
		"Title":                "Edit Hazard",
		"CanReopen":            editData.CanReopen,
		"IsModerator":          editData.IsModerator,
		"Hazard":               editData.Hazard,
		"CorrectiveAction":     editData.CorrectiveAction,
		"CorrectiveActions":    toJSON(editData.CorrectiveActionsDTO),
		"CorrectiveActionsRaw": editData.Hazard.CorrectiveActions,
		"Matrices":             editData.Matrices,
		"Consequences":         editData.Consequences,
		"Likelihoods":          editData.Likelihoods,
		"Assessments":          editData.Assessments,
		"Audits":               editData.Hazard.Audits,
		"EventCategories":      editData.EventCategories,
		"Locations":            editData.Locations,
		"RiskMatrices":         editData.RiskMatrices,
		"Departments":          editData.Departments,
		"Contractors":          editData.Contractors,
		"Users":                editData.Users,
		"ScatOptions":          editData.ScatOptions,

		// dynamic select
		"AllTypes": editData.AllTypes,
		"AllPics":  editData.AllPics,

		// JSON SAFE FOR ALPINE
		"DescDocs":       toJSON(editData.DescDocs),
		"CorrectiveDocs": toJSON(editData.CorrectiveDocs),

		"WorkType":   editData.WorkType,
		"StatusArea": editData.StatusArea,
	}

	// 4. Render
	hc.Render(w, r, "/hazard_report/edit.gohtml", data)
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
func (hc *HazardController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	userRaw, ok := r.Context().Value(middlewares.AuthUserKey).(models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := hc.Service.DeleteHazard(id, userRaw)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		// HAPUS hc.setFlash dari sini
		return
	}

	// Set Flash message sebelum mengirim response JSON sukses
	hc.setFlash(w, "Laporan Hazard Dihapus!", "success")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "success"})
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
func (hc *HazardController) SearchContractor(w http.ResponseWriter, r *http.Request) {
	// models.Location{} memberi tahu GORM untuk mencari di tabel locations
	utils.GlobalSearch(hc.DB, w, r, models.Contractor{}, "name")
}
