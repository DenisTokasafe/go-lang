package controllers

import (
	"errors"
	"fmt"
	"latihan1/cmd/web/helpers"
	"latihan1/models"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type DepartmentGroupController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (dgc *DepartmentGroupController) Index(w http.ResponseWriter, r *http.Request) {
	querySearch := r.URL.Query().Get("search")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize := 20
	offset := (page - 1) * pageSize

	var departmentGroups []models.DepartmentGroup
	var totalRows int64

	dbQuery := dgc.DB.Model(&models.DepartmentGroup{}).Joins("Department").Joins("Group")

	// search by department or group name
	if querySearch != "" {
		like := "%" + querySearch + "%"
		dbQuery = dbQuery.Where(
			"`Department`.`name` LIKE ? OR `Group`.`name` LIKE ?",
			like, like,
		)
	}

	dbQuery.Count(&totalRows)

	tx := dbQuery.
		Preload("Department").
		Preload("Group").
		Limit(pageSize).
		Offset(offset).
		Order("id DESC").
		Find(&departmentGroups)

	if tx.Error != nil {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal mengambil data department group", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		http.Redirect(w, r, "/administration/department-group", http.StatusSeeOther)
		return
	}

	var allDepartments []models.Department
	dgc.DB.Order("name ASC").Find(&allDepartments)

	var allGroups []models.Group
	dgc.DB.Order("name ASC").Find(&allGroups)

	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))

	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":             lang,
		"Tr":               helpers.Translations[lang],
		"DepartmentGroups": departmentGroups,
		"Departments":      allDepartments,
		"Groups":           allGroups,
		"CurrentPage":      page,
		"TotalPages":       totalPages,
		"Search":           querySearch,
		"HasNext":          page < totalPages,
		"HasPrev":          page > 1,
		"TotalRows":        totalRows,
		"Title":            "Department Group Management",
		"CurrentPath":      r.URL.Path,
	}

	dgc.Render(w, r, "/administration/department_group/index.html", data)
}

func (dgc *DepartmentGroupController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal membaca form", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/administration/department-group", http.StatusSeeOther)
			return
		}

		// Group tetap single
		groupIDStr := r.FormValue("group_id")
		groupID, errGroup := strconv.ParseUint(groupIDStr, 10, 32)

		// Department multiple (utama: department_ids[])
		departmentIDs := r.Form["department_ids[]"]
		if len(departmentIDs) == 0 {
			// fallback jika name tanpa []
			departmentIDs = r.Form["department_ids"]
		}

		if errGroup != nil || len(departmentIDs) == 0 {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Department atau Group tidak valid", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/administration/department-group", http.StatusSeeOther)
			return
		}

		successCount := 0
		duplicateCount := 0
		invalidCount := 0

		for _, depStr := range departmentIDs {
			depID, errDep := strconv.ParseUint(depStr, 10, 32)
			if errDep != nil {
				invalidCount++
				continue
			}

			if err := models.CreateDepartmentGroup(dgc.DB, uint(depID), uint(groupID)); err != nil {
				// biasanya kena unique constraint (duplikat)
				duplicateCount++
				continue
			}

			successCount++
		}

		if successCount == 0 {
			msg := "Tidak ada relasi baru yang ditambahkan"
			if duplicateCount > 0 || invalidCount > 0 {
				msg += " ("
				parts := []string{}
				if duplicateCount > 0 {
					parts = append(parts, strconv.Itoa(duplicateCount)+" duplikat")
				}
				if invalidCount > 0 {
					parts = append(parts, strconv.Itoa(invalidCount)+" data tidak valid")
				}
				msg += strings.Join(parts, ", ") + ")"
			}
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: msg, Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		} else {
			msg := "Berhasil menambahkan " + strconv.Itoa(successCount) + " relasi department-group"
			if duplicateCount > 0 || invalidCount > 0 {
				msg += " ("
				parts := []string{}
				if duplicateCount > 0 {
					parts = append(parts, strconv.Itoa(duplicateCount)+" duplikat dilewati")
				}
				if invalidCount > 0 {
					parts = append(parts, strconv.Itoa(invalidCount)+" data tidak valid")
				}
				msg += strings.Join(parts, ", ") + ")"
			}
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: msg, Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "success", Path: "/"})
		}
	}

	http.Redirect(w, r, "/administration/department-group", http.StatusSeeOther)
}

