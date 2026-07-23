package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"latihan1/models"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WpiReportService struct {
	DB *gorm.DB
}

var (
	ErrUnsupportedWpiFile = errors.New("format dokumen WPI tidak didukung")
	ErrWpiStorage         = errors.New("gagal menyimpan dokumen WPI")
	ErrWpiUploadTooLarge  = errors.New("ukuran upload WPI melebihi batas")
)

func NewWpiReportService(db *gorm.DB) *WpiReportService {
	return &WpiReportService{
		DB: db,
	}
}

func (s *WpiReportService) CreateWpiReport(r *http.Request, reviewerID uint) (models.WpiReport, error) {
	// Parse standard form data
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		return models.WpiReport{}, err
	}

	tanggalJamStr := r.FormValue("tanggal_jam")

	// Parse Date & Time from 'YYYY-MM-DD HH:MM' format
	loc, _ := time.LoadLocation("Asia/Makassar")
	parsedTime, err := time.ParseInLocation("2006-01-02 15:04", tanggalJamStr, loc)
	if err != nil {
		return models.WpiReport{}, fmt.Errorf("format waktu inspeksi tidak valid: %v", err)
	}

	report := models.WpiReport{
		TanggalJam: parsedTime,
		ReviewerID: &reviewerID,
	}

	// Master data references
	if v := r.FormValue("location_id"); v != "" {
		id, _ := parseUint(v)
		report.LocationID = &id
	}
	report.LocationSpecific = r.FormValue("location_specific")
	report.SiteName = r.FormValue("site_name")
	report.Area = r.FormValue("area")

	if v := r.FormValue("company_id"); v != "" {
		id, _ := parseUint(v)
		report.CompanyID = &id
	}
	workType := r.FormValue("work_type")
	if workType == "department" {
		if v := r.FormValue("department_id"); v != "" {
			id, _ := parseUint(v)
			report.DepartmentID = &id
		}
		report.ContractorID = nil
	} else if workType == "contractor" {
		if v := r.FormValue("contractor_id"); v != "" {
			id, _ := parseUint(v)
			report.ContractorID = &id
		}
		report.DepartmentID = nil
	}

	// Parse JSON for dynamic lists (Inspectors & Items)
	inspectorsJSON := r.FormValue("inspectors_json")
	var inspectors []models.WpiInspector
	if inspectorsJSON != "" {
		if err := json.Unmarshal([]byte(inspectorsJSON), &inspectors); err != nil {
			return models.WpiReport{}, fmt.Errorf("gagal parse data inspektur: %v", err)
		}
	}
	// Salin Input (dari JSON string Alpine.js) ke field DB masing-masing
	for i := range inspectors {
		if inspectors[i].InspectorIDInput.Valid {
			v := inspectors[i].InspectorIDInput.Val
			inspectors[i].InspectorID = &v
		}
		// Nullify berdasarkan work_type per baris inspektur & salin value
		if inspectors[i].InspectorWorkType == "contractor" {
			inspectors[i].DepartmentID = nil
			if inspectors[i].ContractorIDInput.Valid {
				v := inspectors[i].ContractorIDInput.Val
				inspectors[i].ContractorID = &v
			} else {
				inspectors[i].ContractorID = nil
			}
		} else {
			inspectors[i].ContractorID = nil
			if inspectors[i].DepartmentIDInput.Valid {
				v := inspectors[i].DepartmentIDInput.Val
				inspectors[i].DepartmentID = &v
			} else {
				inspectors[i].DepartmentID = nil
			}
		}
	}
	report.Inspectors = inspectors

	itemsJSON := r.FormValue("items_json")
	var items []models.WpiItem
	if itemsJSON != "" {
		if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
			return models.WpiReport{}, fmt.Errorf("gagal parse data temuan: %v", err)
		}
	}
	// Salin Input (dari JSON string Alpine.js) ke field DB
	for i := range items {
		if items[i].PicIDInput.Valid {
			v := items[i].PicIDInput.Val
			items[i].PicID = &v
		}
		if items[i].DueDateInput.Valid {
			v := items[i].DueDateInput.Val
			items[i].DueDate = &v
		}
		if items[i].CompletionDateInput.Valid {
			v := items[i].CompletionDateInput.Val
			items[i].CompletionDate = &v
		}
		if err := s.attachItemDocuments(r, &items[i], i, nil); err != nil {
			return models.WpiReport{}, err
		}
	}
	report.Items = items

	// Gunakan Transaction agar jika salah satu gagal, semuanya batal (Rollback)
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		reference, err := s.GenerateReference(tx)
		if err != nil {
			return err
		}
		report.Reference = reference
		if err := tx.Create(&report).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		removeNewItemDocuments(report.Items, nil)
	}
	return report, err
}

