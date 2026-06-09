package controllers

import (
	"latihan1/cmd/web/helpers"
	"latihan1/models"
	"math"
	"net/http"
	"strconv"
	"text/template"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type RiskMatrixController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (rmc *RiskMatrixController) Index(w http.ResponseWriter, r *http.Request) {
	// 1. Parameter URL
	querySearch := r.URL.Query().Get("search")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize := 25 // Biasanya matrix butuh view lebih banyak (5x5 = 25)
	offset := (page - 1) * pageSize

	var matrices []models.RiskMatrix
	var totalRows int64

	// 2. Query dengan Preload untuk Relasi
	// Kita join tabel terkait agar search bisa mencari nama Consequence atau Likelihood
	dbQuery := rmc.DB.Model(&models.RiskMatrix{}).
		Preload("RiskConsequence").
		Preload("RiskLikelihood").
		Preload("RiskAssessment").
		Joins("JOIN risk_consequences ON risk_consequences.id = risk_matrices.risk_consequence_id").
		Joins("JOIN risk_likelihoods ON risk_likelihoods.id = risk_matrices.risk_likelihood_id").
		Joins("JOIN risk_assessment_codes ON risk_assessment_codes.id = risk_matrices.risk_assessment_id")

	// 3. Filter Search (Bisa cari berdasarkan nama consequence atau likelihood)
	if querySearch != "" {
		dbQuery = dbQuery.Where("risk_consequences.name LIKE ? OR risk_likelihoods.name LIKE ?",
			"%"+querySearch+"%", "%"+querySearch+"%")
	}

	// 4. Hitung Total & Ambil Data
	dbQuery.Count(&totalRows)
	dbQuery.Limit(pageSize).Offset(offset).
		Order("risk_consequences.severity_level DESC, risk_likelihoods.sequence ASC").
		Find(&matrices)

	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))
	var consequences []models.RiskConsequence
	var likelihoods []models.RiskLikelihood
	var assessments []models.RiskAssessmentCode

	// Ambil semua data referensi untuk membangun grid matriks
	rmc.DB.Order("severity_level DESC").Find(&consequences)
	rmc.DB.Order("sequence ASC").Find(&likelihoods)
	rmc.DB.Find(&assessments)
	// 5. Data untuk Template
	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":         lang,
		"Tr":           helpers.Translations[lang],
		"Matrices":     matrices,
		"Consequences": consequences, // Untuk header kolom grid
		"Likelihoods":  likelihoods,  // Untuk row baris grid
		"Assessments":  assessments,  // Untuk pilihan di modal
		"CurrentPage":  page,
		"TotalPages":   totalPages,
		"Search":       querySearch,
		"HasNext":      page < totalPages,
		"HasPrev":      page > 1,
		"TotalRows":    totalRows,
	}

	rmc.Render(w, r, "/administration/riskMatrix/index.gohtml", data)
}

func (rmc *RiskMatrixController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		consID, _ := strconv.ParseUint(r.FormValue("risk_consequence_id"), 10, 32)
		likeID, _ := strconv.ParseUint(r.FormValue("risk_likelihood_id"), 10, 32)
		asmtID, _ := strconv.ParseUint(r.FormValue("risk_assessment_id"), 10, 32)

		if consID > 0 && likeID > 0 && asmtID > 0 {
			models.CreateRiskMatrix(rmc.DB, uint(consID), uint(likeID), uint(asmtID))
		}
	}
	rmc.setFlash(w, "Aturan matrix berhasil ditambahkan", "success")
	http.Redirect(w, r, "/administration/risk/matrix", http.StatusSeeOther)
}

func (rmc *RiskMatrixController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		asmtID, _ := strconv.ParseUint(r.FormValue("risk_assessment_id"), 10, 32)

		if id != "" && asmtID > 0 {
			models.UpdateRiskMatrix(rmc.DB, id, uint(asmtID))
		}
	}
	rmc.setFlash(w, "Aturan matrix berhasil diperbarui", "success")
	http.Redirect(w, r, "/administration/risk/matrix", http.StatusSeeOther)
}

func (rmc *RiskMatrixController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id != "" {
		models.DeleteRiskMatrix(rmc.DB, id)
	}
	rmc.setFlash(w, "Aturan matrix berhasil dihapus", "error")
	http.Redirect(w, r, "/administration/risk/matrix", http.StatusSeeOther)
}

func (rmc *RiskMatrixController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("excel_file")
	if err != nil {
		http.Error(w, "Gagal mengambil file", 400)
		return
	}
	defer file.Close()

	f, _ := excelize.OpenReader(file)
	defer f.Close()

	rows, _ := f.GetRows(f.GetSheetName(0))

	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}

		// Di Excel sebaiknya berisi ID atau Nama yang unik untuk di-lookup
		consID, _ := strconv.ParseUint(row[0], 10, 32)
		likeID, _ := strconv.ParseUint(row[1], 10, 32)
		asmtID, _ := strconv.ParseUint(row[2], 10, 32)

		if consID > 0 && likeID > 0 {
			rmc.DB.Where(models.RiskMatrix{
				RiskConsequenceID: uint(consID),
				RiskLikelihoodID:  uint(likeID),
			}).Assign(models.RiskMatrix{
				RiskAssessmentID: uint(asmtID),
			}).FirstOrCreate(&models.RiskMatrix{})
		}
	}

	rmc.setFlash(w, "Bulk upload matrix berhasil", "success")
	http.Redirect(w, r, "/administration/risk/matrix", http.StatusSeeOther)
}

// Helper untuk kemudahan set cookie flash
func (rmc *RiskMatrixController) setFlash(w http.ResponseWriter, msg, msgType string) {
	http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: msg, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: msgType, Path: "/"})
}

// Buat fungsi ini di dalam file controller atau utility Anda
func GetMatrixFuncs() template.FuncMap {
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
	}
}
