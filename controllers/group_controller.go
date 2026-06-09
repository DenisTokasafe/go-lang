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

type GroupController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (gc *GroupController) Index(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil parameter 'page' dan 'search' dari URL
	querySearch := r.URL.Query().Get("search")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	// 2. Tentukan ukuran data per halaman
	pageSize := 20
	offset := (page - 1) * pageSize

	var groups []models.Group
	var totalRows int64

	// 3. Inisialisasi Query Dasar
	dbQuery := gc.DB.Model(&models.Group{})

	// 4. Terapkan Filter Search jika ada
	if querySearch != "" {
		dbQuery = dbQuery.Where("name LIKE ?", "%"+querySearch+"%")
	}

	// 5. Hitung total baris berdasarkan filter
	dbQuery.Count(&totalRows)

	// 6. Ambil data dengan limit, offset, dan filter yang sama
	dbQuery.Limit(pageSize).Offset(offset).Order("id DESC").Find(&groups)

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
		"Groups":      groups,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Search":      querySearch,
		"HasNext":     page < totalPages,
		"HasPrev":     page > 1,
		"TotalRows":   totalRows,
	}

	gc.Render(w, r, "/administration/group/index.html", data)
}

func (gc *GroupController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		if name != "" {
			models.CreateGroup(gc.DB, name)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg",
		Value: "Data berhasil ditambahkan",
		Path:  "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "success",
		Path:  "/",
	})
	http.Redirect(w, r, "/administration/department-group/group", http.StatusSeeOther)
}

func (gc *GroupController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		name := r.FormValue("name")

		if id != "" && name != "" {
			gc.DB.Model(&models.Group{}).Where("id = ?", id).Update("name", name)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg",
		Value: "Data berhasil diupdate",
		Path:  "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "success",
		Path:  "/",
	})
	http.Redirect(w, r, "/administration/department-group/group", http.StatusSeeOther)
}

func (gc *GroupController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		gc.DB.Unscoped().Delete(&models.Group{}, id)
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg",
		Value: "Data berhasil dihapus",
		Path:  "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "success",
		Path:  "/",
	})

	http.Redirect(w, r, "/administration/department-group/group", http.StatusSeeOther)
}

func (gc *GroupController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/administration/department-group/group", http.StatusSeeOther)
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

		groupName := strings.TrimSpace(row[0])
		if groupName != "" {
			gc.DB.Where(models.Group{Name: groupName}).FirstOrCreate(&models.Group{})
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg",
		Value: "Data berhasil diupload",
		Path:  "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "success",
		Path:  "/",
	})

	http.Redirect(w, r, "/administration/department-group/group", http.StatusSeeOther)
}