func (s *WpiReportService) GetByID(id string) (models.WpiReport, error) {
	var report models.WpiReport
	err := s.DB.Preload("Location").
		Preload("Company").
		Preload("Department").
		Preload("Contractor").
		Preload("Reviewer").
		Preload("Inspectors.Inspector").
		Preload("Inspectors.Department").
		Preload("Inspectors.Contractor").
		Preload("Items.PIC").
		First(&report, id).Error
	return report, err
}

func (s *WpiReportService) Update(id string, r *http.Request, reviewerID uint) (models.WpiReport, error) {
	// 1. Ambil data lama
	report, err := s.GetByID(id)
	if err != nil {
		return report, err
	}

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		return models.WpiReport{}, err
	}

	// 2. Parse Date & Time from 'YYYY-MM-DD HH:MM' format
	tanggalJamStr := r.FormValue("tanggal_jam")
	loc, _ := time.LoadLocation("Asia/Makassar")
	parsedTime, err := time.ParseInLocation("2006-01-02 15:04", tanggalJamStr, loc)
	if err != nil {
		return models.WpiReport{}, fmt.Errorf("format waktu inspeksi tidak valid: %v", err)
	}

	report.TanggalJam = parsedTime
	report.ReviewerID = &reviewerID

	// Master data references
	if v := r.FormValue("location_id"); v != "" {
		idLoc, _ := parseUint(v)
		report.LocationID = &idLoc
	}
	report.LocationSpecific = r.FormValue("location_specific")
	report.SiteName = r.FormValue("site_name")
	report.Area = r.FormValue("area")

	if v := r.FormValue("company_id"); v != "" {
		idComp, _ := parseUint(v)
		report.CompanyID = &idComp
	} else {
		report.CompanyID = nil
	}

	workType := r.FormValue("work_type")
	if workType == "department" {
		if v := r.FormValue("department_id"); v != "" {
			idDept, _ := parseUint(v)
			report.DepartmentID = &idDept
		}
		report.ContractorID = nil
	} else if workType == "contractor" {
		if v := r.FormValue("contractor_id"); v != "" {
			idCont, _ := parseUint(v)
			report.ContractorID = &idCont
		}
		report.DepartmentID = nil
	}

	// Parse JSON for dynamic lists (Inspectors & Items)
	inspectorsJSON := r.FormValue("inspectors_json")
	var inspectors []models.WpiInspector
	if inspectorsJSON != "" {
		if err := json.Unmarshal([]byte(inspectorsJSON), &inspectors); err != nil {
			return models.WpiReport{}, fmt.Errorf("gagal parse data inspektur: %v", err)
		}
	}
	// Salin Input (dari JSON string Alpine.js) ke field DB masing-masing
	for i := range inspectors {
		if inspectors[i].InspectorIDInput.Valid {
			v := inspectors[i].InspectorIDInput.Val
			inspectors[i].InspectorID = &v
		}
		if inspectors[i].InspectorWorkType == "contractor" {
			inspectors[i].DepartmentID = nil
			if inspectors[i].ContractorIDInput.Valid {
				v := inspectors[i].ContractorIDInput.Val
				inspectors[i].ContractorID = &v
			} else {
				inspectors[i].ContractorID = nil
			}
		} else {
			inspectors[i].ContractorID = nil
			if inspectors[i].DepartmentIDInput.Valid {
				v := inspectors[i].DepartmentIDInput.Val
				inspectors[i].DepartmentID = &v
			} else {
				inspectors[i].DepartmentID = nil
			}
		}
		// Set Report ID explicitly for safety
		inspectors[i].WpiReportID = report.ID
	}

	itemsJSON := r.FormValue("items_json")
	var items []models.WpiItem
	if itemsJSON != "" {
		if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
			return models.WpiReport{}, fmt.Errorf("gagal parse data temuan: %v", err)
		}
	}
	for i := range items {
		if items[i].PicIDInput.Valid {
			v := items[i].PicIDInput.Val
			items[i].PicID = &v
		}
		if items[i].DueDateInput.Valid {
			v := items[i].DueDateInput.Val
			items[i].DueDate = &v
		}
		if items[i].CompletionDateInput.Valid {
			v := items[i].CompletionDateInput.Val
			items[i].CompletionDate = &v
		}
		var existing *models.WpiItem
		if i < len(report.Items) {
			existing = &report.Items[i]
		}
		if err := s.attachItemDocuments(r, &items[i], i, existing); err != nil {
			return models.WpiReport{}, err
		}
		// Set Report ID explicitly for safety
		items[i].WpiReportID = report.ID
	}

	// Gunakan Transaction
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		// Update header
		if err := tx.Save(&report).Error; err != nil {
			return err
		}

		// Delete old detail records
		if err := tx.Where("wpi_report_id = ?", report.ID).Delete(&models.WpiInspector{}).Error; err != nil {
			return err
		}
		if err := tx.Where("wpi_report_id = ?", report.ID).Delete(&models.WpiItem{}).Error; err != nil {
			return err
		}

		// Insert new detail records (if any)
		if len(inspectors) > 0 {
			// Clear IDs so GORM inserts as new records
			for i := range inspectors {
				inspectors[i].ID = 0
			}
			if err := tx.Create(&inspectors).Error; err != nil {
				return err
			}
		}
		if len(items) > 0 {
			// Clear IDs
			for i := range items {
				items[i].ID = 0
			}
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		removeNewItemDocuments(items, report.Items)
	}
	if err == nil {
		deleted := valuesToSet(append(r.MultipartForm.Value["deleted_doc_uraian_tindakan[]"], r.MultipartForm.Value["deleted_doc_jenis_tindakan[]"]...))
		var filesToDelete []string
		for _, oldItem := range report.Items {
			for _, url := range append(documentURLs(oldItem.DocUraianTindakan), documentURLs(oldItem.DocJenisTindakan)...) {
				if deleted[url] {
					filesToDelete = append(filesToDelete, url)
				}
			}
		}
		deleteWpiFiles(filesToDelete)
	}

	return report, err
}

