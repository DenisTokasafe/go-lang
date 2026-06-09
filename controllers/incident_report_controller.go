package controllers

import (
	"encoding/json"
	"fmt"
	"latihan1/cmd/web/helpers"
	"latihan1/models"
	"latihan1/services"
	"latihan1/utils"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"gorm.io/gorm"
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
