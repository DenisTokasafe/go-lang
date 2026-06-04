package services

import (
	"latihan1/models" // Pastikan import ke model Anda sudah benar
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

type DashboardService struct {
	DB *gorm.DB
}

type LocationTrend struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type ScatPieData struct {
	Type            string `json:"type"`              // unsafe_act atau personal_factor
	EventCategoryID uint   `json:"event_category_id"` // Tambahan baru
	CategoryName    string `json:"category_name"`     // Tambahan baru
	Total           int64  `json:"total"`             // Jumlah temuan
}
type CategoryBarData struct {
	Month        string `json:"month"`
	CategoryName string `json:"category_name"`
	Total        int64  `json:"total"`
}

type KPIData struct {
	Submit     int64 `json:"submit"`
	InProgress int64 `json:"in_progress"`
	Pending    int64 `json:"pending"`
	Closed     int64 `json:"closed"`
}

type MonthlyTrend struct {
	Month string `json:"month"`
	Count int64  `json:"count"`
}

type RecentHazard struct {
	TanggalWaktu string `json:"tanggal_waktu"`
	LocationName string `json:"location_name"`
	DeptName     string `json:"dept_name"`
	Description  string `json:"description"`
	Mitigation   string `json:"mitigation"`
	Status       string `json:"status"`
}

type HazardSummaryResult struct {
	KPI                KPIData           `json:"kpi"`
	MonthlyTrend       []MonthlyTrend    `json:"monthly_trend"`
	TopLocations       []LocationTrend   `json:"top_locations"`
	RecentHazards      []RecentHazard    `json:"recent_hazards"`
	ScatPieData        []ScatPieData     `json:"scat_pie_data"`
	EnvScatPieData     []ScatPieData     `json:"env_scat_pie_data"`
	CategoryBarData    []CategoryBarData `json:"category_bar_data"`
	EnvCategoryBarData []CategoryBarData `json:"env_category_bar_data"`
}

func (s *DashboardService) GetHazardSummary(rawRange string) (*HazardSummaryResult, error) {
	var result HazardSummaryResult

	// 1. SET DEFAULT TANGGAL (Format murni YYYY-MM-DD)
	startStr := time.Now().AddDate(0, -6, 0).Format("2006-01-02")
	endStr := time.Now().Format("2006-01-02")

	rawRange = strings.TrimSpace(rawRange)

	if rawRange != "" {
		dates := strings.Split(rawRange, " to ")

		// Ambil tanggal mulai
		if len(dates) >= 1 && strings.TrimSpace(dates[0]) != "" {
			startStr = strings.TrimSpace(dates[0])
		}

		// Ambil tanggal akhir
		if len(dates) == 2 && strings.TrimSpace(dates[1]) != "" {
			endStr = strings.TrimSpace(dates[1])
		} else {
			// Jika user baru klik 1 tanggal di picker, samakan akhir dengan awal
			endStr = startStr
		}
	}
	println("==================================================")
	log.Printf("[DEBUG-HSE] Raw Range dari Frontend : '%s'\n", rawRange)
	log.Printf("[DEBUG-HSE] Start Date untuk Query : '%s'\n", startStr)
	log.Printf("[DEBUG-HSE] End Date untuk Query   : '%s'\n", endStr)
	println("==================================================")

	// Inisialisasi slice kosong agar JSON tidak bernilai null
	result.MonthlyTrend = []MonthlyTrend{}
	result.TopLocations = []LocationTrend{}
	result.RecentHazards = []RecentHazard{}
	result.ScatPieData = []ScatPieData{}
	result.CategoryBarData = []CategoryBarData{}

	// =================================================================
	// 2. EKSEKUSI QUERY DENGAN FUNGSI DATE() MYSQL
	// =================================================================

	// A. KPI Counters
	statuses := []struct {
		Status string
		Dest   *int64
	}{
		{"submit", &result.KPI.Submit},
		{"in_progress", &result.KPI.InProgress},
		{"pending", &result.KPI.Pending},
		{"closed", &result.KPI.Closed},
	}

	for _, item := range statuses {
		// 💡 Perhatikan: "DATE(tanggal_waktu)" memaksa MySQL membandingkan tanggalnya saja
		err := s.DB.Model(&models.Hazard{}).
			Where("status = ? AND DATE(tanggal_waktu) BETWEEN ? AND ?", item.Status, startStr, endStr).
			Count(item.Dest).Error
		if err != nil {
			return nil, err
		}
	}

	// B. Tren Bulanan
	var trendResults []struct {
		Bulan   string
		Total   int64
		SortKey string
	}
	err := s.DB.Model(&models.Hazard{}).
		Select("DATE_FORMAT(tanggal_waktu, '%b') as bulan, COUNT(id) as total, DATE_FORMAT(tanggal_waktu, '%Y-%m') as sort_key").
		Where("DATE(tanggal_waktu) BETWEEN ? AND ?", startStr, endStr).
		Group("DATE_FORMAT(tanggal_waktu, '%b'), DATE_FORMAT(tanggal_waktu, '%Y-%m')").
		Order("sort_key ASC").
		Scan(&trendResults).Error

	if err != nil {
		return nil, err
	}

	for _, res := range trendResults {
		result.MonthlyTrend = append(result.MonthlyTrend, MonthlyTrend{
			Month: res.Bulan,
			Count: res.Total,
		})
	}

	// C. Top 5 Lokasi Kritis
	err = s.DB.Model(&models.Hazard{}).
		Select("locations.name as name, COUNT(hazards.id) as count").
		Joins("INNER JOIN locations ON locations.id = hazards.location_id").
		Where("DATE(hazards.tanggal_waktu) BETWEEN ? AND ?", startStr, endStr).
		Group("locations.name").
		Order("count DESC").
		Limit(5).
		Scan(&result.TopLocations).Error

	if err != nil {
		return nil, err
	}

	// D. Log 5 Temuan Terbaru
	err = s.DB.Model(&models.Hazard{}).
		Select(`
			DATE_FORMAT(hazards.tanggal_waktu, '%Y-%m-%d %H:%i') as tanggal_waktu, 
			locations.name as location_name, 
			departments.name as dept_name, 
			hazards.deskripsi as description, 
			hazards.corrective_action as mitigation, 
			hazards.status
		`).
		Joins("LEFT JOIN locations ON locations.id = hazards.location_id").
		Joins("LEFT JOIN departments ON departments.id = hazards.department_id").
		Where("DATE(hazards.tanggal_waktu) BETWEEN ? AND ?", startStr, endStr).
		Order("hazards.tanggal_waktu DESC").
		Limit(5).
		Scan(&result.RecentHazards).Error

	if err != nil {
		return nil, err
	}

	// E. Pie Chart SCAT (Khusus OHS / Parent ID 1)
	err = s.DB.Model(&models.Hazard{}).
		Select("scat_options.type as type, COUNT(hazards.id) as total"). // 🔥 Kembalikan ke Select awal
		Joins("INNER JOIN scat_options ON scat_options.id = hazards.scat_option_id").
		Joins("INNER JOIN event_categories as child_cat ON child_cat.id = hazards.event_category_id").
		Where("DATE(hazards.tanggal_waktu) BETWEEN ? AND ? AND scat_options.type IN ('unsafe_act', 'personal_factor')", startStr, endStr).
		Where("child_cat.parent_id = ?", 1). // 🔥 Tetap pertahankan filter OHS ini
		Group("scat_options.type").          // 🔥 Hapus grouping berdasarkan kategori
		Scan(&result.ScatPieData).Error

	if err != nil {
		return nil, err
	}

	// 🔥 F. Pie Chart SCAT Baru (Khusus ENV / Parent ID 2)
	err = s.DB.Model(&models.Hazard{}).
		Select("scat_options.type as type, COUNT(hazards.id) as total").
		Joins("INNER JOIN scat_options ON scat_options.id = hazards.scat_option_id").
		Joins("INNER JOIN event_categories as child_cat ON child_cat.id = hazards.event_category_id").
		Where("DATE(hazards.tanggal_waktu) BETWEEN ? AND ? AND scat_options.type IN ('unsafe_act', 'personal_factor')", startStr, endStr).
		Where("child_cat.parent_id = ?", 2). // 🔥 Ubah filter ke 2 (ENV Hazard Report)
		Group("scat_options.type").
		Scan(&result.EnvScatPieData).Error // 🔥 Scan ke struct penampung khusus ENV
	if err != nil {
		return nil, err
	}

	// F. Bar Chart Kategori
	err = s.DB.Model(&models.Hazard{}).
		// PASTI KAN DI SINI MENGGUNAKAN child_cat.name
		Select("DATE_FORMAT(hazards.tanggal_waktu, '%Y-%m') as month, COALESCE(child_cat.name, 'Uncategorized') as category_name, COUNT(hazards.id) as total").
		Joins("INNER JOIN event_categories as child_cat ON child_cat.id = hazards.event_category_id").
		Where("DATE(hazards.tanggal_waktu) BETWEEN ? AND ?", startStr, endStr).
		Where("child_cat.parent_id = ?", 1).
		// PASTIKAN DI SINI JUGA MENGGUNAKAN child_cat.name
		Group("DATE_FORMAT(hazards.tanggal_waktu, '%Y-%m'), child_cat.name").
		Order("DATE_FORMAT(hazards.tanggal_waktu, '%Y-%m') ASC").
		Scan(&result.CategoryBarData).Error

	if err != nil {
		return nil, err
	}
	err = s.DB.Model(&models.Hazard{}).
		// PASTI KAN DI SINI MENGGUNAKAN child_cat.name
		Select("DATE_FORMAT(hazards.tanggal_waktu, '%Y-%m') as month, COALESCE(child_cat.name, 'Uncategorized') as category_name, COUNT(hazards.id) as total").
		Joins("INNER JOIN event_categories as child_cat ON child_cat.id = hazards.event_category_id").
		Where("DATE(hazards.tanggal_waktu) BETWEEN ? AND ?", startStr, endStr).
		Where("child_cat.parent_id = ?", 2).
		// PASTIKAN DI SINI JUGA MENGGUNAKAN child_cat.name
		Group("DATE_FORMAT(hazards.tanggal_waktu, '%Y-%m'), child_cat.name").
		Order("DATE_FORMAT(hazards.tanggal_waktu, '%Y-%m') ASC").
		Scan(&result.EnvCategoryBarData).Error

	if err != nil {
		return nil, err
	}
	return &result, nil
}
