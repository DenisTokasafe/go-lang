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

type BodyPartController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (bpc *BodyPartController) Index(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil parameter filter dari URL
	querySearch := r.URL.Query().Get("search")
	queryCategory := r.URL.Query().Get("category") // Filter berdasarkan kategori (select)

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	// 2. Konfigurasi Pagination
	pageSize := 20
	offset := (page - 1) * pageSize

	var bodyParts []models.BodyPart
	var totalRows int64

	// 3. Inisialisasi Query Dasar
	dbQuery := bpc.DB.Model(&models.BodyPart{})

	// 4. Terapkan Filter Pencarian Teks (Search)
	if querySearch != "" {
		// Mencari di kolom name, name_en, atau code
		dbQuery = dbQuery.Where(
			"(name LIKE ? OR name_en LIKE ? OR code LIKE ?)",
			"%"+querySearch+"%", "%"+querySearch+"%", "%"+querySearch+"%",
		)
	}

	// 5. Terapkan Filter Kategori (Select)
	if queryCategory != "" {
		dbQuery = dbQuery.Where("category = ?", queryCategory)
	}

	// 6. Hitung Total Baris (untuk Pagination)
	dbQuery.Count(&totalRows)

	// 7. Ambil Data dengan urutan numerik (Natural Sort)
	dbQuery.Limit(pageSize).
		Offset(offset).
		Order("LENGTH(code) ASC, code ASC").
		Find(&bodyParts)

	// 8. Hitung Total Halaman
	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))

	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":           lang,
		"Tr":             helpers.Translations[lang],
		"BodyParts":      bodyParts,
		"CurrentPage":    page,
		"TotalPages":     totalPages,
		"Search":         querySearch,
		"CategoryFilter": queryCategory, // Dikirim balik agar select tidak reset
		"HasNext":        page < totalPages,
		"HasPrev":        page > 1,
		"TotalRows":      totalRows,
	}

	bpc.Render(w, r, "/administration/body-part/index.html", data)
}
func (bpc *BodyPartController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		code := r.FormValue("code")
		name := r.FormValue("name")
		category := r.FormValue("category")
		nameEnRaw := r.FormValue("name_en")

		// Handling Nullable untuk name_en
		var nameEn *string
		if nameEnRaw != "" {
			nameEn = &nameEnRaw
		}

		if code != "" && name != "" && category != "" {
			models.CreateBodyPart(bpc.DB, code, name, category, nameEn)
		}
	}

	// Set Flash Message
	bpc.setFlash(w, "Data Body Part berhasil ditambahkan", "success")
	http.Redirect(w, r, "/administration/body-part", http.StatusSeeOther)
}
func (bpc *BodyPartController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		code := r.FormValue("code")
		name := r.FormValue("name")
		category := r.FormValue("category")
		nameEnRaw := r.FormValue("name_en")

		var nameEn *string
		if nameEnRaw != "" {
			nameEn = &nameEnRaw
		}

		if id != "" && code != "" && name != "" && category != "" {
			models.UpdateBodyPart(bpc.DB, id, code, name, category, nameEn)
		}
	}

	// Set Flash Message
	bpc.setFlash(w, "Data Body Part berhasil diupdate", "success")
	http.Redirect(w, r, "/administration/body-part", http.StatusSeeOther)
}
func (bpc *BodyPartController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		// Hapus permanen menggunakan Unscoped
		bpc.DB.Unscoped().Delete(&models.BodyPart{}, id)
	}

	bpc.setFlash(w, "Data Body Part berhasil dihapus", "success")
	http.Redirect(w, r, "/administration/body-part", http.StatusSeeOther)
}
func (bpc *BodyPartController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/administration/body-part", http.StatusSeeOther)
		return
	}

	// 1. Ambil file dari form
	file, _, err := r.FormFile("excel_file")
	if err != nil {
		bpc.setFlash(w, "Gagal mengambil file", "error")
		http.Redirect(w, r, "/administration/body-part", http.StatusSeeOther)
		return
	}
	defer file.Close()

	// 2. Buka file excel menggunakan excelize
	f, err := excelize.OpenReader(file)
	if err != nil {
		bpc.setFlash(w, "Format file tidak didukung", "error")
		http.Redirect(w, r, "/administration/body-part", http.StatusSeeOther)
		return
	}
	defer f.Close()

	// 3. Baca sheet pertama
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		bpc.setFlash(w, "Gagal membaca data excel", "error")
		http.Redirect(w, r, "/administration/body-part", http.StatusSeeOther)
		return
	}

	// 4. Looping data (Lewati header)
	// Asumsi urutan kolom Excel:
	// [0] Code, [1] Name (ID), [2] NameEn (EN), [3] Category
	for i, row := range rows {
		// Lewati baris pertama (header) dan pastikan kolom minimal terpenuhi
		if i == 0 || len(row) < 4 {
			continue
		}

		code := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])
		nameEnRaw := strings.TrimSpace(row[2])
		category := strings.TrimSpace(row[3])

		if code != "" && name != "" && category != "" {
			var nameEn *string
			if nameEnRaw != "" {
				nameEn = &nameEnRaw
			}

			// Menggunakan FirstOrCreate berdasarkan Code
			// Agar data dengan Code yang sama tidak duplikat
			bpc.DB.Where(models.BodyPart{Code: code}).
				Attrs(models.BodyPart{
					Name:     name,
					NameEn:   nameEn,
					Category: category,
				}).
				FirstOrCreate(&models.BodyPart{})
		}
	}

	bpc.setFlash(w, "Data Body Part berhasil diimport", "success")
	http.Redirect(w, r, "/administration/body-part", http.StatusSeeOther)
}

// Helper untuk menghindari pengulangan penulisan cookie
func (bpc *BodyPartController) setFlash(w http.ResponseWriter, msg, msgType string) {
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