// GenerateReference mengikuti format referensi Hazard/Incident: HZ-YYYYMMDD-0001.
// Lock mencegah dua laporan yang dibuat bersamaan mendapatkan nomor yang sama.
func (s *WpiReportService) GenerateReference(db *gorm.DB) (string, error) {
	prefix := "WPI-" + time.Now().Format("20060102") + "-"
	var last models.WpiReport
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("reference LIKE ?", prefix+"%").
		Order("reference DESC").
		First(&last).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	if last.ID == 0 {
		return prefix + "0001", nil
	}
	n, err := strconv.Atoi(strings.TrimPrefix(last.Reference, prefix))
	if err != nil {
		return "", fmt.Errorf("format reference WPI terakhir tidak valid: %w", err)
	}
	return fmt.Sprintf("%s%04d", prefix, n+1), nil
}

// attachItemDocuments menyimpan URL dokumen sebagai JSON pada tiap item. Field file
// diberi indeks supaya beberapa temuan dalam satu form tidak saling tertukar.
func (s *WpiReportService) attachItemDocuments(r *http.Request, item *models.WpiItem, index int, existing *models.WpiItem) error {
	deletedUraian := valuesToSet(r.MultipartForm.Value["deleted_doc_uraian_tindakan[]"])
	deletedJenis := valuesToSet(r.MultipartForm.Value["deleted_doc_jenis_tindakan[]"])

	var oldUraian, oldJenis []string
	if existing != nil {
		oldUraian = documentURLs(existing.DocUraianTindakan)
		oldJenis = documentURLs(existing.DocJenisTindakan)
	}

	newUraian, err := saveWpiFiles(r, fmt.Sprintf("doc_uraian_tindakan_%d", index))
	if err != nil {
		return err
	}
	newJenis, err := saveWpiFiles(r, fmt.Sprintf("doc_jenis_tindakan_%d", index))
	if err != nil {
		deleteWpiFiles(newUraian)
		return err
	}

	uraian := append(filterDocuments(oldUraian, deletedUraian), newUraian...)
	jenis := append(filterDocuments(oldJenis, deletedJenis), newJenis...)
	item.DocUraianTindakan = marshalDocumentURLs(uraian)
	item.DocJenisTindakan = marshalDocumentURLs(jenis)
	return nil
}

