package controllers

import (
	"encoding/json"
	"errors"
	"latihan1/middlewares"
	"latihan1/models"
	"latihan1/services"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type WpiReportController struct {
	DB      *gorm.DB
	Service *services.WpiReportService
	Render  func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

type WpiUnitFilterOption struct {
	ID   string
	Name string
}

func (c *WpiReportController) Index(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(middlewares.AuthUserKey).(models.User)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	search := strings.TrimSpace(r.URL.Query().Get("search"))
	locationID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	unitFilter := strings.TrimSpace(r.URL.Query().Get("unit"))
	startDate := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endDate := strings.TrimSpace(r.URL.Query().Get("end_date"))
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	result, err := c.Service.GetWpiReports(search, locationID, unitFilter, startDate, endDate, page)
	if err != nil {
		c.setFlash(w, "Gagal memuat data laporan: "+err.Error(), "error")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	locations := locationsForWpiIndex(c.DB)
	unitOptions := unitOptionsForWpiIndex(c.DB)
	data := map[string]interface{}{
		"Reports":     result.Reports,
		"TotalRows":   result.TotalRows,
		"TotalPages":  result.TotalPages,
		"CurrentPage": result.CurrentPage,
		"HasPrev":     result.HasPrev,
		"HasNext":     result.HasNext,
		"Search":      result.Search,
		"LocationID":  result.LocationID,
		"UnitFilter":  result.UnitFilter,
		"StartDate":   startDate,
		"EndDate":     endDate,
		"Locations":   locations,
		"UnitOptions": unitOptions,
	}

	c.Render(w, r, "/wpi_report/index.gohtml", data)
}

func locationsForWpiIndex(db *gorm.DB) []models.Location {
	var rows []models.Location
	// Hanya tampilkan lokasi yang dipakai oleh setidaknya satu laporan WPI.
	db.Model(&models.Location{}).
		Joins("INNER JOIN wpi_reports ON wpi_reports.location_id = locations.id").
		Distinct("locations.*").
		Order("locations.name asc").
		Find(&rows)
	return rows
}

func unitOptionsForWpiIndex(db *gorm.DB) []WpiUnitFilterOption {
	options := make([]WpiUnitFilterOption, 0)

	var departments []models.Department
	db.Model(&models.Department{}).
		Joins("INNER JOIN wpi_reports ON wpi_reports.department_id = departments.id").
		Distinct("departments.*").
		Order("departments.name asc").
		Find(&departments)
	for _, department := range departments {
		options = append(options, WpiUnitFilterOption{
			ID:   "department:" + strconv.FormatUint(uint64(department.ID), 10),
			Name: "Unit Kerja — " + department.Name,
		})
	}

	var contractors []models.Contractor
	db.Model(&models.Contractor{}).
		Joins("INNER JOIN wpi_reports ON wpi_reports.contractor_id = contractors.id").
		Distinct("contractors.*").
		Order("contractors.name asc").
		Find(&contractors)
	for _, contractor := range contractors {
		options = append(options, WpiUnitFilterOption{
			ID:   "contractor:" + strconv.FormatUint(uint64(contractor.ID), 10),
			Name: "Kontraktor — " + contractor.Name,
		})
	}

	return options
}

func (c *WpiReportController) Create(w http.ResponseWriter, r *http.Request) {
	var locations []models.Location
	c.DB.Order("name asc").Limit(100).Find(&locations)

	var companies []models.Company
	c.DB.Order("name asc").Find(&companies)

	var departments []models.Department
	c.DB.Order("name asc").Find(&departments)

	var contractors []models.Contractor
	c.DB.Order("name asc").Find(&contractors)

	var users []models.User
	c.DB.Order("name asc").Limit(100).Find(&users)

	data := map[string]interface{}{
		"Locations":   locations,
		"Companies":   companies,
		"Departments": departments,
		"Contractors": contractors,
		"Users":       users,
	}

	c.Render(w, r, "/wpi_report/create.gohtml", data)
}

func (c *WpiReportController) setFlash(w http.ResponseWriter, msg, msgType string) {
	http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: msg, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: msgType, Path: "/"})
}

func (c *WpiReportController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		if wantsJSON(r) {
			writeWpiJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := r.Context().Value(middlewares.AuthUserKey).(models.User)
	if !ok {
		if wantsJSON(r) {
			writeWpiJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			return
		}
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)

	_, err := c.Service.CreateWpiReport(r, user.ID)
	if err != nil {
		if wantsJSON(r) {
			writeWpiJSONError(w, err)
			return
		}
		c.setFlash(w, "Gagal menyimpan laporan: "+err.Error(), "error")
		http.Redirect(w, r, "/wpi-report/create", http.StatusSeeOther)
		return
	}
	if wantsJSON(r) {
		writeWpiJSON(w, http.StatusCreated, map[string]string{"message": "WPI Report berhasil disimpan"})
		return
	}

	c.setFlash(w, "WPI Report berhasil disimpan!", "success")
	http.Redirect(w, r, "/wpi-report/create", http.StatusSeeOther)
}

func (c *WpiReportController) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "ID tidak valid", http.StatusBadRequest)
		return
	}

	report, err := c.Service.GetByID(id)
	if err != nil {
		c.setFlash(w, "Data tidak ditemukan", "error")
		http.Redirect(w, r, "/wpi-report", http.StatusSeeOther)
		return
	}

	// Ambil data master untuk dropdown
	var locations []models.Location
	c.DB.Order("name asc").Limit(100).Find(&locations)

	var companies []models.Company
	c.DB.Order("name asc").Find(&companies)

	var departments []models.Department
	c.DB.Order("name asc").Find(&departments)

	var contractors []models.Contractor
	c.DB.Order("name asc").Find(&contractors)

	var users []models.User
	c.DB.Order("name asc").Limit(100).Find(&users)

	data := map[string]interface{}{
		"Report":      report,
		"Locations":   locations,
		"Companies":   companies,
		"Departments": departments,
		"Contractors": contractors,
		"Users":       users,
	}

	c.Render(w, r, "/wpi_report/edit.gohtml", data)
}

