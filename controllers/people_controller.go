package controllers

import (
	// Sesuaikan dengan nama module di go.mod Anda

	"fmt"
	"latihan1/cmd/web/helpers"
	"latihan1/models"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

func (uc *UserController) Index(w http.ResponseWriter, r *http.Request) {
	querySearch := r.URL.Query().Get("search")

	// Filter berdasarkan Role
	filterRole := r.URL.Query().Get("role_id")

	// =========================
	// PAGINATION
	// =========================
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize := 20
	offset := (page - 1) * pageSize

	var users []models.User
	var totalRows int64

	// =========================
	// BASE QUERY
	// =========================
	dbQuery := uc.DB.Model(&models.User{})

	// =========================
	// SEARCH FILTER
	// =========================
	if querySearch != "" {

		like := "%" + querySearch + "%"

		dbQuery = dbQuery.
			Joins("LEFT JOIN roles ON roles.id = users.role_id").
			Joins("LEFT JOIN contractors ON contractors.id = users.contractor_id").
			Joins("LEFT JOIN departments ON departments.id = users.department_id").
			Where(`(
				users.name LIKE ? OR
				users.username LIKE ? OR 
				users.email LIKE ? OR 
				roles.name LIKE ? OR 
				contractors.name LIKE ? OR 
				departments.name LIKE ?
			)`,
				like,
				like,
				like,
				like,
				like,
				like,
			)
	}

	// =========================
	// FILTER ROLE
	// =========================
	if filterRole != "" {
		dbQuery = dbQuery.Where("users.role_id = ?", filterRole)
	}

	// =========================
	// TOTAL DATA
	// =========================
	dbQuery.Count(&totalRows)

	// =========================
	// GET USERS
	// =========================
	tx := dbQuery.
		Preload("Role").
		Preload("Contractor").
		Preload("Department").
		Preload("ModeratedCategories").
		Limit(pageSize).
		Offset(offset).
		Order("users.created_at DESC").
		Find(&users)

	if tx.Error != nil {

		http.Error(
			w,
			"Gagal mengambil data user",
			http.StatusInternalServerError,
		)

		return
	}

	// =========================
	// GET ROLES
	// =========================
	var allRoles []models.Role

	uc.DB.
		Order("name ASC").
		Find(&allRoles)

	// =========================
	// GET CONTRACTORS
	// =========================
	var allContractors []models.Contractor

	uc.DB.
		Order("name ASC").
		Find(&allContractors)

	// =========================
	// GET DEPARTMENTS
	// =========================
	var allDepartments []models.Department

	uc.DB.
		Order("name ASC").
		Find(&allDepartments)

	// =========================
	// GET EVENT CATEGORIES
	// HANYA PARENT CATEGORY
	// =========================
	var eventCategories []models.EventCategory

	uc.DB.
		Where("parent_id IS NULL").
		Order("name ASC").
		Find(&eventCategories)

	// =========================
	// TOTAL PAGES
	// =========================
	totalPages := int(
		math.Ceil(
			float64(totalRows) / float64(pageSize),
		),
	)

	// =========================
	// SEND DATA TO TEMPLATE
	// =========================
	lang := "id"
	if cookie, err := r.Cookie("lang"); err == nil {
		lang = cookie.Value
	}

	data := map[string]interface{}{
		"Lang":            lang,
		"Tr":              helpers.Translations[lang],
		"Users":           users,
		"Roles":           allRoles,
		"Contractors":     allContractors,
		"Departments":     allDepartments,
		"EventCategories": eventCategories,
		"TotalRows":       totalRows,
		"CurrentPage":     page,
		"TotalPages":      totalPages,
		"Search":          querySearch,
		"FilterRole":      filterRole,
		"HasNext":         page < totalPages,
		"HasPrev":         page > 1,
		"Title":           "User Management",
		"CurrentPath":     r.URL.Path,
	}

	// =========================
	// RENDER
	// =========================
	uc.Render(
		w,
		r,
		"people/index.gohtml",
		data,
	)
}
func (uc *UserController) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return
	}

	name := r.FormValue("name")
	username := r.FormValue("username")
	email := r.FormValue("email")
	gender := r.FormValue("gender")
	employeeID := r.FormValue("employee_id")
	dateOfBirth := r.FormValue("date_of_birth")
	password := r.FormValue("password")
	roleID, _ := strconv.Atoi(r.FormValue("role_id"))

	// Konversi nilai checkbox is_pic
	isPIC := r.FormValue("is_pic") == "true" || r.FormValue("is_pic") == "on"

	// Logika Hubungan Kerja
	workType := r.FormValue("work_type")
	contractorIDStr := r.FormValue("contractor_id")
	departmentIDStr := r.FormValue("department_id")
	moderatorCategoryIDs := r.Form["moderator_category_ids[]"]

	// 1. Validasi Input Wajib (Early Return agar tidak bersarang/nested)
	if name == "" || username == "" || email == "" || password == "" {
		uc.setFlash(w, "Nama, Username, Email, dan Password wajib diisi", "error")
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return
	}

	// 2. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		uc.setFlash(w, "Gagal mengenkripsi password", "error")
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return
	}

	// 3. Handle Tanggal Lahir (Mendukung NULL jika kosong)
	var dob *time.Time // <--- GANTI INI (secara default bernilai nil/NULL)

	if dateOfBirth != "" {
		parsedDate, err := time.Parse("2006-01-02", dateOfBirth)
		if err != nil {
			uc.setFlash(w, "Format tanggal lahir tidak valid", "error")
			http.Redirect(w, r, "/people", http.StatusSeeOther)
			return
		}
		dob = &parsedDate // Ambil pointer dari parsedDate
	}

	// 4. Ambil Kategori Moderator (OPTIMASI: Menghindari N+1 Query di dalam Loop)
	var categories []models.EventCategory
	if len(moderatorCategoryIDs) > 0 {
		var catIDs []uint
		for _, idStr := range moderatorCategoryIDs {
			if idUint, err := strconv.ParseUint(idStr, 10, 64); err == nil {
				catIDs = append(catIDs, uint(idUint))
			}
		}

		if len(catIDs) > 0 {
			// Cukup jalankan 1 query dengan operator IN untuk mengambil semua kategori
			uc.DB.Where("id IN ? AND parent_id IS NULL", catIDs).Find(&categories)
		}
	}

	// 5. Inisialisasi Model User
	newUser := models.User{
		Name:                name,
		Username:            username,
		IsPIC:               isPIC,
		Email:               email,
		Gender:              gender,
		EmployeeID:          employeeID,
		DateOfBirth:         dob,
		Password:            string(hashedPassword),
		RoleID:              uint(roleID),
		ModeratedCategories: categories, // Langsung pasang ke struct agar GORM menyimpannya secara otomatis
	}

	// 6. Set Unit Kerja Berdasarkan Pilihan (Pointer Logic)
	if workType == "contractor" && contractorIDStr != "" {
		if id, err := strconv.Atoi(contractorIDStr); err == nil {
			cID := uint(id)
			newUser.ContractorID = &cID
			newUser.DepartmentID = nil
		}
	} else if workType == "department" && departmentIDStr != "" {
		if id, err := strconv.Atoi(departmentIDStr); err == nil {
			dID := uint(id)
			newUser.DepartmentID = &dID
			newUser.ContractorID = nil
		}
	}

	// 7. Simpan ke Database (User beserta relasi Many-to-Many akan tersimpan sekaligus)
	if err := uc.DB.Create(&newUser).Error; err != nil {
		uc.setFlash(w, "Gagal menambahkan user: "+err.Error(), "error")
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return
	}

	uc.setFlash(w, "User berhasil ditambahkan", "success")
	http.Redirect(w, r, "/people", http.StatusSeeOther)
}
func (uc *UserController) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return
	}

	id := r.FormValue("id")
	name := r.FormValue("name")
	username := r.FormValue("username")
	email := r.FormValue("email")
	gender := r.FormValue("gender")
	employeeID := r.FormValue("employee_id")
	dateOfBirth := r.FormValue("date_of_birth")
	roleID, _ := strconv.Atoi(r.FormValue("role_id"))
	isPIC := r.FormValue("is_pic") == "true" || r.FormValue("is_pic") == "on"

	// Logika Hubungan Kerja
	workType := r.FormValue("work_type")
	contractorIDStr := r.FormValue("contractor_id")
	departmentIDStr := r.FormValue("department_id")
	moderatorCategoryIDs := r.Form["moderator_category_ids[]"]

	// 1. Validasi Input Wajib (Early Return)
	if id == "" || username == "" || email == "" {
		uc.setFlash(w, "ID, Username, dan Email wajib diisi", "error")
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return
	}

	// 2. Ambil Data User Lama dari Database
	var user models.User
	if err := uc.DB.First(&user, id).Error; err != nil {
		uc.setFlash(w, "User tidak ditemukan", "error")
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return
	}

	// 3. Handle Tanggal Lahir (Disamakan menggunakan sql.NullTime seperti fungsi Store)
	var dob *time.Time // <--- GANTI INI (secara default bernilai nil/NULL)

	if dateOfBirth != "" {
		parsedDate, err := time.Parse("2006-01-02", dateOfBirth)
		if err != nil {
			uc.setFlash(w, "Format tanggal lahir tidak valid", "error")
			http.Redirect(w, r, "/people", http.StatusSeeOther)
			return
		}
		dob = &parsedDate // Ambil pointer dari parsedDate
	}

	// 4. Update Data Dasar pada Struct
	user.Name = name
	user.Username = username
	user.Email = email
	user.Gender = gender
	user.EmployeeID = employeeID
	user.DateOfBirth = dob // FIXED: Menggunakan sql.NullTime yang aman
	user.RoleID = uint(roleID)
	user.IsPIC = isPIC

	// 5. Update Password (Hanya jika diisi dari form)
	password := r.FormValue("password")
	if password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			uc.setFlash(w, "Gagal mengenkripsi password baru", "error")
			http.Redirect(w, r, "/people", http.StatusSeeOther)
			return
		}
		user.Password = string(hashedPassword)
	}

	// 6. Logika Reset & Set Unit Kerja (Pointer Handling agar bisa NULL di DB)
	if workType == "contractor" && contractorIDStr != "" {
		if idVal, err := strconv.Atoi(contractorIDStr); err == nil {
			cID := uint(idVal)
			user.ContractorID = &cID
			user.DepartmentID = nil
		}
	} else if workType == "department" && departmentIDStr != "" {
		if idVal, err := strconv.Atoi(departmentIDStr); err == nil {
			dID := uint(idVal)
			user.DepartmentID = &dID
			user.ContractorID = nil
		}
	} else {
		user.ContractorID = nil
		user.DepartmentID = nil
	}

	// 7. Eksekusi Update Data User Utama ke Database
	err := uc.DB.Model(&user).
		Select("Name", "Username", "Email", "Password", "EmployeeID", "DateOfBirth", "Gender", "RoleID", "ContractorID", "DepartmentID", "IsPIC").
		Updates(user).Error

	if err != nil {
		uc.setFlash(w, "Gagal mengupdate data user: "+err.Error(), "error")
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return // FIXED: Jika update utama gagal, jangan lanjut ke update relasi
	}

	// 8. UPDATE MODERATOR CATEGORY (OPTIMASI: Menggunakan 1 Query IN)
	var categories []models.EventCategory
	if len(moderatorCategoryIDs) > 0 {
		var catIDs []uint
		for _, idStr := range moderatorCategoryIDs {
			if idUint, err := strconv.ParseUint(idStr, 10, 64); err == nil {
				catIDs = append(catIDs, uint(idUint))
			}
		}

		if len(catIDs) > 0 {
			uc.DB.Where("id IN ? AND parent_id IS NULL", catIDs).Find(&categories)
		}
	}

	// Replace hubungan Many-to-Many di tabel pivot
	if err := uc.DB.Model(&user).Association("ModeratedCategories").Replace(categories); err != nil {
		uc.setFlash(w, "Data user terupdate, namun gagal memperbarui kategori moderator: "+err.Error(), "error")
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return
	}

	uc.setFlash(w, "Data user berhasil diupdate", "success")
	http.Redirect(w, r, "/people", http.StatusSeeOther)
}
func (uc *UserController) Delete(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil ID dari Form (POST) atau URL Query (GET)
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id != "" {
		// 2. Gunakan Unscoped untuk menghapus permanen dari database
		// Ini akan menghapus data di tabel 'users' berdasarkan ID yang diberikan
		err := uc.DB.Unscoped().Delete(&models.User{}, id).Error

		if err != nil {
			uc.setFlash(w, "Gagal menghapus user: "+err.Error(), "error")
		} else {
			uc.setFlash(w, "User berhasil dihapus secara permanen", "success")
		}
	} else {
		uc.setFlash(w, "ID user tidak ditemukan", "error")
	}

	// 3. Redirect kembali ke halaman manajemen user
	http.Redirect(w, r, "/people", http.StatusSeeOther)
}