func (dgc *DepartmentGroupController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/administration/department-group", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal membaca form", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		http.Redirect(w, r, "/administration/department-group", http.StatusSeeOther)
		return
	}

	id := r.FormValue("id")

	// utama untuk mode edit
	departmentIDStr := r.FormValue("department_id")
	// fallback jika frontend masih kirim format create
	if departmentIDStr == "" {
		departmentIDStr = r.FormValue("department_ids[]")
	}

	groupIDStr := r.FormValue("group_id_edit")

	departmentID, errDept := strconv.ParseUint(departmentIDStr, 10, 32)
	groupID, errGroup := strconv.ParseUint(groupIDStr, 10, 32)
	fmt.Printf("%+v\n", groupIDStr)

	if id == "" || errDept != nil || errGroup != nil {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Data update tidak valid", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		http.Redirect(w, r, "/administration/department-group", http.StatusSeeOther)
		return
	}
	err := models.UpdateDepartmentGroup(dgc.DB, id, uint(departmentID), uint(groupID))
	if err != nil {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal memperbarui department group", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
	} else {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Department Group berhasil diperbarui", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "success", Path: "/"})
	}

	http.Redirect(w, r, "/administration/department-group", http.StatusSeeOther)
}

func (dgc *DepartmentGroupController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		err := models.DeleteDepartmentGroup(dgc.DB, id)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal menghapus department group", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/administration/department-group", http.StatusSeeOther)
			return
		}
	}

	http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Department Group berhasil dihapus", Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
	http.Redirect(w, r, "/administration/department-group", http.StatusSeeOther)
}

func (dgc *DepartmentGroupController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/administration/department-group", http.StatusSeeOther)
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
		// format: A=Department Name, B=Group Name
		if i == 0 || len(row) < 2 {
			continue
		}

		departmentName := strings.TrimSpace(row[0])
		groupName := strings.TrimSpace(row[1])
		if departmentName == "" || groupName == "" {
			continue
		}

		var dept models.Department
		if err := dgc.DB.Where("name = ?", departmentName).First(&dept).Error; err != nil {
			continue
		}

		var grp models.Group
		if err := dgc.DB.Where("name = ?", groupName).First(&grp).Error; err != nil {
			continue
		}

		err := dgc.DB.Where(models.DepartmentGroup{
			DepartmentID: dept.ID,
			GroupID:      grp.ID,
		}).FirstOrCreate(&models.DepartmentGroup{}).Error

		if err == nil {
			count++
		}
	}

	msg := "Berhasil mengupload " + strconv.Itoa(count) + " relasi department-group"
	http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: msg, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "success", Path: "/"})

	http.Redirect(w, r, "/administration/department-group", http.StatusSeeOther)
}

func (mc *DepartmentGroupController) ExportExcel(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil data dari database dengan Preload relasi
	var data []models.DepartmentGroup
	if err := mc.DB.Preload("Department").Preload("Group").Find(&data).Error; err != nil {
		http.Error(w, "Gagal mengambil data", http.StatusInternalServerError)
		return
	}

	// 2. Buat file excel baru
	f := excelize.NewFile()
	sheetName := "Department Groups"
	index, _ := f.NewSheet(sheetName)
	f.DeleteSheet("Sheet1") // Hapus sheet default

	// 3. Buat Header
	headers := []string{"ID", "Department Name", "Group Name", "Created At"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Styling Header (Opsional agar rapi)
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	f.SetCellStyle(sheetName, "A1", "D1", style)

	// 4. Isi Data ke baris Excel
	for i, item := range data {
		rowNum := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), item.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), item.Department.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), item.Group.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), item.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	f.SetActiveSheet(index)

	// 5. Set header respons HTTP agar browser mendownload file
	filename := fmt.Sprintf("Department_Groups_%s.xlsx", time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	if err := f.Write(w); err != nil {
		http.Error(w, "Gagal mengirim file", http.StatusInternalServerError)
	}
}
