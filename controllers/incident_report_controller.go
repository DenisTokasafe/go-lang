package controllers

import (
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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/csrf"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IncidentController struct {
	DB              *gorm.DB
	ServiceIncident services.IncidentService
	Render          func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{}) // Sesuaikan tipe render Anda

}

func (ic *IncidentController) Index(w http.ResponseWriter, r *http.Request) {
	// ==========================================
	// AMBIL CURRENT USER DARI CONTEXT
	// ==========================================
	userRaw, ok := r.Context().Value(middlewares.AuthUserKey).(models.User)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// ==========================================
	// AMBIL QUERY PARAMETERS (FILTERS & SEARCH)
	// ==========================================
	search := strings.TrimSpace(
		r.URL.Query().Get("search"),
	)

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	filterCategory := r.URL.Query().Get("category")
	filterLocation := r.URL.Query().Get("location")
	filterRisk := r.URL.Query().Get("risk")
	filterScat := r.URL.Query().Get("scat")
	filterStatus := r.URL.Query().Get("status") // Tambahan opsional yang penting untuk insiden

	page, err := strconv.Atoi(
		r.URL.Query().Get("page"),
	)

	if err != nil || page < 1 {
		page = 1
	}

	// ==========================================
	// GET DATA FROM SERVICE (USER RAW DISISIPKAN)
	// ==========================================
	result, err := ic.ServiceIncident.GetIncidentReports(
		userRaw,
		search,
		startDate,
		endDate,
		filterCategory,
		filterLocation,
		filterRisk,
		filterScat,
		filterStatus,
		page,
	)

	if err != nil {
		log.Println(
			"GET INCIDENT REPORTS ERROR:",
			err,
		)

		http.Error(
			w,
			"Failed load incident reports",
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
		"Title": "Incident Reports",

		"Incidents": result.Incidents, // Berisi []IncidentReport atau []IncidentReportDisplay

		"Search":         search,
		"FilterCategory": filterCategory,
		"FilterLocation": filterLocation,
		"FilterRisk":     filterRisk,
		"FilterScat":     filterScat,
		"FilterStatus":   filterStatus,

		// Master data untuk dropdown filter di view
		"Categories":      result.Categories,
		"ScatOptions":     result.ScatOptions,
		"RiskAssessments": result.RiskAssessments, // atau result.RiskMatrices
		"Locations":       result.Locations,

		"StartDate": startDate,
		"EndDate":   endDate,

		"CurrentPage": result.CurrentPage,
		"TotalPages":  result.TotalPages,
		"TotalRows":   result.TotalRows,
	}

	ic.Render(
		w,
		r,
		"/incident_report/index.gohtml",
		data,
	)
}

func (ic *IncidentController) Create(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil string parent_id (Sama seperti Hazard)

	// 2. Ambil parameter page untuk Risk Matrix grid
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	// 3. Panggil Service untuk mengambil semua master data + pagination matrix
	refs, err := ic.ServiceIncident.GetFormDataReferences(page)
	if err != nil {
		log.Println("GET FORM DATA REFERENCES ERROR:", err)
		http.Error(w, "Gagal mengambil data referensi formulir.", http.StatusInternalServerError)
		return
	}

	// 4. Pengaturan Bahasa/Lokalisasi
	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	// 5. Gabungkan data master dengan data template context (Format persis Hazard)
	data := map[string]interface{}{
		"Lang":            lang,
		"Tr":              helpers.Translations[lang],
		"Title":           "Tambah Laporan Investigasi Insiden KPLH",
		"CSRFField":       csrf.TemplateField(r), // Token untuk form biasa (hidden input)
		"CSRFToken":       csrf.Token(r),         // Token mentah untuk dipakai di header fetch/AJAX
		"Matrices":        refs["Matrices"],
		"Consequences":    refs["Consequences"],
		"Likelihoods":     refs["Likelihoods"],
		"Assessments":     refs["Assessments"],
		"EventCategories": refs["EventCategories"],
		"Locations":       refs["Locations"],
		"RiskMatrices":    refs["RiskMatrices"],
		"Departments":     refs["Departments"],
		"Contractors":     refs["Contractors"],
		"Users":           refs["Users"],
		"ScatOptions":     refs["ScatOptions"],
		"TotalRows":       refs["TotalRows"],
		"TotalPages":      refs["TotalPages"],
		"CurrentTime":     time.Now().Format("2006-01-02T15:04"),
	}

	// 6. Render ke view gohtml
	ic.Render(w, r, "/incident_report/create.gohtml", data)
}

func (ic *IncidentController) Store(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	err := r.ParseMultipartForm(5 << 20)
	if err != nil {
		http.Error(w, "Gagal memproses form", http.StatusBadRequest)
		return
	}

	// ==========================================
	// 1. HELPER PARSING & BAGIAN 1 (INFO UTAMA)
	// ==========================================
	parseDate := func(str string) time.Time {
		if str == "" {
			return time.Now()
		}
		layout := "2006-01-02 15:04"
		t, err := time.ParseInLocation(layout, str, time.Local)
		if err != nil {
			log.Printf("Gagal memproses string tanggal '%s': %v", str, err)
			return time.Now()
		}
		return t
	}
	parseUint := func(s string) uint {
		val, _ := strconv.ParseUint(s, 10, 32)
		return uint(val)
	}
	parseUintPtr := func(s string) *uint {
		if s == "" || s == "0" || s == "undefined" {
			return nil
		}
		val, _ := strconv.ParseUint(s, 10, 32)
		u := uint(val)
		return &u
	}
	parseBool := func(s string) bool {
		return s == "true" || s == "1" || s == "on"
	}

	newRefNumber, err := ic.GenerateRefNumber(ic.DB)
	if err != nil {
		log.Println("GENERATE REF NUMBER ERROR:", err)
		http.Error(w, "Gagal membuat nomor laporan.", http.StatusInternalServerError)
		return
	}

	incident := models.IncidentReport{
		RefNumber:              newRefNumber,
		EventCategoryID:        parseUint(r.FormValue("event_type_id")),
		ScatOptionID:           parseUint(r.FormValue("scat_option_id")),
		RiskMatrixID:           parseUint(r.FormValue("risk_matrix_id")),
		TanggalWaktu:           parseDate(r.FormValue("event_date")),
		LocationID:             parseUint(r.FormValue("location_id")),
		LocationSpecific:       r.FormValue("location_specific"),
		PicID:                  parseUintPtr(r.FormValue("pic_id")),
		DepartmentID:           parseUintPtr(r.FormValue("department_id")),
		ContractorID:           parseUintPtr(r.FormValue("contractor_id")),
		ReportByID:             parseUintPtr(r.FormValue("report_by_id")),
		ReporterManual:         r.FormValue("reporter_manual"),
		TugasDijalankan:        r.FormValue("tugas_dijalankan"),
		Deskripsi:              r.FormValue("deskripsi"),
		TindakanLangsung:       r.FormValue("tindakan_langsung"),
		DetilKerusakanKerugian: r.FormValue("detil_kerusakan_kerugian"),
		AreaKontrakKarya:       r.FormValue("area_kontrak_karya"),
		PotensiLTIFatality:     parseBool(r.FormValue("potensi_lti_fatality")),
		KlasifikasiLingkungan:  r.FormValue("klasifikasi_lingkungan"),
		PekerjaanBerhenti:      parseBool(r.FormValue("pekerjaan_berhenti")),
	}

	// ==========================================
	// 2. BAGIAN 2 (INVOLVED PARTIES JSON)
	// ==========================================
	partiesJSON := r.FormValue("involved_parties_json")
	var finalParties []models.InvolvedParty

	if partiesJSON != "" {
		var rawParties []struct {
			UserID         interface{} `json:"user_id"`
			ReporterManual string      `json:"reporter_manual"`
			WorkType       string      `json:"work_type"`
			DepartmentID   interface{} `json:"department_id"`
			ContractorID   interface{} `json:"contractor_id"`
			Jabatan        string      `json:"jabatan"`
			Roster         string      `json:"roster"`
			Shift          string      `json:"shift"`
			Keterlibatan   string      `json:"keterlibatan"`
			Pengalaman     interface{} `json:"pengalaman"`
		}

		parseJSONToUintPtr := func(val interface{}) *uint {
			if val == nil {
				return nil
			}
			strVal := fmt.Sprintf("%v", val)
			if strVal == "" || strVal == "0" || strVal == "undefined" || strVal == "null" {
				return nil
			}
			parsedVal, _ := strconv.ParseUint(strVal, 10, 32)
			u := uint(parsedVal)
			return &u
		}

		parseJSONToInt := func(val interface{}) int {
			if val == nil {
				return 0
			}
			strVal := fmt.Sprintf("%v", val)
			if strVal == "" || strVal == "undefined" || strVal == "null" {
				return 0
			}
			parsedVal, _ := strconv.Atoi(strVal)
			return parsedVal
		}

		err := json.Unmarshal([]byte(partiesJSON), &rawParties)
		if err != nil {
			log.Println("DEBUG ERROR UNMARSHAL JSON:", err)
		} else {
			for _, rp := range rawParties {
				userIDPtr := parseJSONToUintPtr(rp.UserID)
				if rp.ReporterManual == "" && userIDPtr == nil {
					continue
				}

				party := models.InvolvedParty{
					ReporterManual: rp.ReporterManual,
					Jabatan:        rp.Jabatan,
					Roster:         rp.Roster,
					Shift:          rp.Shift,
					Keterlibatan:   rp.Keterlibatan,
					Pengalaman:     parseJSONToInt(rp.Pengalaman),
				}

				if userIDPtr != nil && *userIDPtr > 0 {
					party.UserID = userIDPtr
				}
				if rp.WorkType == "department" {
					party.DepartmentID = parseJSONToUintPtr(rp.DepartmentID)
				} else if rp.WorkType == "contractor" {
					party.ContractorID = parseJSONToUintPtr(rp.ContractorID)
				}

				finalParties = append(finalParties, party)
			}
		}
	}

	// ==========================================
	// 3. PANGGIL SERVICE (Simpan Bagian 1, 2, & Upload Dokumentasi)
	// ==========================================
	// Upload file sekarang ditangani SEPENUHNYA di dalam service (s.uploadFiles),
	// sama persis dengan jalur Update. Controller cukup meneruskan *http.Request.
	err = ic.ServiceIncident.CreateIncident(&incident, finalParties, r)
	if err != nil {
		log.Printf("Error saat menyimpan insiden ke service: %v", err)
		http.Error(w, "Gagal menyimpan laporan insiden ke database", http.StatusInternalServerError)
		return
	}

	// Redirect sukses
	http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Laporan+Berhasil+Dibuat", Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "success", Path: "/"})
	http.Redirect(w, r, "/incident", http.StatusSeeOther)
}

// Document serves an Incident attachment from private storage after confirming
// that the documentation record belongs to an Incident.
func (ic *IncidentController) Document(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var doc models.Documentation
	if err := ic.DB.First(&doc, uint(id)).Error; err != nil || !strings.HasPrefix(doc.FileURL, "/storage/uploads/incidents/") {
		http.NotFound(w, r)
		return
	}
	var count int64
	ic.DB.Model(&models.IncidentDocumentation{}).Where("documentation_id = ?", doc.ID).Count(&count)
	if count == 0 {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(".", filepath.FromSlash(strings.TrimPrefix(doc.FileURL, "/")))
	if _, err := os.Stat(path); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, path)
}

// incident_report_controller.go

// Edit menampilkan halaman form edit
// incident_report_controller.go

func (ic *IncidentController) Edit(w http.ResponseWriter, r *http.Request) {
	idParam := r.PathValue("id")
	if idParam == "" {
		http.Error(w, "ID tidak ditemukan di URL", http.StatusBadRequest)
		return
	}

	// --- PERBAIKAN: Konversi string ke int/uint menggunakan strconv ---
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		http.Error(w, "Format ID tidak valid", http.StatusBadRequest)
		return
	}

	// Dapatkan User ID yang sedang login
	userID, ok := r.Context().Value(middlewares.AuthUserKey).(models.User)
	if !ok {
		http.Error(w, "User tidak terautentikasi", http.StatusUnauthorized)
		return
	}

	// Panggil Service dengan variabel 'id' yang sudah berhasil dikonversi
	editData, err := ic.ServiceIncident.GetEditData(uint(id), userID.ID, 1)
	if err != nil {
		log.Println("GET EDIT DATA ERROR:", err)
		if strings.Contains(err.Error(), "unauthorized") {
			// Pesan ini sengaja dibuat aman ditampilkan (tidak mengandung detail internal)
			http.Error(w, "Anda tidak memiliki akses untuk mengedit laporan ini.", http.StatusForbidden)
		} else {
			http.Error(w, "Data insiden tidak ditemukan.", http.StatusNotFound)
		}
		return
	}

	toJSON := func(v interface{}) template.JS {
		b, _ := json.Marshal(v)
		return template.JS(b)
	}

	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}
	groupedCauses := map[string][]models.IncidentCause{
		"unsafe_condition": {},
		"unsafe_act":       {},
		"personal_factor":  {},
		"job_factor":       {},
		"control_system":   {},
	}

	// Kelompokkan berdasarkan CategoryType
	for _, cause := range editData.Incident.Causes {

		if _, exists := groupedCauses[cause.CategoryType]; exists {
			groupedCauses[cause.CategoryType] = append(groupedCauses[cause.CategoryType], cause)
		} else {
			log.Printf("WARNING: CategoryType '%s' tidak cocok dengan map!\n", cause.CategoryType)
		}
	}

	// Bongkar struct EditData ke dalam map agar template HTML tidak pecah
	data := map[string]interface{}{
		"Lang":            lang,
		"Tr":              helpers.Translations[lang],
		"Title":           "Edit Laporan Investigasi Insiden KPLH",
		"CSRFField":       csrf.TemplateField(r), // Token untuk form biasa (hidden input)
		"CSRFToken":       csrf.Token(r),         // Token mentah untuk dipakai di header fetch/AJAX
		"Incident":        editData.Incident,
		"Docs":            toJSON(editData.Docs),
		"WorkType":        editData.WorkType,
		"CanReopen":       editData.CanReopen,
		"IsModerator":     editData.IsModerator,
		"Matrices":        editData.Matrices,
		"Consequences":    editData.Consequences,
		"Likelihoods":     editData.Likelihoods,
		"Assessments":     editData.Assessments,
		"EventCategories": editData.EventCategories,
		"Locations":       editData.Locations,
		"RiskMatrices":    editData.RiskMatrices,
		"Departments":     editData.Departments,
		"Contractors":     editData.Contractors,
		"Users":           editData.Users,
		"ScatOptions":     editData.ScatOptions,
		"ScatOptionsAll":  editData.ScatOptionsAll,
		"GroupedCauses":   toJSON(groupedCauses),
		"TotalRows":       editData.TotalRows,
		"AllTypes":        editData.AllTypes,
		"AllPics":         editData.AllPics,
		"PageSize":        editData.PageSize,
		"TotalPages":      editData.TotalPages,
		"CurrentTime":     time.Now().Format("2006-01-02T15:04"),
	}

	ic.Render(w, r, "incident_report/edit.gohtml", data)
}

