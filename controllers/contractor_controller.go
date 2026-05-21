package controllers

import (
	"encoding/csv"
	"io"
	"latihan1/models" // Sesuaikan dengan nama module di go.mod Anda
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ContractorController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (cc *ContractorController) Index(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil parameter 'page' dan 'search' dari URL
	querySearch := r.URL.Query().Get("search")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	// 2. Tentukan ukuran data per halaman
	pageSize := 20
	offset := (page - 1) * pageSize

	var contractors []models.Contractor
	var totalRows int64

	// 3. Inisialisasi Query Dasar
	// Gunakan objek query yang sama agar filter 'search' konsisten
	dbQuery := cc.DB.Model(&models.Contractor{})

	// 4. Terapkan Filter Search jika ada
	if querySearch != "" {
		// Mencari berdasarkan nama perusahaan (Case Insensitive di MySQL)
		dbQuery = dbQuery.Where("name LIKE ?", "%"+querySearch+"%")
	}

	// 5. Hitung total baris berdasarkan filter
	dbQuery.Count(&totalRows)

	// 6. Ambil data dengan limit, offset, dan filter yang sama
	dbQuery.Limit(pageSize).Offset(offset).Order("id DESC").Find(&contractors)

	// 7. Hitung total halaman
	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))

	// 8. Kirim data ke template (Sertakan 'Search' agar link pagination bisa sinkron)
	data := map[string]interface{}{
		"Contractors": contractors,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Search":      querySearch, // Kunci utama untuk sinkronisasi link
		"HasNext":     page < totalPages,
		"HasPrev":     page > 1,
		"TotalRows":   totalRows,
	}

	cc.Render(w, r, "/administration/contractor/index.html", data)
}

func (cc *ContractorController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		if name != "" {
			models.CreateContractor(cc.DB, name)
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
	http.Redirect(w, r, "/administration/contractor", http.StatusSeeOther)
}
func (cc *ContractorController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		name := r.FormValue("name")

		if id != "" && name != "" {
			// Update data menggunakan GORM
			cc.DB.Model(&models.Contractor{}).Where("id = ?", id).Update("name", name)
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
	http.Redirect(w, r, "/administration/contractor", http.StatusSeeOther)
}

func (cc *ContractorController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		// Gunakan Unscoped() agar data dihapus permanen dari tabel
		cc.DB.Unscoped().Delete(&models.Contractor{}, id)
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg", // Sesuaikan dengan Alpine.js
		Value: "Data berhasil dihapus",
		Path:  "/",
	})
	// Tambahkan juga flash_type agar warnanya sesuai tema DaisyUI
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "error",
		Path:  "/",
	})

	http.Redirect(w, r, "/administration/contractor", http.StatusSeeOther)
}
func (cc *ContractorController) UploadExcel(
	w http.ResponseWriter,
	r *http.Request,
) {

	if r.Method != http.MethodPost {

		http.Redirect(
			w,
			r,
			"/administration/contractor",
			http.StatusSeeOther,
		)

		return
	}

	// =========================
	// AMBIL FILE
	// =========================
	file, header, err := r.FormFile("excel_file")

	if err != nil {

		http.Error(
			w,
			"Gagal mengambil file",
			http.StatusBadRequest,
		)

		return
	}

	defer file.Close()

	ext := strings.ToLower(
		filepath.Ext(header.Filename),
	)

	// =========================
	// HANDLE CSV
	// =========================
	if ext == ".csv" {

		reader := csv.NewReader(file)

		rowIndex := 0

		for {

			row, err := reader.Read()

			if err == io.EOF {
				break
			}

			if err != nil {

				http.Error(
					w,
					"Gagal membaca file CSV",
					http.StatusBadRequest,
				)

				return
			}

			// Skip Header
			if rowIndex == 0 {
				rowIndex++
				continue
			}

			if len(row) < 1 {
				continue
			}

			companyName := strings.TrimSpace(row[0])

			if companyName != "" {

				cc.DB.
					Where(
						models.Contractor{
							Name: companyName,
						},
					).
					FirstOrCreate(
						&models.Contractor{},
					)
			}

			rowIndex++
		}

	} else {

		// =========================
		// HANDLE EXCEL
		// =========================
		f, err := excelize.OpenReader(file)

		if err != nil {

			http.Error(
				w,
				"Format file tidak didukung",
				http.StatusBadRequest,
			)

			return
		}

		defer f.Close()

		sheetName := f.GetSheetName(0)

		rows, err := f.GetRows(sheetName)

		if err != nil {

			http.Error(
				w,
				"Gagal membaca data",
				http.StatusInternalServerError,
			)

			return
		}

		// =========================
		// LOOP DATA
		// =========================
		for i, row := range rows {

			if i == 0 || len(row) < 1 {
				continue
			}

			companyName := strings.TrimSpace(row[0])

			if companyName != "" {

				cc.DB.
					Where(
						models.Contractor{
							Name: companyName,
						},
					).
					FirstOrCreate(
						&models.Contractor{},
					)
			}
		}
	}

	// =========================
	// FLASH SUCCESS
	// =========================
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

	http.Redirect(
		w,
		r,
		"/administration/contractor",
		http.StatusSeeOther,
	)
}
