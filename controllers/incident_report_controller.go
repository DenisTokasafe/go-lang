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
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IncidentController struct {
	DB              *gorm.DB
	ServiceIncident services.IncidentService
	Render          func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{}) // Sesuaikan tipe render Anda

}

func (ic *IncidentController) Create(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil string parent_id (Sama seperti Hazard)
	rawID := r.FormValue("parent_id")
	eventCategoryID, _ := strconv.ParseUint(rawID, 10, 32)
	fmt.Printf("Value: %v | Type: %T\n", eventCategoryID, eventCategoryID)

	// 2. Ambil parameter page untuk Risk Matrix grid
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	// 3. Panggil Service untuk mengambil semua master data + pagination matrix
	refs, err := ic.ServiceIncident.GetFormDataReferences(page)
	if err != nil {
		http.Error(w, "Gagal mengambil data referensi: "+err.Error(), http.StatusInternalServerError)
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
	// ... (Logika r.ParseMultipartForm, tangkap input Bagian 1, dan upload file tetap ada di sini) ...
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Gagal memproses form", http.StatusBadRequest)
		return
	}
	// Siapkan objek Bagian 1
	// 1. Buat helper parsing lokal untuk mengubah string dari form ke tipe data Go
	parseDate := func(str string) time.Time {
		if str == "" {
			return time.Now()
		}
		// Ganti layout format string ini sesuai dengan format keluaran dari flatpickr JS Anda
		t, err := time.Parse("2006-01-02 15:04", str)
		if err != nil {
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
		http.Error(w, "Gagal membuat nomor laporan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Inisialisasi struct dengan nomor yang sudah digenerate
	incident := models.IncidentReport{
		RefNumber:       newRefNumber, // Masukkan hasil generate di sini
		EventCategoryID: parseUint(r.FormValue("event_type_id")),
		ScatOptionID:    parseUint(r.FormValue("scat_option_id")),
		RiskMatrixID:    parseUint(r.FormValue("risk_matrix_id")),

		// Waktu Kejadian
		TanggalWaktu: parseDate(r.FormValue("event_date")),

		// Lokasi (Non-Pointer uint)
		LocationID:       parseUint(r.FormValue("location_id")),
		LocationSpecific: r.FormValue("location_specific"),

		// PIC & Penanggung Jawab
		PicID:        parseUint(r.FormValue("pic_id")),           // Wajib ada (Non-Pointer)
		DepartmentID: parseUintPtr(r.FormValue("department_id")), // Pointer (Bisa NULL)
		ContractorID: parseUintPtr(r.FormValue("contractor_id")), // Pointer (Bisa NULL)

		// Pelapor / User yang menginput (Pointer)
		ReportByID:     parseUintPtr(r.FormValue("reporter_id")),
		ReporterManual: r.FormValue("reporter_manual"), // Jika dari template select-user diinput manual

		// Deskripsi & Detail Operasional Lapangan (CKEditor Text)
		TugasDijalankan:        r.FormValue("tugas_dijalankan"),
		Deskripsi:              r.FormValue("deskripsi"),
		TindakanLangsung:       r.FormValue("tindakan_langsung"),
		DetilKerusakanKerugian: r.FormValue("detil_kerusakan_kerugian"),

		// Field Opsional (Bisa diambil dari form jika nanti ada inputannya)
		AreaKontrakKarya:      r.FormValue("area_kontrak_karya"),
		PotensiLTIFatality:    parseBool(r.FormValue("potensi_lti_fatality")),
		KlasifikasiLingkungan: r.FormValue("klasifikasi_lingkungan"),
		PekerjaanBerhenti:     parseBool(r.FormValue("pekerjaan_berhenti")),
	}

	// Tangkap string JSON Bagian 2 dari hidden input
	// Tangkap string JSON Bagian 2 dari hidden input
	partiesJSON := r.FormValue("involved_parties_json")
	var finalParties []models.InvolvedParty

	if partiesJSON != "" {
		// 1. Ubah struct sementara: Pengalaman sekarang menjadi interface{}
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
			Pengalaman     interface{} `json:"pengalaman"` // <-- UBAH DI SINI
		}

		// Helper 1: Untuk Pointer Uint (ID)
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

		// Helper 2: Untuk Angka Biasa (Pengalaman) <-- TAMBAHAN BARU
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

		// Proses Unmarshal
		err := json.Unmarshal([]byte(partiesJSON), &rawParties)
		if err != nil {
			fmt.Println("DEBUG ERROR UNMARSHAL JSON:", err)
		} else {
			for _, rp := range rawParties {
				userIDPtr := parseJSONToUintPtr(rp.UserID)

				// Cegah baris kosong
				if rp.ReporterManual == "" && userIDPtr == nil {
					continue
				}

				party := models.InvolvedParty{
					ReporterManual: rp.ReporterManual,
					Jabatan:        rp.Jabatan,
					Roster:         rp.Roster,
					Shift:          rp.Shift,
					Keterlibatan:   rp.Keterlibatan,
					Pengalaman:     parseJSONToInt(rp.Pengalaman), // <-- GUNAKAN HELPER
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

	fmt.Printf("DEBUG: Jumlah data parties yang akan disimpan: %d\n", len(finalParties))
	for i, p := range finalParties {
		fmt.Printf("DEBUG: Party[%d] - Manual: %s, UserID: %v\n", i, p.ReporterManual, p.UserID)
	}

	// ==========================================
	// PANGGIL SERVICE (Simpan Bagian 1 & Bagian 2)
	// ==========================================
	err = ic.ServiceIncident.CreateIncident(&incident, finalParties)

	if err != nil {
		// Log error jika diperlukan
		http.Error(w, "Gagal menyimpan laporan insiden ke database", http.StatusInternalServerError)
		return
	}

	// Redirect sukses
	http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Laporan+Berhasil+Dibuat", Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "success", Path: "/"})
	http.Redirect(w, r, "/incident", http.StatusSeeOther)
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
		fmt.Printf("Error fetching edit data: %v\n", err)
		http.Error(w, err.Error(), http.StatusNotFound)
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
		fmt.Printf("DEBUG: Memproses Cause ID: %d | Type: '%s'\n", cause.ID, cause.CategoryType) // <--- TAMBAHKAN INI

		if _, exists := groupedCauses[cause.CategoryType]; exists {
			groupedCauses[cause.CategoryType] = append(groupedCauses[cause.CategoryType], cause)
		} else {
			fmt.Printf("WARNING: CategoryType '%s' tidak cocok dengan map!\n", cause.CategoryType) // <--- TAMBAHKAN INI
		}
	}

	// Bongkar struct EditData ke dalam map agar template HTML tidak pecah
	data := map[string]interface{}{
		"Lang":            lang,
		"Tr":              helpers.Translations[lang],
		"Title":           "Edit Laporan Investigasi Insiden KPLH",
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
	fmt.Println("--- CONTROLLER UPDATE DIPANGGIL ---")
	fmt.Printf("DEBUG - URL Request: %s\n", r.URL.Path)
	fmt.Printf("DEBUG - ID yang tertangkap: '%s'\n", r.PathValue("id"))

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

	// 1. PARSE MULTIPART FORM terlebih dahulu (Maksimal 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Gagal memproses form multipart: "+err.Error(), http.StatusBadRequest)
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
	}

	// 3. UNMARSHAL string JSON tersebut ke struct req
	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		http.Error(w, "Invalid input JSON: "+err.Error(), http.StatusBadRequest)
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
	partiesUpdated, err := ic.ServiceIncident.UpdateIncident(uint(id), userID.ID, &req.Incident, req.InvolvedParties, req.InvestigationParticipants, req.PeepoFactors, req.Timelines, req.Causes, r)

	if err != nil {
		fmt.Printf("DEBUG - Service Error: %v\n", err)
		http.Error(w, "Gagal mengupdate data: "+err.Error(), http.StatusInternalServerError)
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
	redirectURL := fmt.Sprintf("/incident/edit/%s", idStr)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":       "success",
		"message":      message,
		"redirect_url": redirectURL,
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

			ic.DB.
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

			ic.DB.
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

			ic.DB.
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