// Update memproses form submit
// Update memproses form submit via JSON Fetch Multipart
func (ic *IncidentController) Update(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middlewares.AuthUserKey).(models.User)
	if !ok {
		http.Error(w, "User tidak terautentikasi", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Batas request body diterapkan sebelum multipart parsing untuk mencegah DoS.
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		log.Println("PARSE MULTIPART FORM ERROR:", err)
		http.Error(w, "Gagal memproses form. Pastikan ukuran file tidak melebihi batas.", http.StatusBadRequest)
		return
	}

	// 2. AMBIL STRING JSON dari Form Data dengan key 'payload'
	jsonPayload := r.FormValue("payload")
	if jsonPayload == "" {
		http.Error(w, "Error: Payload JSON tidak ditemukan", http.StatusBadRequest)
		return
	}

	var req struct {
		Incident                  models.IncidentReport             `json:"Incident"`
		InvolvedParties           []models.InvolvedParty            `json:"InvolvedParties"`
		InvestigationParticipants []models.InvestigationParticipant `json:"InvestigationParticipants"`
		PeepoFactors              []models.PeepoFactor              `json:"PeepoFactors"`
		Timelines                 []models.Timeline                 `json:"Timelines"`
		Causes                    []models.IncidentCause            `json:"IncidentCauses"`
		CorrectiveActionIncidents []models.CorrectiveActionIncident `json:"CorrectiveActionIncidents"`
		Reviews                   *models.IncidentReview            `json:"Reviews"`
	}

	// 3. UNMARSHAL string JSON tersebut ke struct req
	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		log.Println("UNMARSHAL UPDATE PAYLOAD ERROR:", err)
		http.Error(w, "Format data yang dikirim tidak valid.", http.StatusBadRequest)
		return
	}
	// --- TAMBAHAN BARU: Filter Data Causes Kosong ---
	var validCauses []models.IncidentCause
	for _, cause := range req.Causes {
		// Pastikan ID SCAT tidak kosong (sesuaikan dengan tipe data struct Anda)
		// Jika ScatOptionID berupa *uint (pointer):
		if cause.ScatOptionID != nil && *cause.ScatOptionID > 0 {
			validCauses = append(validCauses, cause)
		}

		// Catatan: Jika ScatOptionID di model berupa uint biasa (bukan pointer), gunakan ini:
		// if cause.ScatOptionID > 0 { validCauses = append(validCauses, cause) }
	}
	// -------------------------------------------------

	// 4. PANGGIL SERVICE dengan menyertakan 'r' untuk memproses file dokumentasi
	partiesUpdated, err := ic.ServiceIncident.UpdateIncident(uint(id), userID.ID, &req.Incident, req.InvolvedParties, req.InvestigationParticipants, req.PeepoFactors, req.Timelines, req.Causes, req.CorrectiveActionIncidents, req.Reviews, r)

	if err != nil {
		log.Println("UPDATE INCIDENT ERROR:", err)
		if strings.Contains(err.Error(), "unauthorized") {
			http.Error(w, "Anda tidak memiliki akses untuk mengedit laporan ini.", http.StatusForbidden)
		} else {
			http.Error(w, "Gagal mengupdate data laporan insiden.", http.StatusInternalServerError)
		}
		return
	}

	// --- PESAN SUKSES DINAMIS ---
	message := "Laporan Insiden Berhasil Diperbarui!"
	ic.setFlash(
		w, message, "success",
	)
	if partiesUpdated {
		message = "Laporan Insiden & Data Pihak Terlibat Berhasil Diperbarui!"
		ic.setFlash(
			w, message, "success",
		)
	}
	// redirectURL := fmt.Sprintf("/incident/edit/%s", idStr)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": message,
	})
	// Note: http.Redirect dihapus di sini karena redirect sudah ditangani oleh window.location di JS frontend Anda.
}

