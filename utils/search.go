package utils

import (
	"encoding/json"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

// SelectOption adalah format standard JSON untuk Alpine.js
type SelectOption struct {
	ID    uint   `json:"id"`
	Label string `json:"label"`
}

// GlobalSearch menangani pencarian dinamis untuk tabel apa pun di GORM tanpa refleksi lambat
// model mewakili instance struct kosong (misal: models.Location{} atau &models.User{}) untuk menentukan tabel tujuan
func GlobalSearch(db *gorm.DB, w http.ResponseWriter, r *http.Request, model interface{}, searchColumn string) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	// Menggunakan literal slice kosong agar jika kosong, JSON keluar [] bukan null
	result := []SelectOption{}

	if q != "" {
		// Menampung hasil query langsung ke bentuk slice of map
		var rows []map[string]interface{}

		// GORM akan otomatis mendeteksi tabel berdasarkan parameter 'model'
		err := db.Model(model).
			Select("id, "+searchColumn).
			Where("LOWER("+searchColumn+") LIKE ?", "%"+strings.ToLower(q)+"%").
			Order(searchColumn + " asc").
			Limit(100).
			Find(&rows).Error

		// Jika tidak ada error dan data ditemukan
		if err == nil && len(rows) > 0 {
			for _, row := range rows {
				var id uint
				var label string

				// Ambil ID (Database biasanya mengembalikan int64, int32, atau uint)
				if idVal, ok := row["id"]; ok {
					switch v := idVal.(type) {
					case int64:
						id = uint(v)
					case uint:
						id = v
					case int:
						id = uint(v) // FIXED: Menambahkan casting uint(v) agar tidak error compile
					}
				}

				// Ambil Label berdasarkan nama kolom database asli yang di-query
				if labelVal, ok := row[searchColumn].(string); ok {
					label = labelVal
				}

				result = append(result, SelectOption{
					ID:    id,
					Label: label,
				})
			}
		} else {
			// JIKA DATA TIDAK DITEMUKAN (Atau terjadi error database)
			result = append(result, SelectOption{
				ID:    0,
				Label: "Data tidak ditemukan",
			})
		}
	} else {
		// JIKA USER BELUM MENGETIK APAPUN
		result = append(result, SelectOption{
			ID:    0,
			Label: "Ketik untuk mencari...",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
