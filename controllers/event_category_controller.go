package controllers

import (
	"latihan1/cmd/web/helpers"
	"latihan1/models" // Sesuaikan dengan nama module di go.mod Anda
	"math"
	"net/http"
	"strconv"

	"gorm.io/gorm"
)

type EventCategoryController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (ecc *EventCategoryController) Index(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil parameter filter dari URL
	querySearch := r.URL.Query().Get("search")
	queryGroup := r.URL.Query().Get("group")   // Filter: lead atau incident
	queryParent := r.URL.Query().Get("parent") // Filter berdasarkan Parent ID

	// Default Group jika kosong (misal kita arahkan ke lead/hazard dulu)
	if queryGroup == "" {
		queryGroup = "lead"
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	// 2. Konfigurasi Pagination
	pageSize := 20
	offset := (page - 1) * pageSize

	var eventCategories []models.EventCategory
	var totalRows int64

	// 3. Inisialisasi Query Dasar
	dbQuery := ecc.DB.Model(&models.EventCategory{})

	// 4. Terapkan Filter Category Group (Lead/Incident)
	// Ini wajib agar data tidak bercampur antara Hazard dan Incident
	dbQuery = dbQuery.Where("category_group = ?", queryGroup)

	// 5. Terapkan Filter Parent (Jika ingin melihat sub-type dari parent tertentu)
	if queryParent != "" {
		dbQuery = dbQuery.Where("parent_id = ?", queryParent)
	}

	// 6. Terapkan Filter Pencarian Teks (Search)
	if querySearch != "" {
		// Mencari di kolom name atau code
		dbQuery = dbQuery.Where(
			"(name LIKE ? OR code LIKE ?)",
			"%"+querySearch+"%", "%"+querySearch+"%",
		)
	}

	// 7. Hitung Total Baris (untuk Pagination) sebelum Limit/Offset
	dbQuery.Count(&totalRows)

	// 8. Ambil Data dengan urutan numerik (Natural Sort)
	// Kita gunakan Preload("SubCategories") jika ingin menampilkan tree di index,
	// tapi untuk tabel flat seperti BodyPart, kita cukup Find saja.
	dbQuery.Limit(pageSize).
		Offset(offset).
		Order("LENGTH(code) ASC, code ASC").
		Find(&eventCategories)

	// 9. Hitung Total Halaman
	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))

	// 10. Ambil daftar Parent untuk dropdown filter di View (Opsional)
	var parents []models.EventCategory
	ecc.DB.Where("category_group = ? AND parent_id IS NULL", queryGroup).Find(&parents)

	// 11. Siapkan Data untuk Template
	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":            lang,
		"Tr":              helpers.Translations[lang],
		"EventCategories": eventCategories,
		"Parents":         parents, // Untuk dropdown filter parent di UI
		"CurrentPage":     page,
		"TotalPages":      totalPages,
		"Search":          querySearch,
		"GroupFilter":     queryGroup,  // Untuk navigasi tab Lead/Incident
		"ParentFilter":    queryParent, // Agar filter parent tetap terpilih
		"HasNext":         page < totalPages,
		"HasPrev":         page > 1,
		"TotalRows":       totalRows,
	}

	ecc.Render(w, r, "/administration/event-category/index.html", data)
}
func (ecc *EventCategoryController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// 1. Ambil nilai dari form
		code := r.FormValue("code")
		name := r.FormValue("name")
		group := r.FormValue("category_group") // 'lead' atau 'incident'
		status := r.FormValue("status")        // 'enabled' atau 'disabled'
		parentIDRaw := r.FormValue("parent_id")

		// 2. Handling Nullable untuk ParentID
		// Jika parent_id kosong, berarti ini adalah 'Event Type' (Induk)
		// Jika berisi ID, berarti ini adalah 'Sub Event Type' (Anak)
		var parentID *uint
		if parentIDRaw != "" {
			idUint, err := strconv.ParseUint(parentIDRaw, 10, 32)
			if err == nil {
				u := uint(idUint)
				parentID = &u
			}
		}

		// 3. Validasi minimal dan eksekusi Create
		if code != "" && name != "" && group != "" {
			err := models.CreateEventCategory(ecc.DB, parentID, group, name, code, status)

			if err != nil {
				ecc.setFlash(w, "Gagal menambahkan data: "+err.Error(), "error")
			} else {
				ecc.setFlash(w, "Data Kategori berhasil ditambahkan", "success")
			}
		} else {
			ecc.setFlash(w, "Gagal: Kode, Nama, dan Grup wajib diisi", "error")
		}
	}

	// 4. Redirect kembali ke halaman index dengan membawa parameter group agar tab tidak berpindah
	groupRedirect := r.FormValue("category_group")
	if groupRedirect == "" {
		groupRedirect = "lead"
	}

	http.Redirect(w, r, "/administration/event-category?group="+groupRedirect, http.StatusSeeOther)
}
func (ecc *EventCategoryController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// 1. Ambil nilai dari form
		id := r.FormValue("id")
		code := r.FormValue("code")
		name := r.FormValue("name")
		group := r.FormValue("category_group")
		status := r.FormValue("status")
		parentIDRaw := r.FormValue("parent_id")

		// 2. Handling Nullable untuk ParentID
		var parentID *uint
		if parentIDRaw != "" {
			idUint, err := strconv.ParseUint(parentIDRaw, 10, 32)
			if err == nil {
				u := uint(idUint)
				parentID = &u
			}
		}

		// 3. Validasi ID dan field wajib lainnya
		if id != "" && code != "" && name != "" && group != "" {
			err := models.UpdateEventCategory(ecc.DB, id, parentID, group, name, code, status)

			if err != nil {
				ecc.setFlash(w, "Gagal mengupdate data: "+err.Error(), "error")
			} else {
				ecc.setFlash(w, "Data Kategori berhasil diupdate", "success")
			}
		} else {
			ecc.setFlash(w, "Gagal: ID, Kode, Nama, dan Grup wajib diisi", "error")
		}
	}

	// 4. Redirect kembali dengan mempertahankan filter group agar posisi tab konsisten
	groupRedirect := r.FormValue("category_group")
	if groupRedirect == "" {
		groupRedirect = "lead"
	}

	http.Redirect(w, r, "/administration/event-category?group="+groupRedirect, http.StatusSeeOther)
}
func (ecc *EventCategoryController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	// Ambil group sebelum data dihapus untuk keperluan redirect
	var category models.EventCategory
	groupRedirect := "lead" // Default redirect

	if id != "" {
		// Cari data untuk mengetahui group-nya (lead/incident)
		if err := ecc.DB.Select("category_group").First(&category, id).Error; err == nil {
			groupRedirect = category.CategoryGroup
		}

		// Hapus permanen menggunakan Unscoped
		err := ecc.DB.Unscoped().Delete(&models.EventCategory{}, id).Error
		if err != nil {
			ecc.setFlash(w, "Gagal menghapus data: "+err.Error(), "error")
		} else {
			ecc.setFlash(w, "Data Kategori berhasil dihapus", "success")
		}
	}

	// Redirect kembali ke group yang sesuai agar user tidak bingung
	http.Redirect(w, r, "/administration/event-category?group="+groupRedirect, http.StatusSeeOther)
}

// Helper untuk menghindari pengulangan penulisan cookie
func (bpc *EventCategoryController) setFlash(w http.ResponseWriter, msg, msgType string) {
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