func (ic *IncidentController) setFlash(w http.ResponseWriter, msg string, msgType string) {
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

func GetIncidentMatrixFunc() template.FuncMap {
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

func (ic *IncidentController) SyncField(w http.ResponseWriter, r *http.Request) {

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

			err := ic.DB.
				Where("parent_id = ?", value).
				Order("name asc").
				Find(&filteredTypes).Error

			if err != nil {
				log.Println("SYNC FIELD DB ERROR:", err)
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

			ic.DB.
				Where(
					"department_id = ? AND is_pic = ?",
					value,
					true,
				).
				Order("name asc").
				Limit(20).
				Find(&pics)

		}

		resp["pics"] = pics
	// =========================
	// PICS - DEPARTMENT TERKAIT
	// =========================
	case "department_terkait_id":

		var pics_terkait []models.User

		if value != "" && value != "0" {

			ic.DB.
				Where(
					"department_id = ? AND is_pic = ?",
					value,
					true,
				).
				Order("name asc").
				Limit(20).
				Find(&pics_terkait)

		}

		resp["pics_terkait"] = pics_terkait

	// =========================
	// PICS - CONTRACTOR
	// =========================
	case "contractor_id":

		var pics []models.User

		if value != "" && value != "0" {

			ic.DB.
				Where(
					"contractor_id = ? AND is_pic = ?",
					value,
					true,
				).
				Order("name asc").
				Limit(20).
				Find(&pics)

		}

		resp["pics"] = pics
	// =========================
	// PICS - CONTRACTOR TERKAIT
	// =========================
	case "contractor_terkait_id":

		var pics_terkait []models.User

		if value != "" && value != "0" {

			ic.DB.
				Where(
					"contractor_id = ? AND is_pic = ?",
					value,
					true,
				).
				Order("name asc").
				Limit(20).
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

func (ic *IncidentController) Search(w http.ResponseWriter, r *http.Request) {
	// models.User{} memberi tahu GORM untuk mencari di tabel users
	utils.GlobalSearch(ic.DB, w, r, models.User{}, "name")
}

// PENCARIAN LAIN (Contoh: Unit Kerja / Departemen)
// PENCARIAN LOKASI
func (ic *IncidentController) SearchLocation(w http.ResponseWriter, r *http.Request) {
	// models.Location{} memberi tahu GORM untuk mencari di tabel locations
	utils.GlobalSearch(ic.DB, w, r, models.Location{}, "name")
}
func (ic *IncidentController) SearchContractor(w http.ResponseWriter, r *http.Request) {
	// models.Location{} memberi tahu GORM untuk mencari di tabel locations
	utils.GlobalSearch(ic.DB, w, r, models.Contractor{}, "name")
}
func (ic *IncidentController) SearchDepartment(w http.ResponseWriter, r *http.Request) {
	// models.Location{} memberi tahu GORM untuk mencari di tabel locations
	utils.GlobalSearch(ic.DB, w, r, models.Department{}, "name")
}

func (ic *IncidentController) GenerateRefNumber(db *gorm.DB) (string, error) {
	today := time.Now().Format("20060102")
	prefix := "HZ-" + today + "-"

	var lastIncident models.IncidentReport

	// Tambahkan .Clauses tepat di sini untuk mengunci baris data
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("ref_number LIKE ?", prefix+"%").
		Order("ref_number DESC").
		First(&lastIncident).
		Error

	// Penanganan error tetap sama
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	var refNumber string
	if lastIncident.ID == 0 {
		refNumber = prefix + "0001"
	} else {
		// Logika increment tetap sama
		lastNumber := lastIncident.RefNumber[len(prefix):]
		lastNumberInt, err := strconv.Atoi(lastNumber)
		if err != nil {
			return "", err
		}
		refNumber = fmt.Sprintf("%s%04d", prefix, lastNumberInt+1)
	}

	return refNumber, nil
}
