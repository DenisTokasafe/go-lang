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

type RoleController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (rc *RoleController) Index(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil parameter 'page' dan 'search' dari URL
	querySearch := r.URL.Query().Get("search")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	// 2. Tentukan ukuran data per halaman
	pageSize := 20
	offset := (page - 1) * pageSize

	var roles []models.Role
	var totalRows int64

	// 3. Inisialisasi Query Dasar
	// Gunakan objek query yang sama agar filter 'search' konsisten
	dbQuery := rc.DB.Model(&models.Role{})

	// 4. Terapkan Filter Search jika ada
	if querySearch != "" {
		// Mencari berdasarkan nama perusahaan (Case Insensitive di MySQL)
		dbQuery = dbQuery.Where("name LIKE ?", "%"+querySearch+"%")
	}

	// 5. Hitung total baris berdasarkan filter
	dbQuery.Count(&totalRows)

	// 6. Ambil data dengan limit, offset, dan filter yang sama
	dbQuery.Limit(pageSize).Offset(offset).Order("id DESC").Find(&roles)

	// 7. Hitung total halaman
	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))

	// 8. Kirim data ke template (Sertakan 'Search' agar link pagination bisa sinkron)
	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":        lang,
		"Tr":          helpers.Translations[lang],
		"Roles":       roles,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Search":      querySearch, // Kunci utama untuk sinkronisasi link
		"HasNext":     page < totalPages,
		"HasPrev":     page > 1,
		"TotalRows":   totalRows,
	}

	rc.Render(w, r, "/administration/role/index.html", data)
}
func (rc *RoleController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		if name != "" {
			models.CreateRole(rc.DB, name)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg", // Sesuaikan dengan Alpine.js
		Value: "Data berhasil ditambahkan",
		Path:  "/",
	})
	// Tambahkan juga flash_type agar warnanya sesuai tema DaisyUI
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "success",
		Path:  "/",
	})
	http.Redirect(w, r, "/administration/role", http.StatusSeeOther)
}
func (rc *RoleController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		name := r.FormValue("name")

		if id != "" && name != "" {
			// Update data menggunakan GORM
			rc.DB.Model(&models.Role{}).Where("id = ?", id).Update("name", name)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg", // Sesuaikan dengan Alpine.js
		Value: "Data berhasil diupdate",
		Path:  "/",
	})
	// Tambahkan juga flash_type agar warnanya sesuai tema DaisyUI
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "success",
		Path:  "/",
	})
	http.Redirect(w, r, "/administration/role", http.StatusSeeOther)
}
func (rc *RoleController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		// Gunakan Unscoped() agar data dihapus permanen dari tabel
		rc.DB.Unscoped().Delete(&models.Role{}, id)
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg", // Sesuaikan dengan Alpine.js
		Value: "Data berhasil dihapus",
		Path:  "/",
	})
	// Tambahkan juga flash_type agar warnanya sesuai tema DaisyUI
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "success",
		Path:  "/",
	})

	http.Redirect(w, r, "/administration/role", http.StatusSeeOther)
}
func (rc *RoleController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/administration/role", http.StatusSeeOther)
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

	// 3. Baca sheet pertama (Index 0)
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

		roleName := strings.TrimSpace(row[0])
		if roleName != "" {
			// Ini akan mencari data berdasarkan nama,
			// jika tidak ada baru dibuatkan (tanpa bikin error duplicate)
			rc.DB.Where(models.Role{Name: roleName}).FirstOrCreate(&models.Role{})
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg", // Sesuaikan dengan Alpine.js
		Value: "Data berhasil diupload",
		Path:  "/",
	})
	// Tambahkan juga flash_type agar warnanya sesuai tema DaisyUI
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "success",
		Path:  "/",
	})

	http.Redirect(w, r, "/administration/role", http.StatusSeeOther)
}
