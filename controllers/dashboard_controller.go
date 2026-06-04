package controllers

import (
	"encoding/json"
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
	// 🔥 FIX: Ditambahkan string kosong "" sebagai argumen default agar lolos kompilasi
	rawRange := r.URL.Query().Get("hazard_period")
	data, err := c.Service.GetHazardSummary(rawRange)
	if err != nil {
		http.Error(w, "Gagal memproses data dashboard: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return data ke frontend sebagai JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}
