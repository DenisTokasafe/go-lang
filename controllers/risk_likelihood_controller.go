package controllers

import (
	"latihan1/cmd/web/helpers"
	"latihan1/models" // Sesuaikan dengan nama module di go.mod Anda
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type RiskLikelihoodController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (rlc *RiskLikelihoodController) Index(w http.ResponseWriter, r *http.Request) {
	querySearch := r.URL.Query().Get("search")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize := 20
	offset := (page - 1) * pageSize

	var likelihoods []models.RiskLikelihood
	var totalRows int64

	dbQuery := rlc.DB.Model(&models.RiskLikelihood{})

	if querySearch != "" {
		dbQuery = dbQuery.Where("name LIKE ? OR notes LIKE ?", "%"+querySearch+"%", "%"+querySearch+"%")
	}

	dbQuery.Count(&totalRows)

	// Urutkan berdasarkan Sequence sesuai gambar
	dbQuery.Limit(pageSize).Offset(offset).Order("sequence ASC").Find(&likelihoods)

	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))

	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":        lang,
		"Tr":          helpers.Translations[lang],
		"Likelihoods": likelihoods,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Search":      querySearch,
		"TotalRows":   totalRows,
		"Title":       "Risk Likelihood Management",
		"CurrentPath": r.URL.Path,
	}

	rlc.Render(w, r, "/administration/riskLikelihood/index.gohtml", data)
}

func (rlc *RiskLikelihoodController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		notes := r.FormValue("notes")
		seq, _ := strconv.Atoi(r.FormValue("sequence"))

		if name != "" {
			models.CreateRiskLikelihood(rlc.DB, name, notes, seq)
		}
	}
	rlc.setFlash(w, "Data berhasil disimpan", "success")
	http.Redirect(w, r, "/administration/risk/likelihood", http.StatusSeeOther)
}
func (rlc *RiskLikelihoodController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		name := r.FormValue("name")
		notes := r.FormValue("notes")
		sequence, _ := strconv.Atoi(r.FormValue("sequence"))

		if id != "" && name != "" {
			// Menggunakan Updates map agar bisa mengupdate beberapa kolom sekaligus
			rlc.DB.Model(&models.RiskLikelihood{}).Where("id = ?", id).Updates(map[string]interface{}{
				"name":     name,
				"notes":    notes,
				"sequence": sequence,
			})
		}
	}

	rlc.setFlash(w, "Data berhasil diupdate", "success")
	http.Redirect(w, r, "/administration/risk/likelihood", http.StatusSeeOther)
}

func (rlc *RiskLikelihoodController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		// Menggunakan Unscoped agar terhapus permanen sesuai pola yang Anda minta
		rlc.DB.Unscoped().Delete(&models.RiskLikelihood{}, id)
	}

	rlc.setFlash(w, "Data berhasil dihapus", "success")
	http.Redirect(w, r, "/administration/risk/likelihood", http.StatusSeeOther)
}

func (rlc *RiskLikelihoodController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/administration/risk/likelihood", http.StatusSeeOther)
		return
	}

	// 1. Ambil file dari form
	file, _, err := r.FormFile("excel_file")
	if err != nil {
		http.Error(w, "Gagal mengambil file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 2. Buka file excel
	f, err := excelize.OpenReader(file)
	if err != nil {
		http.Error(w, "Format file tidak didukung", http.StatusBadRequest)
		return
	}
	defer f.Close()

	// 3. Baca sheet pertama
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		http.Error(w, "Gagal membaca data", http.StatusInternalServerError)
		return
	}

	// 4. Looping data (Lewati baris pertama/header)
	for i, row := range rows {
		if i == 0 || len(row) < 1 {
			continue
		}

		// Ambil data dari kolom Excel
		name := strings.TrimSpace(row[0])
		notes := ""
		if len(row) > 1 {
			notes = strings.TrimSpace(row[1])
		}
		sequence := 0
		if len(row) > 2 {
			sequence, _ = strconv.Atoi(strings.TrimSpace(row[2]))
		}

		if name != "" {
			// Assign digunakan untuk mengupdate Notes dan Sequence jika Name sudah ditemukan
			rlc.DB.Where(models.RiskLikelihood{Name: name}).
				Assign(models.RiskLikelihood{
					Notes:    notes,
					Sequence: sequence,
				}).
				FirstOrCreate(&models.RiskLikelihood{})
		}
	}

	rlc.setFlash(w, "Data Likelihood berhasil diupload", "success")
	http.Redirect(w, r, "/administration/risk/likelihood", http.StatusSeeOther)
}

// Helper untuk flash message
func (rlc *RiskLikelihoodController) setFlash(w http.ResponseWriter, msg string, msgType string) {
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
