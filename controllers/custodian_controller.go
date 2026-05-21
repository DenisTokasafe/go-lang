package controllers

import (
	"errors"
	"latihan1/models"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type CustodianController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (cc *CustodianController) Index(w http.ResponseWriter, r *http.Request) {
	querySearch := r.URL.Query().Get("search")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize := 20
	offset := (page - 1) * pageSize

	var custodians []models.Custodian
	var totalRows int64

	dbQuery := cc.DB.Model(&models.Custodian{}).Joins("Department").Joins("Contractor")

	// search by department or contractor name
	if querySearch != "" {
		like := "%" + querySearch + "%"
		dbQuery = dbQuery.Where(
			"`Department`.`name` LIKE ? OR `Contractor`.`name` LIKE ?",
			like, like,
		)
	}

	dbQuery.Count(&totalRows)

	tx := dbQuery.
		Preload("Department").
		Preload("Contractor").
		Limit(pageSize).
		Offset(offset).
		Order("id DESC").
		Find(&custodians)

	if tx.Error != nil {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal mengambil data kustodian", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		http.Redirect(w, r, "/administration/custodian", http.StatusSeeOther)
		return
	}

	var allDepartments []models.Department
	cc.DB.Order("name ASC").Find(&allDepartments)

	var allContractors []models.Contractor
	cc.DB.Order("name ASC").Find(&allContractors)

	totalPages := int(math.Ceil(float64(totalRows) / float64(pageSize)))

	data := map[string]interface{}{
		"Custodians":  custodians,
		"Departments": allDepartments,
		"Contractors": allContractors,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Search":      querySearch,
		"HasNext":     page < totalPages,
		"HasPrev":     page > 1,
		"TotalRows":   totalRows,
		"Title":       "Custodian Management",
		"CurrentPath": r.URL.Path,
	}

	cc.Render(w, r, "/administration/custodian/index.html", data)
}

func (cc *CustodianController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal membaca form", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/administration/custodian", http.StatusSeeOther)
			return
		}

		// Department tetap single (satu department sebagai parent)
		departmentIDStr := r.FormValue("department_id")
		departmentID, errDept := strconv.ParseUint(departmentIDStr, 10, 32)

		// Contractor multiple (mengambil array dari form)
		contractorIDs := r.Form["contractor_ids[]"]
		if len(contractorIDs) == 0 {
			// fallback jika name di html tidak menggunakan []
			contractorIDs = r.Form["contractor_ids"]
		}

		// Validasi dasar
		if errDept != nil || len(contractorIDs) == 0 {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Department atau Contractor tidak valid", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/administration/custodian", http.StatusSeeOther)
			return
		}

		successCount := 0
		duplicateCount := 0
		invalidCount := 0

		// Iterasi semua contractor yang dipilih
		for _, ctrStr := range contractorIDs {
			ctrID, errCtr := strconv.ParseUint(ctrStr, 10, 32)
			if errCtr != nil {
				invalidCount++
				continue
			}

			// Simpan relasi ke database
			err := models.CreateCustodian(cc.DB, uint(departmentID), uint(ctrID))
			if err != nil {
				// Biasanya error karena unique constraint (pasangan Dept + Ctr sudah ada)
				duplicateCount++
				continue
			}

			successCount++
		}

		// Logika Pesan Flash
		if successCount == 0 {
			msg := "Tidak ada kustodian baru yang ditambahkan"
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
			msg := "Berhasil menambahkan " + strconv.Itoa(successCount) + " kustodian"
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

	http.Redirect(w, r, "/administration/custodian", http.StatusSeeOther)
}

func (cc *CustodianController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/administration/custodian", http.StatusSeeOther)
		return
	}

	// Wajib parse form agar data POST dapat dibaca
	if err := r.ParseForm(); err != nil {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal membaca form", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		http.Redirect(w, r, "/administration/custodian", http.StatusSeeOther)
		return
	}

	id := r.FormValue("id")

	// Ambil Department ID (Biasanya single di mode edit)
	departmentIDStr := r.FormValue("department_id")

	// Ambil Contractor ID (Gunakan suffix _edit jika frontend membedakan field modal create dan edit)
	contractorIDStr := r.FormValue("contractor_id_edit")

	// Fallback jika frontend menggunakan nama field yang sama dengan mode create
	if contractorIDStr == "" {
		contractorIDStr = r.FormValue("contractor_id")
	}

	departmentID, errDept := strconv.ParseUint(departmentIDStr, 10, 32)
	contractorID, errCtr := strconv.ParseUint(contractorIDStr, 10, 32)

	// Validasi kelengkapan data
	if id == "" || errDept != nil || errCtr != nil {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Data update tidak valid", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
		http.Redirect(w, r, "/administration/custodian", http.StatusSeeOther)
		return
	}

	// Eksekusi update ke database melalui model
	err := models.UpdateCustodian(cc.DB, id, uint(departmentID), uint(contractorID))
	if err != nil {
		// Logika ini menangani jika update gagal, misal karena melanggar unique constraint
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal memperbarui kustodian (Data mungkin sudah ada)", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
	} else {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Kustodian berhasil diperbarui", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "success", Path: "/"})
	}

	http.Redirect(w, r, "/administration/custodian", http.StatusSeeOther)
}

func (cc *CustodianController) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		err := models.DeleteCustodian(cc.DB, id)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Gagal menghapus kustodian", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
			http.Redirect(w, r, "/administration/custodian", http.StatusSeeOther)
			return
		}
	}

	http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "Kustodian berhasil dihapus", Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "error", Path: "/"})
	http.Redirect(w, r, "/administration/custodian", http.StatusSeeOther)
}

func (cc *CustodianController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/administration/custodian", http.StatusSeeOther)
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
		// format: A=Department Name, B=Contractor Name
		if i == 0 || len(row) < 2 {
			continue
		}

		departmentName := strings.TrimSpace(row[0])
		contractorName := strings.TrimSpace(row[1])
		if departmentName == "" || contractorName == "" {
			continue
		}

		var dept models.Department
		if err := cc.DB.Where("name = ?", departmentName).First(&dept).Error; err != nil {
			continue
		}

		var ctr models.Contractor
		if err := cc.DB.Where("name = ?", contractorName).First(&ctr).Error; err != nil {
			continue
		}

		err := cc.DB.Where(models.Custodian{
			DepartmentID: dept.ID,
			ContractorID: ctr.ID,
		}).FirstOrCreate(&models.Custodian{}).Error

		if err == nil {
			count++
		}
	}

	msg := "Berhasil mengupload " + strconv.Itoa(count) + " relasi kustodian"
	http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: msg, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "flash_type", Value: "success", Path: "/"})

	http.Redirect(w, r, "/administration/custodian", http.StatusSeeOther)
}