func (c *WpiReportController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		if wantsJSON(r) {
			writeWpiJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		if wantsJSON(r) {
			writeWpiJSON(w, http.StatusBadRequest, map[string]string{"error": "ID tidak valid"})
			return
		}
		http.Error(w, "ID tidak valid", http.StatusBadRequest)
		return
	}

	user, ok := r.Context().Value(middlewares.AuthUserKey).(models.User)
	if !ok {
		if wantsJSON(r) {
			writeWpiJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			return
		}
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)

	_, err := c.Service.Update(id, r, user.ID)
	if err != nil {
		if wantsJSON(r) {
			writeWpiJSONError(w, err)
			return
		}
		c.setFlash(w, "Gagal mengupdate laporan: "+err.Error(), "error")
		http.Redirect(w, r, "/wpi-report/edit/"+id, http.StatusSeeOther)
		return
	}
	if wantsJSON(r) {
		writeWpiJSON(w, http.StatusOK, map[string]string{"message": "WPI Report berhasil diupdate"})
		return
	}

	c.setFlash(w, "WPI Report berhasil diupdate!", "success")
	http.Redirect(w, r, "/wpi-report/edit/"+id, http.StatusSeeOther) // Boleh diarahkan ke halaman detail/index
}

// Document serves documents from the isolated storage directory. The route itself
// is protected by withAuth and only UUID-generated filenames are accepted.
func (c *WpiReportController) Document(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ext := filepath.Ext(name)
	allowed := map[string]bool{".pdf": true, ".jpg": true, ".png": true}
	if filepath.Base(name) != name || !allowed[ext] {
		http.NotFound(w, r)
		return
	}
	
	file, err := os.Open(filepath.Join("./storage/uploads/wpi-reports", name))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, name, fileInfoModTime(file), file)
}

func fileInfoModTime(file *os.File) time.Time {
	info, err := file.Stat()
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json") || r.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

func writeWpiJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeWpiJSONError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	message := "Permintaan upload tidak valid"
	var maxBytesErr *http.MaxBytesError
	switch {
	case errors.As(err, &maxBytesErr):
		status, message = http.StatusRequestEntityTooLarge, "Ukuran total upload maksimal 5 MB"
	case errors.Is(err, services.ErrWpiUploadTooLarge):
		status, message = http.StatusRequestEntityTooLarge, "Ukuran total upload maksimal 5 MB"
	case errors.Is(err, services.ErrUnsupportedWpiFile):
		status, message = http.StatusUnsupportedMediaType, "Format file harus PDF, JPEG, atau PNG"
	case errors.Is(err, services.ErrWpiStorage):
		status, message = http.StatusInternalServerError, "Gagal menyimpan dokumen"
	}
	writeWpiJSON(w, status, map[string]string{"error": message})
}
