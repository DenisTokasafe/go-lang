package controllers

import (
	"latihan1/cmd/web/helpers"
	"latihan1/models"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type RiskAssessmentController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

// Index: Menampilkan daftar data
func (rac *RiskAssessmentController) Index(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil parameter 'page' dan 'search' dari URL
	querySearch := r.URL.Query().Get("search")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	// 2. Tentukan ukuran data per halaman
	pageSize := 20
	offset := (page - 1) * pageSize

	var assessments []models.RiskAssessmentCode
	var totalRows int64

	// 3. Inisialisasi Query Dasar
	dbQuery := rac.DB.Model(&models.RiskAssessmentCode{})

	// 4. Terapkan Filter Search jika ada
	if querySearch != "" {
		dbQuery = dbQuery.Where("name LIKE ?", "%"+querySearch+"%")
	}

	// 5. Hitung total baris berdasarkan filter
	dbQuery.Count(&totalRows)

	// 6. Ambil data dengan limit, offset, dan filter yang sama
	// Menggunakan Order sequence DESC sesuai logika risk assessment
	dbQuery.Limit(pageSize).Offset(offset).Order("sequence DESC").Find(&assessments)

	// 7. Hitung total halaman
	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))

	// 8. Kirim data ke template
	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":        lang,
		"Tr":          helpers.Translations[lang],
		"Assessments": assessments,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Search":      querySearch,
		"HasNext":     page < totalPages,
		"HasPrev":     page > 1,
		"TotalRows":   totalRows,
		"Title":       "Risk Assessment Codes",
	}

	rac.Render(w, r, "administration/riskAssessment/index.gohtml", data)
}

// Store: Menyimpan data baru
func (rac *RiskAssessmentController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		seq, _ := strconv.Atoi(r.FormValue("sequence"))
		days, _ := strconv.Atoi(r.FormValue("action_days"))

		assessment := models.RiskAssessmentCode{
			Name:                r.FormValue("name"),
			Notes:               r.FormValue("notes"),
			ActionDays:          days,
			Sequence:            seq,
			InvestigationReqd:   r.FormValue("investigation_reqd"),
			ReportingObligation: r.FormValue("reporting_obligation"),
			Colour:              r.FormValue("colour"),
		}
		rac.DB.Create(&assessment)
	}
	rac.setFlash(w, "Data berhasil ditambahkan", "success")
	http.Redirect(w, r, "/administration/risk/assessment", http.StatusSeeOther)
}

// Update: Memperbarui data
func (rac *RiskAssessmentController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		seq, _ := strconv.Atoi(r.FormValue("sequence"))
		days, _ := strconv.Atoi(r.FormValue("action_days"))

		rac.DB.Model(&models.RiskAssessmentCode{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name":                 r.FormValue("name"),
			"notes":                r.FormValue("notes"),
			"action_days":          days,
			"sequence":             seq,
			"investigation_reqd":   r.FormValue("investigation_reqd"),
			"reporting_obligation": r.FormValue("reporting_obligation"),
			"colour":               r.FormValue("colour"),
		})
	}
	rac.setFlash(w, "Data berhasil diupdate", "success")
	http.Redirect(w, r, "/administration/risk/assessment", http.StatusSeeOther)
}

// Delete: Menghapus data permanen
func (rac *RiskAssessmentController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id != "" {
		rac.DB.Unscoped().Delete(&models.RiskAssessmentCode{}, id)
	}
	rac.setFlash(w, "Data berhasil dihapus", "success")
	http.Redirect(w, r, "/administration/risk/assessment", http.StatusSeeOther)
}

// UploadExcel: Import massal dari Excel
func (rac *RiskAssessmentController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	file, _, _ := r.FormFile("excel_file")
	f, _ := excelize.OpenReader(file)
	rows, _ := f.GetRows(f.GetSheetName(0))

	for i, row := range rows {
		if i == 0 || len(row) < 1 {
			continue
		}

		name := strings.TrimSpace(row[0])
		days, _ := strconv.Atoi(row[2])
		seq, _ := strconv.Atoi(row[3])

		rac.DB.Where(models.RiskAssessmentCode{Name: name}).Assign(models.RiskAssessmentCode{
			Notes:               row[1],
			ActionDays:          days,
			Sequence:            seq,
			InvestigationReqd:   row[4],
			ReportingObligation: row[5],
			Colour:              row[6],
		}).FirstOrCreate(&models.RiskAssessmentCode{})
	}
	rac.setFlash(w, "Import Excel berhasil", "success")
	http.Redirect(w, r, "/administration/risk/assessment", http.StatusSeeOther)
}

// Helper untuk flash message
func (rac *RiskAssessmentController) setFlash(w http.ResponseWriter, msg string, msgType string) {
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