func saveWpiFiles(r *http.Request, fieldName string) ([]string, error) {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, nil
	}
	files := r.MultipartForm.File[fieldName]
	if len(files) == 0 {
		return nil, nil
	}
	// Penyimpanan tidak berada di public root; dokumen hanya dapat diakses lewat
	// endpoint terautentikasi yang memvalidasi nama file.
	folder := "./storage/uploads/wpi-reports"
	if err := os.MkdirAll(folder, 0755); err != nil {
		return nil, fmt.Errorf("%w: membuat direktori", ErrWpiStorage)
	}
	urls := make([]string, 0, len(files))
	for _, header := range files {
		if header.Size > 5<<20 {
			deleteWpiFiles(urls)
			return nil, fmt.Errorf("%w: ukuran file melebihi 5 MB", ErrWpiUploadTooLarge)
		}
		source, err := header.Open()
		if err != nil {
			deleteWpiFiles(urls)
			return nil, fmt.Errorf("%w: membuka file", ErrWpiStorage)
		}
		sniff := make([]byte, 512)
		n, _ := io.ReadFull(source, sniff)
		mimeType := http.DetectContentType(sniff[:n])
		_, ok := map[string]string{
			"application/pdf": ".pdf",
			"image/jpeg":      ".jpg",
			"image/png":       ".png",
		}[mimeType]
		if !ok {
			source.Close()
			deleteWpiFiles(urls)
			return nil, ErrUnsupportedWpiFile
		}
		if _, err := source.Seek(0, io.SeekStart); err != nil {
			source.Close()
			deleteWpiFiles(urls)
			return nil, fmt.Errorf("%w: membaca file", ErrWpiStorage)
		}
		name := sanitizeWpiUploadFilename(header.Filename)
		destination, err := os.OpenFile(filepath.Join(folder, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			_, err = io.Copy(destination, source)
			destination.Close()
		}
		source.Close()
		if err != nil {
			deleteWpiFiles(urls)
			return nil, fmt.Errorf("%w: menulis file", ErrWpiStorage)
		}
		urls = append(urls, "/wpi-report/document/"+name)
	}
	return urls, nil
}

func sanitizeWpiUploadFilename(filename string) string {
	name := filepath.Base(strings.TrimSpace(filename))
	var cleaned strings.Builder
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '.' || char == '-' || char == '_' {
			cleaned.WriteRune(char)
		} else {
			cleaned.WriteByte('_')
		}
	}
	name = strings.Trim(cleaned.String(), "._")
	if name == "" || name == "." || name == ".." {
		return "document"
	}
	return name
}

func documentURLs(value *string) []string {
	if value == nil || *value == "" {
		return nil
	}
	var urls []string
	if json.Unmarshal([]byte(*value), &urls) != nil {
		return nil
	}
	return urls
}

func marshalDocumentURLs(urls []string) *string {
	if len(urls) == 0 {
		return nil
	}
	b, _ := json.Marshal(urls)
	value := string(b)
	return &value
}

func valuesToSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func filterDocuments(urls []string, deleted map[string]bool) []string {
	kept := make([]string, 0, len(urls))
	for _, url := range urls {
		if !deleted[url] {
			kept = append(kept, url)
		}
	}
	return kept
}

func deleteWpiFiles(urls []string) {
	for _, url := range urls {
		if !strings.HasPrefix(url, "/wpi-report/document/") {
			continue
		}
		_ = os.Remove(filepath.Join("./storage/uploads/wpi-reports", filepath.Base(url)))
	}
}

// Hapus hanya file yang baru ditulis jika transaksi database gagal; dokumen lama
// tetap dipertahankan agar edit yang gagal tidak menyebabkan kehilangan data.
func removeNewItemDocuments(items, oldItems []models.WpiItem) {
	oldURLs := make(map[string]bool)
	for _, item := range oldItems {
		for _, url := range append(documentURLs(item.DocUraianTindakan), documentURLs(item.DocJenisTindakan)...) {
			oldURLs[url] = true
		}
	}
	var newURLs []string
	for _, item := range items {
		for _, url := range append(documentURLs(item.DocUraianTindakan), documentURLs(item.DocJenisTindakan)...) {
			if !oldURLs[url] {
				newURLs = append(newURLs, url)
			}
		}
	}
	deleteWpiFiles(newURLs)
}