func (uc *UserController) UploadExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return
	}

	file, _, err := r.FormFile("excel_file")
	if err != nil {
		uc.setFlash(w, "Gagal mengambil file", "error")
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		uc.setFlash(w, "Format file tidak didukung", "error")
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		uc.setFlash(w, "Gagal membaca data excel", "error")
		http.Redirect(w, r, "/people", http.StatusSeeOther)
		return
	}

	tx := uc.DB.Begin()
	// Menggunakan password default Msm12345
	// Default password fallback
	defaultPass, _ := bcrypt.GenerateFromPassword(
		[]byte("Msm12345"),
		bcrypt.DefaultCost,
	)

	for i, row := range rows {
		// Lewati header dan pastikan ada data Nama di kolom pertama
		if i == 0 || len(row) < 1 {
			continue
		}

		name := strings.TrimSpace(row[0])
		if name == "" {
			continue
		}

		// Ambil data dengan proteksi index out of range
		usernameExcel := ""
		if len(row) > 1 {
			usernameExcel = strings.TrimSpace(row[1])
		}

		emailExcel := ""
		if len(row) > 2 {
			emailExcel = strings.TrimSpace(row[2])
		}

		roleNameExcel := ""
		if len(row) > 3 {
			roleNameExcel = strings.TrimSpace(row[3])
		}

		unitNameExcel := ""
		if len(row) > 4 {
			unitNameExcel = strings.TrimSpace(row[4])
		}

		passwordExcel := ""
		if len(row) > 5 {
			passwordExcel = strings.TrimSpace(row[5])
		}

		employeeIDExcel := ""
		if len(row) > 6 {
			employeeIDExcel = strings.TrimSpace(row[6])
		}

		// --- TAMBAHAN BARU: Ekstraksi Tanggal Lahir & Jenis Kelamin ---
		dateOfBirthExcel := ""
		if len(row) > 7 {
			dateOfBirthExcel = strings.TrimSpace(row[7])
		}

		genderExcel := ""
		if len(row) > 8 {
			// Ditrim dan paksa ke UPPERCASE supaya input 'l' atau 'p' kecil tetap seragam jadi 'L' / 'P'
			genderExcel = strings.ToUpper(strings.TrimSpace(row[8]))
		}

		// --- SOLUSI DUPLICATE ENTRY: Generate Unique Fallback ---
		finalUsername := usernameExcel
		if finalUsername == "" {
			// Jika kosong, buat username: nama.kecil.indeks (contoh: budi.1)
			cleanName := strings.ReplaceAll(strings.ToLower(name), " ", ".")
			finalUsername = fmt.Sprintf("%s.%d", cleanName, i)
		}

		finalEmail := emailExcel
		if finalEmail == "" {
			// Jika kosong, buat email dummy: nama.indeks@sentry.local
			cleanName := strings.ReplaceAll(strings.ToLower(name), " ", ".")
			finalEmail = fmt.Sprintf("%s.%d@sentry.local", cleanName, i)
		}

		var user models.User
		var roleID uint
		var contractorID, departmentID *uint

		// --- 1. Cari Role ID ---
		if roleNameExcel != "" {
			var role models.Role
			if err := tx.Where("LOWER(name) = LOWER(?)", roleNameExcel).First(&role).Error; err == nil {
				roleID = role.ID
			}
		}

		// --- 2. Cari Unit Kerja ---
		if unitNameExcel != "" {
			var con models.Contractor
			if errCon := tx.Where("LOWER(name) = LOWER(?)", unitNameExcel).First(&con).Error; errCon == nil {
				contractorID = &con.ID
			} else {
				var dept models.Department
				if errDept := tx.Where("LOWER(name) = LOWER(?)", unitNameExcel).First(&dept).Error; errDept == nil {
					departmentID = &dept.ID
				}
			}
		}

		// --- 3. Upsert Logic ---
		// Cari user yang sudah ada berdasarkan email atau nama
		tx.Where("email = ? OR name = ?", finalEmail, name).First(&user)

		user.Name = name
		user.Username = finalUsername
		user.Email = finalEmail

		// =========================
		// PASSWORD
		// =========================

		// Jika create user baru
		if user.ID == 0 {
			// Pakai password dari excel jika ada
			if passwordExcel != "" {
				user.Password = passwordExcel
			} else {
				// fallback default bcrypt
				user.Password = string(defaultPass)
			}
		} else {
			// Jika update user existing
			// hanya update password bila kolom F diisi
			if passwordExcel != "" {
				user.Password = passwordExcel
			}
		}

		user.RoleID = roleID
		user.ContractorID = contractorID
		user.DepartmentID = departmentID
		user.EmployeeID = employeeIDExcel

		// --- TAMBAHAN BARU: Parsing Tanggal Lahir & Map ke Model ---
		var parsedDOB *time.Time
		if dateOfBirthExcel != "" {
			// Format layout default: "YYYY-MM-DD" (Contoh: 1997-04-28)
			// Jika format di Excel Anda adalah "DD-MM-YYYY", ubah layout menjadi "02-01-2006"
			t, err := time.Parse("2006-01-02", dateOfBirthExcel)
			if err == nil {
				parsedDOB = &t
			}
		}
		user.DateOfBirth = parsedDOB
		user.Gender = genderExcel

		// Eksekusi Save
		if err := tx.Save(&user).Error; err != nil {
			tx.Rollback()
			uc.setFlash(w, fmt.Sprintf("Gagal simpan baris %d: %v", i+1, err), "error")
			http.Redirect(w, r, "/people", http.StatusSeeOther)
			return
		}
	}

	tx.Commit()
	uc.setFlash(w, "Berhasil mengimpor data user", "success")
	http.Redirect(w, r, "/people", http.StatusSeeOther)
}

// Fungsi helper agar kode lebih bersih (opsional)
func (lc *UserController) setFlash(w http.ResponseWriter, msg string, msgType string) {
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
