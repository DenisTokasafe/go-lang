package controllers

import (
	"encoding/json"
	"latihan1/cmd/web/helpers"
	"latihan1/services" // Sesuaikan dengan nama modul Anda
	"net/http"

	"gorm.io/gorm"
)

type DashboardController struct {
	DB      *gorm.DB
	Service *services.DashboardService
	Render  interface{} // Tipe data disesuaikan dengan helper render Anda
}

func (c *DashboardController) GetSummary(w http.ResponseWriter, r *http.Request) {
	rawRange := r.URL.Query().Get("hazard_period")

	// 1. Ambil data utama dari service
	summaryData, err := c.Service.GetHazardSummary(rawRange)
	if err != nil {
		http.Error(w, "Gagal memproses data dashboard: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Deteksi bahasa (karena API ini dipanggil client-side,
	// ia perlu tahu bahasa user untuk mengirimkan string yang tepat)
	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	// 3. Bungkus data utama DAN data terjemahan
	finalPayload := map[string]interface{}{
		"Summary": summaryData,
		"Lang":    lang,
		"Tr":      helpers.Translations[lang],
	}

	// 4. Return sebagai JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalPayload)
}
