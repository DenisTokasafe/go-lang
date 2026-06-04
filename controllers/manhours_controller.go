package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"latihan1/models"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ManhoursController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (mc *ManhoursController) Index(w http.ResponseWriter, r *http.Request) {
	querySearch := r.URL.Query().Get("search")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize := 20
	offset := (page - 1) * pageSize

	var manhours []models.Manhours
	var totalRows int64

	// 1. Inisialisasi Query Dasar untuk Tabel
	dbQuery := mc.DB.Model(&models.Manhours{})

	// 2. Terapkan Filter Search
	if querySearch != "" {
		like := "%" + querySearch + "%"
		dbQuery = dbQuery.
			Joins("LEFT JOIN business_units ON business_units.id = manhours.business_unit_id").
			Joins("LEFT JOIN contractors ON contractors.id = manhours.contractor_id").
			Joins("LEFT JOIN custodians ON custodians.id = manhours.custodian_id").
			Joins("LEFT JOIN departments ON departments.id = custodians.department_id").
			Where(`(business_units.name LIKE ? OR 
                    contractors.name LIKE ? OR 
                    manhours.entity_type LIKE ? OR 
                    departments.name LIKE ?)`, like, like, like, like)
	}

	// 3. Terapkan Filter Date Range
	if startDate != "" && endDate != "" {
		dbQuery = dbQuery.Where("manhours.month BETWEEN ? AND ?", startDate, endDate)
	}

	// Hitung total baris untuk pagination tabel
	dbQuery.Count(&totalRows)

	// --- LOGIKA AGREGASI HIGHCHARTS ---
	type ChartData struct {
		Month string  `json:"month"` // Ini akan menghasilkan format seperti "Jan 2025" (berdasarkan DATE_FORMAT)
		Total float64 `json:"total"`
		Count int64   `json:"count"`
	}
	var chartResults []ChartData

	// Inisialisasi query untuk chart
	chartQuery := mc.DB.Model(&models.Manhours{}).
		Select(`
            DATE_FORMAT(manhours.month, '%b %Y') as month, 
            SUM(operational_hours + supervisor_hours + admin_hours) as total,
            SUM(operational_count + supervisor_count + admin_count) as count,
            DATE(manhours.month) as raw_month
        `) // UBAH: Menggunakan DATE(manhours.month) agar menghasilkan format '2025-01-01'

	// KONDISI KHUSUS: Jika tanpa search dan tanpa date range
	if querySearch == "" && startDate == "" && endDate == "" {
		// Tampilkan hanya 12 bulan terakhir
		chartQuery = chartQuery.
			// UBAH: Menggunakan CURDATE() untuk menggantikan NOW() agar berbasis tanggal murni
			Where("manhours.month > DATE_SUB(CURDATE(), INTERVAL 13 MONTH)")
	} else {
		// Jika ada filter, grafik mengikuti filter tersebut
		chartQuery = chartQuery.Joins("LEFT JOIN business_units ON business_units.id = manhours.business_unit_id").
			Joins("LEFT JOIN contractors ON contractors.id = manhours.contractor_id").
			Joins("LEFT JOIN custodians ON custodians.id = manhours.custodian_id").
			Joins("LEFT JOIN departments ON departments.id = custodians.department_id").
			Where(dbQuery)
	}

	chartQuery.
		Group("DATE_FORMAT(manhours.month, '%b %Y'), DATE(manhours.month)"). // UBAH: Grouping disesuaikan dengan fungsi select asli
		Order("raw_month ASC").
		Scan(&chartResults)

	chartJSON, _ := json.Marshal(chartResults)
	// ----------------------------------

	// 4. Ambil Data untuk Tabel (Pagination)
	tx := dbQuery.
		Preload("BusinessUnit.Company").
		Preload("Contractor").
		Preload("DepartmentGroup.Department").
		Preload("DepartmentGroup.Group").
		Preload("Custodian.Department").
		Preload("Custodian.Contractor").
		Preload("Custodian.Department.DepartmentGroups.Group").
		Limit(pageSize).
		Offset(offset).
		Order("month DESC").
		Find(&manhours)

	if tx.Error != nil {
		http.Error(w, "Gagal mengambil data", http.StatusInternalServerError)
		return
	}

	// Ambil data pendukung (Dropdown, dll)
	var allBusinessUnits []models.BusinessUnit
	mc.DB.Preload("Company").Order("name ASC").Find(&allBusinessUnits)

	var allDepartmentGroups []models.DepartmentGroup
	mc.DB.Preload("Department").Preload("Group").Order("id DESC").Find(&allDepartmentGroups)

	var allContractors []models.Contractor
	mc.DB.Order("name ASC").Find(&allContractors)

	var allCustodians []models.Custodian
	mc.DB.Preload("Department").Preload("Contractor").Order("id DESC").Find(&allCustodians)

	custodiansJSON, _ := json.Marshal(allCustodians)
	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))

	data := map[string]interface{}{
		"Manhours":         manhours,
		"BusinessUnits":    allBusinessUnits,
		"DepartmentGroups": allDepartmentGroups,
		"Contractors":      allContractors,
		"Custodians":       allCustodians,
		"CustodiansJSON":   string(custodiansJSON),
		"ChartJSON":        string(chartJSON),
		"CurrentPage":      page,
		"TotalPages":       totalPages,
		"Search":           querySearch,
		"StartDate":        startDate,
		"EndDate":          endDate,
		"HasNext":          page < totalPages,
		"HasPrev":          page > 1,
		"TotalRows":        totalRows,
		"Title":            "Manhours Management",
		"CurrentPath":      r.URL.Path,
		"EntityTypes":      []string{"mine_company", "contractor"},
	}

	mc.Render(w, r, "/manhours/index.gohtml", data)
}

