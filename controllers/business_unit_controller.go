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

type BusinessUnitController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (buc *BusinessUnitController) Index(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil parameter 'page' dan 'search' dari URL
	querySearch := r.URL.Query().Get("search")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	// 2. Tentukan ukuran data per halaman
	pageSize := 20
	offset := (page - 1) * pageSize

	var units []models.BusinessUnit
	var totalRows int64

	// 3. Inisialisasi Query Dasar
	dbQuery := buc.DB.Model(&models.BusinessUnit{})

	// 4. Terapkan Filter Search (Mencari di nama unit)
	if querySearch != "" {
		dbQuery = dbQuery.Where("name LIKE ?", "%"+querySearch+"%")
	}

	// 5. Hitung total baris berdasarkan filter
	dbQuery.Count(&totalRows)

	// 6. Ambil data dengan Preload Company
	// Kita tambahkan Preload("Company") agar bisa menampilkan nama perusahaan di tabel
	dbQuery.Preload("Company").
		Limit(pageSize).
		Offset(offset).
		Order("id DESC").
		Find(&units)

	// 7. Ambil daftar semua perusahaan untuk Dropdown di Modal
	var allCompanies []models.Company
	buc.DB.Order("name ASC").Find(&allCompanies)

	// 8. Hitung total halaman
	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))

	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":          lang,
		"Tr":            helpers.Translations[lang],
		"BusinessUnits": units,        // Data utama tabel
		"Companies":     allCompanies, // Untuk select dropdown di modal
		"CurrentPage":   page,
		"TotalPages":    totalPages,
		"Search":        querySearch,
		"HasNext":       page < totalPages,
		"HasPrev":       page > 1,
		"TotalRows":     totalRows,
		"Title":         "Business Unit Management",
		"CurrentPath":   r.URL.Path,
	}

	// Sesuaikan dengan method Render di controller kamu
	buc.Render(w, r, "/administration/business_unit/index.html", data)
}
func (buc *BusinessUnitController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		companyIDStr := r.FormValue("company_id")

		if name != "" && companyIDStr != "" {
			// Konversi company_id dari string ke uint
			companyID, err := strconv.ParseUint(companyIDStr, 10, 32)
			if err == nil {
				// Panggil fungsi model yang sudah kita buat sebelumnya
				models.CreateBusinessUnit(buc.DB, name, uint(companyID))

				// Set notifikasi sukses
				http.SetCookie(w, &http.Cookie{
					Name:  "flash_msg",
					Value: "Business Unit berhasil ditambahkan",
					Path:  "/",
				})
				http.SetCookie(w, &http.Cookie{
					Name:  "flash_type",
					Value: "success",
					Path:  "/",
				})
			} else {
				// Jika konversi ID gagal
				http.SetCookie(w, &http.Cookie{
					Name:  "flash_msg",
					Value: "ID Perusahaan tidak valid",
					Path:  "/",
				})
				http.SetCookie(w, &http.Cookie{
					Name:  "flash_type",
					Value: "error",
					Path:  "/",
				})
			}
		}
	}

	// Redirect kembali ke halaman index Business Unit
	http.Redirect(w, r, "/administration/business-unit", http.StatusSeeOther)
}
func (buc *BusinessUnitController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Ambil ID dari form hidden input
		id := r.FormValue("id")
		name := r.FormValue("name")
		companyIDStr := r.FormValue("company_id")

		if id != "" && name != "" && companyIDStr != "" {
			// Konversi company_id ke uint
			companyID, err := strconv.ParseUint(companyIDStr, 10, 32)
			if err == nil {
				// Panggil fungsi model Update yang sudah kita buat sebelumnya
				errUpdate := models.UpdateBusinessUnit(buc.DB, id, name, uint(companyID))

				if errUpdate != nil {
					// Jika gagal update di database
					http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal memperbarui data", Path: "/"})
					http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
				} else {
					// Jika sukses
					http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Business Unit berhasil diperbarui", Path: "/"})
					http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "success", Path: "/"})
				}
			}
		}
	}

	// Redirect kembali ke index
	http.Redirect(w, r, "/administration/business-unit", http.StatusSeeOther)
}
func (buc *BusinessUnitController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		// Hapus permanen dari tabel business_units
		buc.DB.Unscoped().Delete(&models.BusinessUnit{}, id)
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "flash_msg",
		Value: "Business Unit berhasil dihapus",
		Path:  "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "flash_type",
		Value: "error", // Warna merah karena aksi hapus
		Path:  "/",
	})

	http.Redirect(w, r, "/administration/business-unit", http.StatusSeeOther)
}
func (buc *BusinessUnitController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/administration/business-unit", http.StatusSeeOther)
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

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		http.Error(w, "Gagal membaca data", http.StatusInternalServerError)
		return
	}

	count := 0
	for i, row := range rows {
		// Lewati header dan pastikan kolom unit & perusahaan ada
		if i == 0 || len(row) < 2 {
			continue
		}

		unitName := strings.TrimSpace(row[0])
		companyName := strings.TrimSpace(row[1])

		if unitName != "" && companyName != "" {
			var company models.Company
			// Cari perusahaan berdasarkan nama
			if err := buc.DB.Where("name = ?", companyName).First(&company).Error; err == nil {
				// Jika perusahaan ditemukan, buat/update business unit
				buc.DB.Where(models.BusinessUnit{
					Name:      unitName,
					CompanyID: company.ID,
				}).FirstOrCreate(&models.BusinessUnit{})
				count++
			}
		}
	}

	msg := "Berhasil mengupload " + strconv.Itoa(count) + " Business Unit"
	http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: msg, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "success", Path: "/"})

	http.Redirect(w, r, "/administration/business-unit", http.StatusSeeOther)
}