type WpiReportIndexResult struct {
	Reports     []models.WpiReport
	TotalRows   int64
	TotalPages  int
	CurrentPage int
	HasPrev     bool
	HasNext     bool
	Search      string
	LocationID  string
	UnitFilter  string
}

func (s *WpiReportService) GetWpiReports(search, locationID, unitFilter, startDate, endDate string, page int) (WpiReportIndexResult, error) {
	if page < 1 {
		page = 1
	}
	pageSize := 10
	offset := (page - 1) * pageSize

	var reports []models.WpiReport
	var totalRows int64

	dbQuery := s.DB.Model(&models.WpiReport{}).
		Preload("Location").
		Preload("Company").
		Preload("Department").
		Preload("Contractor").
		Preload("Reviewer")

	if search != "" {
		searchLike := "%" + search + "%"
		dbQuery = dbQuery.
			Joins("LEFT JOIN locations ON locations.id = wpi_reports.location_id").
			Joins("LEFT JOIN departments ON departments.id = wpi_reports.department_id").
			Joins("LEFT JOIN contractors ON contractors.id = wpi_reports.contractor_id").
			Joins("LEFT JOIN companies ON companies.id = wpi_reports.company_id").
			Joins("LEFT JOIN users ON users.id = wpi_reports.reviewer_id").
			Where(`
				wpi_reports.site_name LIKE ?
				OR wpi_reports.area LIKE ?
				OR locations.name LIKE ?
				OR departments.name LIKE ?
				OR contractors.name LIKE ?
				OR companies.name LIKE ?
				OR wpi_reports.reference LIKE ?
				OR users.name LIKE ?
			`, searchLike, searchLike, searchLike, searchLike, searchLike, searchLike, searchLike, searchLike)
	}
	if locationID != "" {
		dbQuery = dbQuery.Where("wpi_reports.location_id = ?", locationID)
	}
	inspectionLocation, _ := time.LoadLocation("Asia/Makassar")
	if startDate != "" {
		start, err := time.ParseInLocation("2006-01-02", startDate, inspectionLocation)
		if err != nil {
			return WpiReportIndexResult{}, fmt.Errorf("format tanggal mulai tidak valid: %w", err)
		}
		dbQuery = dbQuery.Where("wpi_reports.tanggal_jam >= ?", start)
	}
	if endDate != "" {
		end, err := time.ParseInLocation("2006-01-02", endDate, inspectionLocation)
		if err != nil {
			return WpiReportIndexResult{}, fmt.Errorf("format tanggal akhir tidak valid: %w", err)
		}
		// Batas eksklusif pada hari berikutnya mencakup seluruh waktu inspeksi
		// di tanggal akhir tanpa membungkus kolom dengan fungsi DATE().
		dbQuery = dbQuery.Where("wpi_reports.tanggal_jam < ?", end.AddDate(0, 0, 1))
	}
	if parts := strings.SplitN(unitFilter, ":", 2); len(parts) == 2 && parts[1] != "" {
		switch parts[0] {
		case "department":
			dbQuery = dbQuery.Where("wpi_reports.department_id = ?", parts[1])
		case "contractor":
			dbQuery = dbQuery.Where("wpi_reports.contractor_id = ?", parts[1])
		}
	}

	// Count total rows
	if err := dbQuery.Count(&totalRows).Error; err != nil {
		return WpiReportIndexResult{}, err
	}

	// Get page rows
	if err := dbQuery.Order("wpi_reports.tanggal_jam DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&reports).Error; err != nil {
		return WpiReportIndexResult{}, err
	}

	totalPages := int((totalRows + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	return WpiReportIndexResult{
		Reports:     reports,
		TotalRows:   totalRows,
		TotalPages:  totalPages,
		CurrentPage: page,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		Search:      search,
		LocationID:  locationID,
		UnitFilter:  unitFilter,
	}, nil
}
