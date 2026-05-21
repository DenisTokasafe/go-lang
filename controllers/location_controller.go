package controllers

import (
	"latihan1/models" // Sesuaikan dengan nama module di go.mod Anda
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type LocationController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (lc *LocationController) Index(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil parameter 'search' dan 'page' dari URL
	querySearch := r.URL.Query().Get("search")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	// 2. Tentukan ukuran data per halaman
	pageSize := 20
	offset := (page - 1) * pageSize

	var locations []models.Location
	var totalRows int64

	// 3. Inisialisasi Query Dasar menggunakan model Location
	dbQuery := lc.DB.Model(&models.Location{})

	// 4. Terapkan Filter Search (Mencari berdasarkan nama lokasi)
	if querySearch != "" {
		dbQuery = dbQuery.Where("name LIKE ?", "%"+querySearch+"%")
	}

	// 5. Hitung total baris berdasarkan filter (sebelum limit/offset)
	dbQuery.Count(&totalRows)

	// 6. Ambil data dengan limit, offset, dan urutan terbaru
	dbQuery.Limit(pageSize).
		Offset(offset).
		Order("id DESC").
		Find(&locations)

	// 7. Hitung total halaman
	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))

	// 8. Kirim data ke template
	data := map[string]interface{}{
		"Locations":   locations, // Data utama untuk tabel
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Search":      querySearch,
		"HasNext":     page < totalPages,
		"HasPrev":     page > 1,
		"TotalRows":   totalRows,
		"Title":       "Location Management",
		"CurrentPath": r.URL.Path,
	}

	// 9. Render ke template location
	lc.Render(w, r, "/administration/location/index.html", data)
}
func (lc *LocationController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		if name != "" {
			// Memanggil fungsi dari models/location.go
			models.CreateLocation(lc.DB, name)
		}
	}

	lc.setFlash(w, "Data berhasil ditambahkan", "success")
	http.Redirect(w, r, "/administration/location", http.StatusSeeOther)
}

func (lc *LocationController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		name := r.FormValue("name")

		if id != "" && name != "" {
			lc.DB.Model(&models.Location{}).Where("id = ?", id).Update("name", name)
		}
	}

	lc.setFlash(w, "Data berhasil diupdate", "success")
	http.Redirect(w, r, "/administration/location", http.StatusSeeOther)
}

func (lc *LocationController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		// Menggunakan Unscoped agar terhapus permanen sesuai contoh Group
		lc.DB.Unscoped().Delete(&models.Location{}, id)
	}

	lc.setFlash(w, "Data berhasil dihapus", "success")
	http.Redirect(w, r, "/administration/location", http.StatusSeeOther)
}
func (lc *LocationController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/administration/location", http.StatusSeeOther)
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

		locationName := strings.TrimSpace(row[0])
		if locationName != "" {
			// Menggunakan FirstOrCreate agar jika nama lokasi sudah ada,
			// sistem tidak akan membuat duplikat.
			lc.DB.Where(models.Location{Name: locationName}).FirstOrCreate(&models.Location{})
		}
	}

	// 5. Set Notifikasi (Flash Message)
	lc.setFlash(w, "Data Lokasi berhasil diupload", "success")

	http.Redirect(w, r, "/administration/location", http.StatusSeeOther)
}

// Fungsi helper agar kode lebih bersih (opsional)
func (lc *LocationController) setFlash(w http.ResponseWriter, msg string, msgType string) {
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
