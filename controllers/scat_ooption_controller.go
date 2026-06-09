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

type ScatOptionController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (soc *ScatOptionController) Index(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil parameter 'page' dan 'search' dari URL
	querySearch := r.URL.Query().Get("search")
	queryType := r.URL.Query().Get("type") // Ambil parameter type dari URL
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	// 2. Tentukan ukuran data per halaman
	pageSize := 20
	offset := (page - 1) * pageSize

	var scatOptions []models.ScatOption
	var totalRows int64

	// 3. Inisialisasi Query Dasar
	dbQuery := soc.DB.Model(&models.ScatOption{})

	// 4. Terapkan Filter Search jika ada
	if querySearch != "" {
		// Mencari berdasarkan nama atau kode (Case Insensitive)
		dbQuery = dbQuery.Where("name LIKE ? OR code LIKE ?", "%"+querySearch+"%", "%"+querySearch+"%")
	}
	// Filter Berdasarkan Kategori (Select)
	if queryType != "" {
		dbQuery = dbQuery.Where("type = ?", queryType)
	}

	// 5. Hitung total baris berdasarkan filter
	dbQuery.Count(&totalRows)

	// 6. Ambil data dengan limit, offset, dan urutan ID terbaru
	// Menggunakan .Order("id DESC") agar data terbaru muncul di atas
	// Di dalam fungsi Index ScatOptionController
	// Di ScatOptionController
	dbQuery.Limit(pageSize).
		Offset(offset).
		Order("LENGTH(code) ASC, code ASC").
		Find(&scatOptions)

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
		"ScatOptions": scatOptions,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Search":      querySearch,
		"TypeFilter":  queryType,
		"HasNext":     page < totalPages,
		"HasPrev":     page > 1,
		"TotalRows":   totalRows,
	}

	soc.Render(w, r, "/administration/scat-option/index.html", data)
}
func (soc *ScatOptionController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		code := r.FormValue("code")
		name := r.FormValue("name")
		optionType := r.FormValue("type") // unsafe_act, personal_factor, dll.

		if code != "" && name != "" && optionType != "" {
			// Memanggil helper CreateScatOption yang sudah kita buat di model
			models.CreateScatOption(soc.DB, code, name, optionType)
		}
	}

	// Set Flash Message
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg",
		Value: "Data SCAT berhasil ditambahkan",
		Path:  "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "success",
		Path:  "/",
	})

	http.Redirect(w, r, "/administration/scat-option", http.StatusSeeOther)
}
func (soc *ScatOptionController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		code := r.FormValue("code")
		name := r.FormValue("name")
		optionType := r.FormValue("type")

		if id != "" && code != "" && name != "" && optionType != "" {
			// Memanggil helper UpdateScatOption yang sudah kita buat di model
			models.UpdateScatOption(soc.DB, id, code, name, optionType)
		}
	}

	// Set Flash Message
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg",
		Value: "Data SCAT berhasil diupdate",
		Path:  "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "success",
		Path:  "/",
	})

	http.Redirect(w, r, "/administration/scat-option", http.StatusSeeOther)
}
func (soc *ScatOptionController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		// Gunakan Unscoped() agar data dihapus permanen dari tabel
		soc.DB.Unscoped().Delete(&models.ScatOption{}, id)
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg",
		Value: "Data SCAT berhasil dihapus",
		Path:  "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "success",
		Path:  "/",
	})

	http.Redirect(w, r, "/administration/scat-option", http.StatusSeeOther)
}
func (soc *ScatOptionController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/administration/scat-option", http.StatusSeeOther)
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

	// 4. Looping data (Lewati header)
	// Asumsi urutan kolom: [0] Code, [1] Name, [2] Type
	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}

		code := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])
		scatType := strings.TrimSpace(row[2])

		if code != "" && name != "" && scatType != "" {
			// Menggunakan FirstOrCreate berdasarkan Code dan Type
			// agar tidak terjadi duplikasi data unik
			soc.DB.Where(models.ScatOption{
				Code: code,
				Type: scatType,
			}).Attrs(models.ScatOption{
				Name: name,
			}).FirstOrCreate(&models.ScatOption{})
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg",
		Value: "Data SCAT berhasil diupload",
		Path:  "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "success",
		Path:  "/",
	})

	http.Redirect(w, r, "/administration/scat-option", http.StatusSeeOther)
}
