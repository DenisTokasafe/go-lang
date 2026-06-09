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

type RiskConsequenceController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (rcc *RiskConsequenceController) Index(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil Parameter Query
	querySearch := r.URL.Query().Get("search")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize := 20
	offset := (page - 1) * pageSize

	var consequences []models.RiskConsequence
	var totalRows int64

	// 2. Query Database dengan Search
	dbQuery := rcc.DB.Model(&models.RiskConsequence{})

	if querySearch != "" {
		// Cari berdasarkan nama atau deskripsi
		dbQuery = dbQuery.Where("name LIKE ? OR description LIKE ?", "%"+querySearch+"%", "%"+querySearch+"%")
	}

	// Hitung total data untuk pagination
	dbQuery.Count(&totalRows)

	// Ambil data dengan limit, offset, dan urutan Severity Level (1 ke 5)
	dbQuery.Limit(pageSize).Offset(offset).Order("severity_level ASC").Find(&consequences)

	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))

	// 3. Siapkan Data untuk Template
	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":         lang,
		"Tr":           helpers.Translations[lang],
		"Consequences": consequences,
		"CurrentPage":  page,
		"TotalPages":   totalPages,
		"Search":       querySearch,
		"TotalRows":    totalRows,
		"Title":        "Risk Consequence Management",
		"CurrentPath":  r.URL.Path,
	}

	rcc.Render(w, r, "/administration/riskRonsequence/index.gohtml", data)
}

func (rcc *RiskConsequenceController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Ambil data dari form
		name := r.FormValue("name")
		description := r.FormValue("description")
		reportable := r.FormValue("reportable")
		sequence, _ := strconv.Atoi(r.FormValue("sequence"))
		severity, _ := strconv.Atoi(r.FormValue("severity_level"))

		// Validasi sederhana
		if name != "" {
			err := models.CreateRiskConsequence(rcc.DB, name, description, reportable, sequence, severity)
			if err != nil {
				rcc.setFlash(w, "Gagal menyimpan data: "+err.Error(), "error")
			} else {
				rcc.setFlash(w, "Data Consequence berhasil disimpan", "success")
			}
		}
	}

	http.Redirect(w, r, "/administration/risk/consequence", http.StatusSeeOther)
}
func (rcc *RiskConsequenceController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		name := r.FormValue("name")
		desc := r.FormValue("description")
		report := r.FormValue("reportable")
		seq, _ := strconv.Atoi(r.FormValue("sequence"))
		sev, _ := strconv.Atoi(r.FormValue("severity_level"))

		if id != "" && name != "" {
			rcc.DB.Model(&models.RiskConsequence{}).Where("id = ?", id).Updates(map[string]interface{}{
				"name":           name,
				"description":    desc,
				"reportable":     report,
				"sequence":       seq,
				"severity_level": sev,
			})
		}
	}
	rcc.setFlash(w, "Data berhasil diupdate", "success")
	http.Redirect(w, r, "/administration/risk/consequence", http.StatusSeeOther)
}

func (rcc *RiskConsequenceController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		rcc.DB.Unscoped().Delete(&models.RiskConsequence{}, id)
	}
	rcc.setFlash(w, "Data berhasil dihapus", "success")
	http.Redirect(w, r, "/administration/risk/consequence", http.StatusSeeOther)
}

func (rcc *RiskConsequenceController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/administration/risk/consequence", http.StatusSeeOther)
		return
	}

	file, _, err := r.FormFile("excel_file")
	if err != nil {
		http.Error(w, "Gagal mengambil file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		http.Error(w, "Format file tidak didukung", http.StatusBadRequest)
		return
	}
	defer f.Close()

	rows, _ := f.GetRows(f.GetSheetName(0))

	for i, row := range rows {
		if i == 0 || len(row) < 1 {
			continue
		}

		// Mapping: A=Name, B=Reportable, C=Description, D=Sequence, E=SeverityLevel
		name := strings.TrimSpace(row[0])
		report := ""
		if len(row) > 1 {
			report = strings.TrimSpace(row[1])
		}
		desc := ""
		if len(row) > 2 {
			desc = strings.TrimSpace(row[2])
		}
		seq := 0
		if len(row) > 3 {
			seq, _ = strconv.Atoi(strings.TrimSpace(row[3]))
		}
		sev := 0
		if len(row) > 4 {
			sev, _ = strconv.Atoi(strings.TrimSpace(row[4]))
		}

		if name != "" {
			rcc.DB.Where(models.RiskConsequence{Name: name}).
				Assign(models.RiskConsequence{
					Reportable:    report,
					Description:   desc,
					Sequence:      seq,
					SeverityLevel: sev,
				}).
				FirstOrCreate(&models.RiskConsequence{})
		}
	}

	rcc.setFlash(w, "Data Consequence berhasil diupload", "success")
	http.Redirect(w, r, "/administration/risk/consequence", http.StatusSeeOther)
}

// Helper untuk flash message
func (rlc *RiskConsequenceController) setFlash(w http.ResponseWriter, msg string, msgType string) {
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