func (mc *ManhoursController) parseMonth(value string) (time.Time, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return time.Time{}, errors.New("bulan tidak valid")
	}

	if t, err := time.Parse("2006-01", value); err == nil {
		return t, nil
	}

	return time.Parse("2006-01-02", value)
}

func (mc *ManhoursController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
		return
	}

	monthStr := r.FormValue("month")
	entityType := r.FormValue("entity_type")
	businessUnitIDStr := r.FormValue("business_unit_id")
	contractorIDStr := r.FormValue("contractor_id")
	departmentGroupIDStr := r.FormValue("department_group_id")
	custodianIDStr := r.FormValue("custodian_id")

	month, err := mc.parseMonth(monthStr)
	if err != nil {
		mc.setFlash(w, "Bulan tidak valid", "error")
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
		return
	}

	var businessUnitID *uint
	var contractorID *uint
	var departmentGroupID *uint
	var custodianID *uint

	switch entityType {
	case "mine_company":
		if businessUnitIDStr == "" {
			mc.setFlash(w, "Business Unit harus dipilih untuk Mine Company", "error")
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		buID, _ := strconv.ParseUint(businessUnitIDStr, 10, 32)
		bu := uint(buID)
		businessUnitID = &bu

		if departmentGroupIDStr == "" {
			mc.setFlash(w, "Department Group harus dipilih untuk Mine Company", "error")
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		dgID, _ := strconv.ParseUint(departmentGroupIDStr, 10, 32)
		dg := uint(dgID)
		departmentGroupID = &dg

	case "contractor":
		if contractorIDStr == "" {
			mc.setFlash(w, "Contractor harus dipilih untuk Contractor", "error")
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		cID, _ := strconv.ParseUint(contractorIDStr, 10, 32)
		c := uint(cID)
		contractorID = &c

		if custodianIDStr == "" {
			mc.setFlash(w, "Custodian harus dipilih untuk Contractor", "error")
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		cusID, _ := strconv.ParseUint(custodianIDStr, 10, 32)
		cus := uint(cusID)
		custodianID = &cus

	default:
		mc.setFlash(w, "Tipe entitas tidak valid", "error")
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
		return
	}

	// Validasi Supervisor
	supervisorHours := 0.0
	supervisorCount := 0
	if r.FormValue("has_supervisor") == "1" {
		shStr := r.FormValue("supervisor_hours")
		scStr := r.FormValue("supervisor_count")
		if shStr == "" || scStr == "" {
			mc.setFlash(w, "Supervisor harus diisi jika dicentang", "error")
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		supervisorHours, _ = strconv.ParseFloat(shStr, 64)
		sc, _ := strconv.ParseUint(scStr, 10, 32)
		supervisorCount = int(sc)

		if supervisorHours <= 0 || supervisorCount <= 0 {
			mc.setFlash(w, "Data Supervisor harus angka positif", "error")
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
	}

	// Validasi Administrator
	adminHours := 0.0
	adminCount := 0
	if r.FormValue("has_admin") == "1" {
		ahStr := r.FormValue("admin_hours")
		acStr := r.FormValue("admin_count")
		if ahStr == "" || acStr == "" {
			mc.setFlash(w, "Administrator harus diisi jika dicentang", "error")
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		adminHours, _ = strconv.ParseFloat(ahStr, 64)
		ac, _ := strconv.ParseUint(acStr, 10, 32)
		adminCount = int(ac)

		if adminHours <= 0 || adminCount <= 0 {
			mc.setFlash(w, "Data Administrator harus angka positif", "error")
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
	}

	// Validasi Operational (Wajib)
	ohStr := r.FormValue("operational_hours")
	ocStr := r.FormValue("operational_count")
	if ohStr == "" || ocStr == "" {
		mc.setFlash(w, "Operational harus diisi", "error")
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
		return
	}
	operationalHours, _ := strconv.ParseFloat(ohStr, 64)
	oc, _ := strconv.ParseUint(ocStr, 10, 32)
	operationalCount := uint(oc)

	if operationalHours <= 0 || operationalCount <= 0 {
		mc.setFlash(w, "Operational harus berupa angka positif", "error")
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
		return
	}

	// Simpan ke Database
	err = models.CreateManhours(
		mc.DB, month, entityType, businessUnitID, contractorID,
		departmentGroupID, custodianID, supervisorHours, uint(supervisorCount),
		operationalHours, operationalCount, adminHours, uint(adminCount),
		r.FormValue("notes"),
	)

	if err != nil {
		mc.setFlash(w, "Gagal menyimpan data manhours", "error")
	} else {
		mc.setFlash(w, "Manhours berhasil ditambahkan", "success")
	}

	http.Redirect(w, r, "/manhours", http.StatusSeeOther)
}

func (mc *ManhoursController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
		return
	}

	id := r.FormValue("id")
	if id == "" {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "ID tidak ditemukan", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
		return
	}

	monthStr := r.FormValue("month")
	entityType := r.FormValue("entity_type")
	businessUnitIDStr := r.FormValue("business_unit_id")
	contractorIDStr := r.FormValue("contractor_id")
	departmentGroupIDStr := r.FormValue("department_group_id")
	custodianIDStr := r.FormValue("custodian_id")

	month, err := mc.parseMonth(monthStr)
	if err != nil {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Bulan tidak valid", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
		return
	}

	var businessUnitID *uint
	var contractorID *uint
	var departmentGroupID *uint
	var custodianID *uint

	switch entityType {
	case "mine_company":
		if businessUnitIDStr == "" {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Business Unit harus dipilih untuk Mine Company", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		buID, err := strconv.ParseUint(businessUnitIDStr, 10, 32)
		if err != nil {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Business Unit tidak valid", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		bu := uint(buID)
		businessUnitID = &bu

		if departmentGroupIDStr == "" {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Department Group harus dipilih untuk Mine Company", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		dgID, err := strconv.ParseUint(departmentGroupIDStr, 10, 32)
		if err != nil {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Department Group tidak valid", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		dg := uint(dgID)
		departmentGroupID = &dg
	case "contractor":
		if contractorIDStr == "" {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Contractor harus dipilih untuk Contractor", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		cID, err := strconv.ParseUint(contractorIDStr, 10, 32)
		if err != nil {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Contractor tidak valid", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		c := uint(cID)
		contractorID = &c

		if custodianIDStr == "" {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Custodian harus dipilih untuk Contractor", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		cID2, err := strconv.ParseUint(custodianIDStr, 10, 32)
		if err != nil {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Custodian tidak valid", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		c2 := uint(cID2)
		custodianID = &c2
	default:
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Tipe entitas tidak valid", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
		return
	}

	hasSupervisor := r.FormValue("has_supervisor") == "1"
	hasAdmin := r.FormValue("has_admin") == "1"

	if hasSupervisor {
		shStr := r.FormValue("supervisor_hours")
		scStr := r.FormValue("supervisor_count")
		if shStr == "" || scStr == "" {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Supervisor harus diisi jika checkbox dicentang", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		sh, err := strconv.ParseFloat(shStr, 64)
		if err != nil || sh <= 0 {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Supervisor hours harus berupa angka positif", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		sc, err := strconv.ParseUint(scStr, 10, 32)
		if err != nil || sc == 0 {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Supervisor count harus berupa angka positif", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
	}

	if hasAdmin {
		ahStr := r.FormValue("admin_hours")
		acStr := r.FormValue("admin_count")
		if ahStr == "" || acStr == "" {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Administrator harus diisi jika checkbox dicentang", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		ah, err := strconv.ParseFloat(ahStr, 64)
		if err != nil || ah <= 0 {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Administrator hours harus berupa angka positif", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
		ac, err := strconv.ParseUint(acStr, 10, 32)
		if err != nil || ac == 0 {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Administrator count harus berupa angka positif", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
	}

	// Validate operational
	ohStr := r.FormValue("operational_hours")
	ocStr := r.FormValue("operational_count")
	if ohStr == "" || ocStr == "" {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Operational harus diisi", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
		return
	}
	oh, err := strconv.ParseFloat(ohStr, 64)
	if err != nil || oh <= 0 {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Operational hours harus berupa angka positif", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
		return
	}
	oc, err := strconv.ParseUint(ocStr, 10, 32)
	if err != nil || oc == 0 {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Operational count harus berupa angka positif", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
		return
	}

	supervisorHours, _ := strconv.ParseFloat(r.FormValue("supervisor_hours"), 64)
	supervisorCount, _ := strconv.ParseUint(r.FormValue("supervisor_count"), 10, 32)
	operationalHours := oh
	operationalCount := oc
	adminHours, _ := strconv.ParseFloat(r.FormValue("admin_hours"), 64)
	adminCount, _ := strconv.ParseUint(r.FormValue("admin_count"), 10, 32)

	if err := mc.DB.Model(&models.Manhours{}).Where("id = ?", id).Updates(map[string]interface{}{
		"month":               month,
		"business_unit_id":    businessUnitID,
		"contractor_id":       contractorID,
		"entity_type":         entityType,
		"department_group_id": departmentGroupID,
		"custodian_id":        custodianID,
		"supervisor_hours":    supervisorHours,
		"supervisor_count":    uint(supervisorCount),
		"operational_hours":   operationalHours,
		"operational_count":   uint(operationalCount),
		"admin_hours":         adminHours,
		"admin_count":         uint(adminCount),
		"notes":               r.FormValue("notes"),
	}).Error; err != nil {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal memperbarui data manhours", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
	} else {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Manhours berhasil diperbarui", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "success", Path: "/"})
		// Kirim email notifikasi (Bungkus dengan goroutine 'go' agar tidak membuat loading web lambat)

	}

	http.Redirect(w, r, "/manhours", http.StatusSeeOther)
}

func (mc *ManhoursController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		if err := mc.DB.Unscoped().Delete(&models.Manhours{}, id).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal menghapus data manhours", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/manhours", http.StatusSeeOther)
			return
		}
	}

	http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Manhours berhasil dihapus", Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})

	http.Redirect(w, r, "/manhours", http.StatusSeeOther)
}
func (mc *ManhoursController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/manhours", http.StatusSeeOther)
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

	// 3. Baca data
	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		http.Error(w, "Gagal membaca data", http.StatusInternalServerError)
		return
	}

	tx := mc.DB.Begin()

	for i, row := range rows {
		// Lewati header dan pastikan kolom cukup
		if i == 0 || len(row) < 8 {
			continue
		}

		// --- 1. Ambil Data Dasar dari Excel ---
		dateStr := strings.TrimSpace(row[0])
		companyName := strings.TrimSpace(row[2])
		deptNameExcel := strings.TrimSpace(row[3])
		groupNameExcel := strings.TrimSpace(row[4])
		jobClass := strings.ToLower(strings.TrimSpace(row[5]))
		manhoursVal, _ := strconv.ParseFloat(strings.TrimSpace(row[6]), 64)
		manpowerVal, _ := strconv.ParseInt(strings.TrimSpace(row[7]), 10, 64)

		// --- 2. Format Tanggal ---
		var tempDate time.Time
		tempDate, err = time.Parse("02/01/2006", dateStr)
		if err != nil {
			tempDate, _ = time.Parse("2006-01-02", dateStr)
		}
		parsedDate := time.Date(tempDate.Year(), tempDate.Month(), 1, 8, 0, 0, 0, tempDate.Location())

		var mhr models.Manhours
		var entityType string
		var buID, contractorID, custodianID, deptGroupID *uint

		// --- 3. Implementasi Alur Logika Baru dengan LOWER() ---

		// A. CEK SEBAGAI BUSINESS UNIT (Internal)
		var bu models.BusinessUnit
		// Menggunakan LOWER() untuk case-insensitive matching
		if errBU := tx.Where("LOWER(name) = LOWER(?)", companyName).First(&bu).Error; errBU == nil {
			entityType = "mine_company"
			buID = &bu.ID

			// Alur BU: Cari ID Departemen dengan LOWER()
			var dept models.Department
			if errDept := tx.Where("LOWER(name) = LOWER(?)", deptNameExcel).First(&dept).Error; errDept == nil {
				var dg models.DepartmentGroup
				errDG := tx.Joins("JOIN groups ON groups.id = department_groups.group_id").
					Where("department_groups.department_id = ? AND LOWER(groups.name) = LOWER(?)", dept.ID, groupNameExcel).
					First(&dg).Error
				if errDG == nil {
					deptGroupID = &dg.ID
				}
			}
		} else {
			// B. CEK SEBAGAI CONTRACTOR
			var con models.Contractor
			if errCon := tx.Where("LOWER(name) = LOWER(?)", companyName).First(&con).Error; errCon == nil {
				entityType = "contractor"
				contractorID = &con.ID

				// Alur Contractor: Cari relasi di tabel Custodians dengan JOIN + LOWER()
				var cust models.Custodian
				errCust := tx.Joins("JOIN departments ON departments.id = custodians.department_id").
					Where("custodians.contractor_id = ? AND LOWER(departments.name) = LOWER(?)", con.ID, deptNameExcel).
					First(&cust).Error

				if errCust == nil {
					custodianID = &cust.ID
					// Ambil DepartmentGroupID
					var dg models.DepartmentGroup
					errDG := tx.Joins("JOIN groups ON groups.id = department_groups.group_id").
						Where("department_groups.department_id = ? AND LOWER(groups.name) = LOWER(?)", cust.DepartmentID, groupNameExcel).
						First(&dg).Error
					if errDG == nil {
						deptGroupID = &dg.ID
					}
				}
			} else {
				entityType = "contractor"
				fmt.Printf("--> Warning: %s tidak terdaftar di sistem\n", companyName)
			}
		}

		// --- 4. Upsert Data ---
		query := tx.Model(&models.Manhours{}).Where("month = ? AND entity_type = ?", parsedDate, entityType)
		if entityType == "mine_company" {
			query = query.Where("business_unit_id = ?", buID)
		} else {
			query = query.Where("contractor_id = ?", contractorID)
		}

		if deptGroupID != nil {
			query = query.Where("department_group_id = ?", deptGroupID)
		}

		query.First(&mhr)

		// --- 5. Mapping Nilai ---
		switch jobClass {
		case "supervisor":
			mhr.SupervisorHours = manhoursVal
			mhr.SupervisorCount = uint(manpowerVal)
		case "operational":
			mhr.OperationalHours = manhoursVal
			mhr.OperationalCount = uint(manpowerVal)
		case "administrator", "admin":
			mhr.AdminHours = manhoursVal
			mhr.AdminCount = uint(manpowerVal)
		}

		// --- 6. Finalisasi & Simpan ---
		mhr.Month = parsedDate
		mhr.EntityType = entityType
		mhr.BusinessUnitID = buID
		mhr.ContractorID = contractorID
		mhr.CustodianID = custodianID
		mhr.DepartmentGroupID = deptGroupID

		if err := tx.Save(&mhr).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Gagal menyimpan ke database", http.StatusInternalServerError)
			return
		}
	}

	tx.Commit()

	http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Import Manhours Berhasil", Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "success", Path: "/"})
	http.Redirect(w, r, "/manhours", http.StatusSeeOther)
}
func (mc *ManhoursController) setFlash(w http.ResponseWriter, msg string, msgType string) {
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
