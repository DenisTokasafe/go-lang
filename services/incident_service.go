package services

import (
	"fmt"
	"io"
	"latihan1/cmd/web/config"
	"latihan1/cmd/web/helpers"
	"latihan1/models"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type IncidentService interface {
	GetFormDataReferences(page int) (map[string]interface{}, error)
	CreateIncident(incident *models.IncidentReport, parties []models.InvolvedParty) error
	// Tambahkan r *http.Request di akhir parameter ini:
	UpdateIncident(id uint, userID uint, updatedIncident *models.IncidentReport, parties []models.InvolvedParty, investigators []models.InvestigationParticipant, peepoFactors []models.PeepoFactor, timelines []models.Timeline, causes []models.IncidentCause, corrective_action_incidents []models.CorrectiveActionIncident, reviews *models.IncidentReview, r *http.Request) (bool, error)
	GetByID(id uint) (*models.IncidentReport, error)
	GetEditData(id uint, currentUserID uint, page int) (*IncidentEditData, error)
}

type incidentService struct {
	db *gorm.DB
}

func NewIncidentService(db *gorm.DB) IncidentService {
	return &incidentService{db: db}

}

// GetFormDataReferences mengambil semua data master untuk dropdown di form KPLH
func (s *incidentService) GetFormDataReferences(page int) (map[string]interface{}, error) {
	var eventCategories []models.EventCategory
	var locations []models.Location
	var riskMatrices []models.RiskMatrix
	var departments []models.Department
	var contractors []models.Contractor
	var users []models.User
	var scatOptions []models.ScatOption
	var consequences []models.RiskConsequence
	var likelihoods []models.RiskLikelihood
	var assessments []models.RiskAssessmentCode

	// Ambil data referensi dasar
	s.db.Order("name asc").Limit(20).Find(&locations)
	s.db.Order("risk_consequence_id asc").Limit(50).Find(&riskMatrices)
	s.db.Order("name asc").Limit(20).Find(&departments)
	s.db.Order("name asc").Limit(20).Find(&contractors)
	s.db.Order("name asc").Limit(20).Find(&users)

	// Ambil Kategori khusus Incident ("%INC%")
	s.db.Where("parent_id IS NULL AND category_group = ? AND code LIKE ?", "incident", "%INC%").
		Order("name asc").Find(&eventCategories)

	// Ambil Scat Options
	s.db.Where("type IN ?", []string{"unsafe_act", "personal_factor"}).
		Order("FIELD(type, 'unsafe_act', 'personal_factor'), code asc").
		Find(&scatOptions)

	// Ambil komponen Risk Matrix
	s.db.Order("severity_level DESC").Find(&consequences)
	s.db.Order("sequence ASC").Find(&likelihoods)
	s.db.Find(&assessments)

	// ==========================================
	// Logika Pagination Matriks Risiko (Grid 5x5)
	// ==========================================
	pageSize := 25
	offset := (page - 1) * pageSize

	var matrices []models.RiskMatrix
	var totalRows int64

	dbQuery := s.db.Model(&models.RiskMatrix{}).
		Preload("RiskConsequence").
		Preload("RiskLikelihood").
		Preload("RiskAssessment").
		Joins("JOIN risk_consequences ON risk_consequences.id = risk_matrices.risk_consequence_id").
		Joins("JOIN risk_likelihoods ON risk_likelihoods.id = risk_matrices.risk_likelihood_id").
		Joins("JOIN risk_assessment_codes ON risk_assessment_codes.id = risk_matrices.risk_assessment_id")

	dbQuery.Count(&totalRows)
	dbQuery.Limit(pageSize).Offset(offset).
		Order("risk_consequences.severity_level DESC, risk_likelihoods.sequence ASC").
		Find(&matrices)

	totalPages := int((totalRows + int64(pageSize) - 1) / int64(pageSize))

	// Bungkus semua ke dalam map
	refs := map[string]interface{}{
		"EventCategories": eventCategories,
		"Locations":       locations,
		"RiskMatrices":    riskMatrices,
		"Departments":     departments,
		"Contractors":     contractors,
		"Users":           users,
		"ScatOptions":     scatOptions,
		"Consequences":    consequences,
		"Likelihoods":     likelihoods,
		"Assessments":     assessments,
		"Matrices":        matrices,
		"TotalRows":       totalRows,
		"TotalPages":      totalPages,
	}

	return refs, nil
}

// CreateIncident menyimpan Laporan beserta baris anak secara transaksional
func (s *incidentService) CreateIncident(incident *models.IncidentReport, parties []models.InvolvedParty) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Simpan Laporan Induk (Bagian 1)
		if err := tx.Create(incident).Error; err != nil {
			return err // Otomatis Rollback Laporan Utama
		}

		// 2. Jika ada data Pihak Terlibat, proses penyimpanannya
		if len(parties) > 0 {
			// Hubungkan ID Laporan Utama yang baru di-generate ke Foreign Key child-nya
			for i := range parties {
				parties[i].IncidentReportID = incident.ID
			}

			// CUKUP PANGGIL SATU KALI DI SINI (Secara Batch Insert)
			if err := tx.Create(&parties).Error; err != nil {

				return err // Otomatis Rollback Laporan Utama & Batalkan Insert Pihak Terlibat
			}
		}

		return nil // Otomatis Commit seluruh data jika sampai di titik ini
	})
}

type FileDTOIncident struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	IsImage bool   `json:"is_image"`
}

func (s *incidentService) GetByID(id uint) (*models.IncidentReport, error) {
	var incident models.IncidentReport

	err := s.db.
		Preload("RiskMatrix").
		Preload("EventCategory").
		Preload("ScatOption").
		Preload("Department").
		Preload("Contractor").
		Preload("Location").
		Preload("PIC").
		// Pihak terlibat
		Preload("InvolvedParties").
		Preload("InvolvedParties.Department").
		Preload("InvolvedParties.Contractor").
		Preload("InvolvedParties.ReportBy").
		// =================================================================
		// BARU: Preload Partisipan Investigasi & Sub-Relasinya
		// =================================================================
		Preload("InvestigationParticipants").
		Preload("InvestigationParticipants.Department").
		Preload("InvestigationParticipants.Contractor").
		Preload("InvestigationParticipants.ReportBy"). // Sesuai nama field di struct Anda
		// =================================================================
		Preload("PeepoFactors").
		Preload("Timelines").
		Preload("Causes").
		Preload("CorrectiveActionIncidents").
		Preload("Documentations.Documentation").
		First(&incident, id).Error

	if err != nil {
		return nil, err
	}

	return &incident, nil
}

// Pastikan struct FileDTO sudah ada di package Anda (biasanya digunakan bersama di hazard)
// Jika belum, Anda bisa mendefinisikannya:
// type FileDTO struct { ID uint `json:"id"`; URL string `json:"url"` }

type IncidentEditData struct {
	Incident    models.IncidentReport
	CanReopen   bool
	IsModerator bool
	Docs        []FileDTOIncident // Insiden tidak memisah desc/corrective
	WorkType    string

	// Master Data
	EventCategories []models.EventCategory
	Locations       []models.Location
	RiskMatrices    []models.RiskMatrix
	Departments     []models.Department
	Contractors     []models.Contractor
	Users           []models.User
	ScatOptions     []models.ScatOption
	ScatOptionsAll  []models.ScatOption
	Consequences    []models.RiskConsequence
	Likelihoods     []models.RiskLikelihood
	Assessments     []models.RiskAssessmentCode
	Matrices        []models.RiskMatrix
	AllTypes        []models.EventCategory
	AllPics         []models.User

	// Pagination Data untuk Matrix
	TotalRows  int64
	TotalPages int
	PageSize   int
}

func (s *incidentService) GetEditData(id uint, currentUserID uint, page int) (*IncidentEditData, error) {
	var result IncidentEditData
	var incident models.IncidentReport

	// ==========================================
	// 1. GET INCIDENT DENGAN SEMUA RELASI
	// ==========================================
	err := s.db.
		Preload("EventCategory").
		Preload("RiskMatrix").
		Preload("ScatOption").
		Preload("Location").
		Preload("ReportBy").
		Preload("Department").
		Preload("Contractor").
		Preload("PIC").
		Preload("InvolvedParties").
		Preload("InvolvedParties.Department").
		Preload("InvolvedParties.Contractor").
		Preload("InvolvedParties.ReportBy").
		// =================================================================
		// BARU: Preload Partisipan Investigasi di halaman Edit
		// =================================================================
		Preload("InvestigationParticipants").
		Preload("InvestigationParticipants.Department").
		Preload("InvestigationParticipants.Contractor").
		Preload("InvestigationParticipants.ReportBy").
		// =================================================================
		Preload("Timelines").
		Preload("Causes").
		Preload("PeepoFactors").
		Preload("Documentations.Documentation").
		Preload("Audits").
		Preload("CorrectiveActionIncidents").
		Preload("Reviews").
		First(&incident, id).Error

	if err != nil {
		return nil, err
	}
	result.Incident = incident

	// ==========================================
	// ⬇️ LETAKKAN BLOK KODE DI SINI ⬇️
	// ==========================================
	deptIDsMap := make(map[uint64]bool)
	contIDsMap := make(map[uint64]bool)
	userIDsMap := make(map[uint64]bool)

	// Dari Insiden Utama
	if incident.DepartmentID != nil && *incident.DepartmentID != 0 {
		deptIDsMap[uint64(*incident.DepartmentID)] = true
	}
	if incident.ContractorID != nil && *incident.ContractorID != 0 {
		contIDsMap[uint64(*incident.ContractorID)] = true
	}
	if incident.PicID != nil && *incident.PicID != 0 {
		userIDsMap[uint64(*incident.PicID)] = true
	}
	if incident.ReportByID != nil && *incident.ReportByID != 0 {
		userIDsMap[uint64(*incident.ReportByID)] = true
	}

	// Dari Involved Parties
	for _, party := range incident.InvolvedParties {
		if party.DepartmentID != nil && *party.DepartmentID != 0 {
			deptIDsMap[uint64(*party.DepartmentID)] = true
		}
		if party.ContractorID != nil && *party.ContractorID != 0 {
			contIDsMap[uint64(*party.ContractorID)] = true
		}
		if party.UserID != nil && *party.UserID != 0 {
			userIDsMap[uint64(*party.UserID)] = true
		}
	}

	// TAMBAHAN: Jika Anda juga ingin mencakup InvestigationParticipants
	for _, inv := range incident.InvestigationParticipants {
		if inv.DepartmentID != nil && *inv.DepartmentID != 0 {
			deptIDsMap[uint64(*inv.DepartmentID)] = true
		}
		if inv.ContractorID != nil && *inv.ContractorID != 0 {
			contIDsMap[uint64(*inv.ContractorID)] = true
		}
		if inv.EmployeID != nil && *inv.EmployeID != 0 {
			userIDsMap[uint64(*inv.EmployeID)] = true
		}
	}

	for _, cai := range incident.CorrectiveActionIncidents {
		if cai.UserID != nil && *cai.UserID != 0 {
			userIDsMap[uint64(*cai.UserID)] = true
		}
	}
	for _, rev := range incident.Reviews {
		if rev.PMUserID != nil && *rev.PMUserID != 0 {
			userIDsMap[uint64(*rev.PMUserID)] = true
		}
		if rev.DeptUserID != nil && *rev.DeptUserID != 0 {
			userIDsMap[uint64(*rev.DeptUserID)] = true
		}
		if rev.OHSUserID != nil && *rev.OHSUserID != 0 {
			userIDsMap[uint64(*rev.OHSUserID)] = true
		}
		if rev.DirOpsUserID != nil && *rev.DirOpsUserID != 0 {
			userIDsMap[uint64(*rev.DirOpsUserID)] = true
		}
		if rev.KTTUserID != nil && *rev.KTTUserID != 0 {
			userIDsMap[uint64(*rev.KTTUserID)] = true
		}

	}

	// Konversi map ke slice untuk query
	var activeDeptIDs []uint64
	for id := range deptIDsMap {
		activeDeptIDs = append(activeDeptIDs, id)
	}

	var activeContIDs []uint64
	for id := range contIDsMap {
		activeContIDs = append(activeContIDs, id)
	}

	var activeUserIDs []uint64
	for id := range userIDsMap {
		activeUserIDs = append(activeUserIDs, id)
	}
	// ==========================================
	// ⬆️ SELESAI PENEMPATAN ⬆️
	// ==========================================

	// ==========================================
	// 2. CHECK PERMISSION & POLICY GUARD (IDOR PROTECTION)
	// ==========================================
	var currentUser models.User
	if err := s.db.Preload("ModeratedCategories").First(&currentUser, currentUserID).Error; err != nil {
		return nil, fmt.Errorf("gagal memverifikasi data user: %w", err)
	}

	result.CanReopen = false
	result.IsModerator = false
	canAccess := false

	// A. Cek jika user adalah PIC
	if currentUser.ID == *incident.PicID {
		result.CanReopen = true
		canAccess = true
	}

	// B. Cek jika user adalah Pelapor
	if incident.ReportByID != nil && currentUser.ID == *incident.ReportByID {
		canAccess = true
	}

	// C. Cek jika user adalah Moderator
	for _, cat := range currentUser.ModeratedCategories {
		if incident.EventCategory.ParentID != nil && cat.ID == *incident.EventCategory.ParentID {
			result.CanReopen = true
			result.IsModerator = true
			canAccess = true
			break
		}
		if cat.ID == incident.EventCategoryID {
			result.CanReopen = true
			result.IsModerator = true
			canAccess = true
			break
		}
	}

	// [CRUCIAL] KUNCI PINTU: Jika bukan Admin/SuperAdmin dan tidak lolos cek, tendang!
	// Hapus komentar ini jika Anda punya check khusus Role Admin
	// if currentUser.Role.Name == "Admin" { canAccess = true }

	if !canAccess {
		return nil, fmt.Errorf("unauthorized: Anda tidak memiliki akses untuk mengedit laporan ini")
	}

	// ==========================================
	// 3. DOCUMENTATIONS
	// ==========================================
	var docsIncident []models.Documentation
	for _, item := range incident.Documentations {
		switch item.DocType {
		case "incident":
			docsIncident = append(docsIncident, item.Documentation)
		}
	}
	result.Docs = s.toFileDTOIncident(docsIncident)
	result.WorkType = "department"
	if incident.ContractorID != nil {
		result.WorkType = "contractor"
	}

	if result.WorkType == "department" && incident.DepartmentID != nil {
		s.db.Where("department_id = ? AND is_pic = ?", *incident.DepartmentID, true).
			Order("name asc").Limit(50).Find(&result.AllPics)
	} else if result.WorkType == "contractor" && incident.ContractorID != nil {
		s.db.Where("contractor_id = ? AND is_pic = ?", *incident.ContractorID, true).
			Order("name asc").Limit(50).Find(&result.AllPics)
	}

	// ==========================================
	// 5. MASTER DATA QUERIES (Dengan Smart Order)
	// ==========================================

	// Kategori khusus Incident
	s.db.Where("parent_id IS NULL AND category_group = ? AND code LIKE ?", "incident", "%INC%").
		Order("name asc").Find(&result.EventCategories)

	s.db.Order("risk_consequence_id asc").Find(&result.RiskMatrices)

	// Location (Urutkan yang terpilih di atas)
	if incident.LocationID != 0 {
		locIDStr := strconv.FormatUint(uint64(incident.LocationID), 10)
		s.db.Where("id = ? OR id NOT IN (?)", incident.LocationID, incident.LocationID).
			Order("CASE WHEN id = " + locIDStr + " THEN 0 ELSE 1 END, name asc").
			Limit(20).Find(&result.Locations)
	} else {
		s.db.Order("name asc").Limit(20).Find(&result.Locations)
	}

	// Department
	// Query Department dengan ID yang aktif dipaksa masuk
	if len(activeDeptIDs) > 0 {
		// Ambil departemen yang terpilih dulu
		var selectedDepts []models.Department
		s.db.Where("id IN ?", activeDeptIDs).Find(&selectedDepts)

		// Ambil sisa departemen (exclude yang sudah terpilih) untuk melengkapi limit
		var otherDepts []models.Department
		s.db.Where("id NOT IN ?", activeDeptIDs).
			Order("name asc").
			Limit(20 - len(selectedDepts)). // Sisa kuota
			Find(&otherDepts)

		result.Departments = append(selectedDepts, otherDepts...)
	} else {
		// Jika tidak ada departemen terpilih, ambil 20 teratas biasa
		s.db.Order("name asc").Limit(20).Find(&result.Departments)
	}

	if len(activeContIDs) > 0 {
		// Ambil contractor yang terpilih dulu
		var selectedConts []models.Contractor
		s.db.Where("id IN ?", activeContIDs).Find(&selectedConts)

		// Ambil sisa contractor (exclude yang sudah terpilih) untuk melengkapi limit
		var otherConts []models.Contractor
		s.db.Where("id NOT IN ?", activeContIDs).
			Order("name asc").
			Limit(20 - len(selectedConts)). // Sisa kuota
			Find(&otherConts)

		result.Contractors = append(selectedConts, otherConts...)
	} else {
		// Jika tidak ada contractor terpilih, ambil 20 teratas biasa
		s.db.Order("name asc").Limit(20).Find(&result.Contractors)
	}
	// Users / ReportBy
	if len(activeUserIDs) > 0 {
		// Ambil user yang terpilih dulu
		var selectedUsers []models.User
		s.db.Where("id IN ?", activeUserIDs).Find(&selectedUsers)

		// Ambil sisa user (exclude yang sudah terpilih) untuk melengkapi limit
		var otherUsers []models.User
		s.db.Where("id NOT IN ?", activeUserIDs).
			Order("name asc").
			Limit(20 - len(selectedUsers)). // Sisa kuota
			Find(&otherUsers)

		result.Users = append(selectedUsers, otherUsers...)
	} else {
		// Jika tidak ada user terpilih, ambil 20 teratas biasa
		s.db.Order("name asc").Limit(20).Find(&result.Users)
	}

	s.db.Where("type IN ?", []string{"unsafe_act", "personal_factor"}).
		Order("FIELD(type, 'unsafe_act', 'personal_factor'), code asc").Find(&result.ScatOptions)

	s.db.Order("code asc").Find(&result.ScatOptionsAll)

	s.db.Order("severity_level DESC").Find(&result.Consequences)
	s.db.Order("sequence ASC").Find(&result.Likelihoods)
	s.db.Find(&result.Assessments)

	// ==========================================
	// 6. PAGINATION MATRIX RISK (Grid)
	// ==========================================
	result.PageSize = 25
	offset := (page - 1) * result.PageSize

	dbQuery := s.db.Model(&models.RiskMatrix{}).
		Preload("RiskConsequence").Preload("RiskLikelihood").Preload("RiskAssessment").
		Joins("JOIN risk_consequences ON risk_consequences.id = risk_matrices.risk_consequence_id").
		Joins("JOIN risk_likelihoods ON risk_likelihoods.id = risk_matrices.risk_likelihood_id").
		Joins("JOIN risk_assessment_codes ON risk_assessment_codes.id = risk_matrices.risk_assessment_id")

	dbQuery.Count(&result.TotalRows)
	dbQuery.Limit(result.PageSize).Offset(offset).
		Order("risk_consequences.severity_level DESC, risk_likelihoods.sequence ASC").
		Find(&result.Matrices)

	result.TotalPages = int((result.TotalRows + int64(result.PageSize) - 1) / int64(result.PageSize))

	// AllTypes (Sub-kategori)
	if incident.EventCategory.ParentID != nil {
		s.db.Where("parent_id = ?", *incident.EventCategory.ParentID).
			Order("name asc").Find(&result.AllTypes)
	} else {
		s.db.Where("parent_id IS NULL").Order("name asc").Find(&result.AllTypes)
	}

	return &result, nil
}

// incident_service.go
type UpdateIncidentRequest struct {
	Incident                  models.IncidentReport             `json:"Incident"`
	InvolvedParties           []models.InvolvedParty            `json:"InvolvedParties"`
	InvestigationParticipants []models.InvestigationParticipant `json:"InvestigationParticipants"`
	PeepoFactors              []models.PeepoFactor              `json:"PeepoFactors"`
	Timelines                 []models.Timeline                 `json:"Timelines"`
	Causes                    []models.IncidentCause            `json:"Causes"`
	CorrectiveActionIncidents []models.CorrectiveActionIncident `json:"CorrectiveActionIncidents"`
	Reviews                   []models.IncidentReview           `json:"Reviews"`
}

// Tambahkan parameter investigators []models.InvestigationParticipant
func (s *incidentService) UpdateIncident(id uint, userID uint,
	updatedIncident *models.IncidentReport,
	parties []models.InvolvedParty,
	investigators []models.InvestigationParticipant,
	peepoFactors []models.PeepoFactor,
	timelines []models.Timeline,
	causes []models.IncidentCause,
	corrective_action_incidents []models.CorrectiveActionIncident,
	reviews *models.IncidentReview,
	r *http.Request) (bool, error) {

	var partiesUpdated bool
	var filesToDeletePhysical []string // List untuk menghapus file fisik pasca commit

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Ambil data lama
		var oldIncident models.IncidentReport
		if err := tx.First(&oldIncident, id).Error; err != nil {
			return err
		}

		// 2. Update Data Utama Incident
		if err := tx.Model(&models.IncidentReport{}).Where("id = ?", id).Updates(updatedIncident).Error; err != nil {
			return err
		}

		// =================================================================
		// 3. Update/Re-sync Involved Parties
		// =================================================================
		if err := tx.Unscoped().Where("incident_report_id = ?", id).Delete(&models.InvolvedParty{}).Error; err != nil {
			return err
		}

		for i := range parties {
			parties[i].IncidentReportID = id
			// Handle pointer cleanup
			if parties[i].ContractorID != nil && *parties[i].ContractorID == 0 {
				parties[i].ContractorID = nil
			}
			if parties[i].DepartmentID != nil && *parties[i].DepartmentID == 0 {
				parties[i].DepartmentID = nil
			}
		}

		if len(parties) > 0 {
			if err := tx.Create(&parties).Error; err != nil {
				return err
			}
			partiesUpdated = true
		}

		// =================================================================
		// 3.5 Update/Re-sync Investigation Participants (BAGIAN BARU)
		// =================================================================
		// Hapus partisipan lama (Hard Delete)
		if err := tx.Unscoped().Where("incident_report_id = ?", id).Delete(&models.InvestigationParticipant{}).Error; err != nil {
			return err
		}

		// Siapkan data partisipan baru
		for i := range investigators {
			investigators[i].IncidentReportID = id

			// Handle pointer cleanup untuk mencegah error relasi (0 menjadi null)
			if investigators[i].EmployeID != nil && *investigators[i].EmployeID == 0 {
				investigators[i].EmployeID = nil
			}
			if investigators[i].DepartmentID != nil && *investigators[i].DepartmentID == 0 {
				investigators[i].DepartmentID = nil
			}
			if investigators[i].ContractorID != nil && *investigators[i].ContractorID == 0 {
				investigators[i].ContractorID = nil
			}
		}

		// Insert partisipan baru jika ada
		if len(investigators) > 0 {
			if err := tx.Create(&investigators).Error; err != nil {
				return err
			}
		}
		// =================================================================

		// =================================================================
		// 4. Update/Re-sync PEEPO Factors
		// =================================================================
		// Hapus data lama
		if err := tx.Unscoped().Where("incident_report_id = ?", id).Delete(&models.PeepoFactor{}).Error; err != nil {
			return err
		}

		// Tambahkan IncidentReportID ke tiap item PEEPO
		for i := range peepoFactors { // Tambahkan peepoFactors ke parameter fungsi
			peepoFactors[i].IncidentReportID = id
		}

		// Insert data baru
		if len(peepoFactors) > 0 {
			if err := tx.Create(&peepoFactors).Error; err != nil {
				return err
			}
		}
		// =================================================================
		// UPDATE/RE-SYNC TIMELINES (BAGIAN BARU)
		// =================================================================
		// Hapus data timeline lama
		if err := tx.Where("incident_report_id = ?", id).Delete(&models.Timeline{}).Error; err != nil {
			return err
		}

		// Siapkan data baru
		for i := range timelines {
			timelines[i].IncidentReportID = id
			// GORM secara otomatis akan menangani datatypes.JSON
			// jika struct Timeline sudah benar
		}

		// Insert data baru
		if len(timelines) > 0 {
			if err := tx.Create(&timelines).Error; err != nil {
				return err
			}
		}
		// =================================================================

		// =================================================================
		// UPDATE/RE-SYNC SCAT CAUSES (BAGIAN BARU)
		// =================================================================
		// Hapus data SCAT lama (Hard Delete)
		if err := tx.Unscoped().Where("incident_report_id = ?", id).Delete(&models.IncidentCause{}).Error; err != nil {
			return err
		}

		// Siapkan data SCAT baru (Pastikan data kosong difilter dari sisi Controller sebelum dikirim kemari)
		for i := range causes {
			causes[i].IncidentReportID = id
		}

		// Insert data baru
		if len(causes) > 0 {
			if err := tx.Create(&causes).Error; err != nil {
				return err
			}
		}
		// =================================================================

		// =================================================================
		// UPDATE TINDAKAN PERBAIKAN
		// =================================================================

		// Hapus data SCAT lama (Hard Delete)
		if err := tx.Unscoped().Where("incident_report_id = ?", id).Delete(&models.CorrectiveActionIncident{}).Error; err != nil {
			return err
		}

		for i := range corrective_action_incidents {
			corrective_action_incidents[i].IncidentReportID = id
		}

		// Insert data baru
		if len(corrective_action_incidents) > 0 {
			if err := tx.Create(&corrective_action_incidents).Error; err != nil {
				return err
			}
		}

		// =================================================================

		// =================================================================
		// UPDATE/RE-SYNC REVIEWS
		// =================================================================

		// 1. Hapus data review lama (Hard Delete) agar tidak duplikat
		if err := tx.Unscoped().Where("incident_report_id = ?", id).Delete(&models.IncidentReview{}).Error; err != nil {
			return err
		}

		// 2. Proses data baru jika ada
		if reviews != nil {
			reviews.IncidentReportID = id

			// CLEANUP POINTER (Sangat Penting!)
			// Mencegah error "Foreign Key Constraint Fails" jika frontend mengirim ID 0 untuk reviewser yang kosong
			if reviews.PMUserID != nil && *reviews.PMUserID == 0 {
				reviews.PMUserID = nil
			}
			if reviews.DeptUserID != nil && *reviews.DeptUserID == 0 {
				reviews.DeptUserID = nil
			}
			if reviews.OHSUserID != nil && *reviews.OHSUserID == 0 {
				reviews.OHSUserID = nil
			}
			if reviews.DirOpsUserID != nil && *reviews.DirOpsUserID == 0 {
				reviews.DirOpsUserID = nil
			}
			if reviews.KTTUserID != nil && *reviews.KTTUserID == 0 {
				reviews.KTTUserID = nil
			}
			fmt.Printf("DEBUG: Data yang akan di-save: %+v\n", reviews)
			// 3. Insert 1 row reviews baru ke database
			if err := tx.Create(reviews).Error; err != nil {
				return err
			}
		}
		// =================================================================

		// 4. Handle Deletion (DB Records & Queue Physical Deletion)
		if r.MultipartForm != nil {

			deletedFiles := r.MultipartForm.Value["deleted_files[]"]
			for _, idStr := range deletedFiles {
				docID, err := strconv.ParseUint(idStr, 10, 64)
				if err != nil {
					continue
				}

				var doc models.Documentation
				// Ambil data dulu (menggunakan Unscoped jika data sudah terlanjur soft-deleted)
				if err := tx.Unscoped().First(&doc, docID).Error; err == nil {

					// --- PERBAIKAN: Tambahkan Unscoped() di sini ---

					// 1. Hapus dari Tabel Pivot Incident (Hard Delete)
					tx.Debug().Unscoped().Where("documentation_id = ? AND incident_report_id = ?", docID, id).Delete(&models.IncidentDocumentation{})

					// 2. Hapus dari Tabel Pivot Hazard (Hard Delete)
					tx.Debug().Unscoped().Where("documentation_id = ?", docID).Delete(&models.HazardDocumentation{})

					// 3. Hapus data utama (Hard Delete)
					result := tx.Debug().Unscoped().Delete(&models.Documentation{}, docID)

					if result.Error != nil {
						fmt.Printf("Gagal hapus dokumentasi: %v\n", result.Error)
					} else if result.RowsAffected > 0 {
						// Hanya masukkan ke queue hapus fisik jika benar-benar terhapus
						filesToDeletePhysical = append(filesToDeletePhysical, "."+doc.FileURL)
					}
				} else {
					fmt.Printf("Dokumen ID %d tidak ditemukan\n", docID)
				}
			}
		}

		// 5. Handle New File Uploads
		newDocs, err := s.uploadFiles(r, "dokumentasi[]")
		if err != nil {
			return err
		}

		for _, doc := range newDocs {
			if err := tx.Create(&doc).Error; err != nil {
				return err
			}
			pivot := models.IncidentDocumentation{
				IncidentReportID: id,
				DocumentationID:  doc.ID,
				DocType:          "incident",
			}
			if err := tx.Create(&pivot).Error; err != nil {
				return err
			}
		}

		// 6. Audit Log
		audit := models.IncidentReportedAudit{
			IncidentReportID: id,
			Action:           "UPDATE",
			ChangedBy:        userID,
			ChangedAt:        time.Now(),
		}
		return tx.Create(&audit).Error
	})

	// Jika transaksi gagal, return error
	if err != nil {
		return false, err
	}

	// 7. Hapus file fisik secara asinkron (HANYA jika transaksi sukses)
	if len(filesToDeletePhysical) > 0 {
		go func(files []string) {
			for _, path := range files {
				os.Remove(path)
			}
		}(filesToDeletePhysical)
	}

	return partiesUpdated, nil
}

// Pastikan method ini ada di services/incident_service.go

func (s *incidentService) toFileDTOIncident(docs []models.Documentation) []FileDTOIncident {
	var result []FileDTOIncident
	for _, doc := range docs {
		if doc.FileURL == "" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(doc.FileURL))
		isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" || ext == ".gif"

		result = append(result, FileDTOIncident{
			ID:      doc.ID,
			Name:    filepath.Base(doc.FileURL),
			URL:     doc.FileURL,
			IsImage: isImage,
		})
	}
	return result
}

func (s *incidentService) uploadFiles(r *http.Request, fieldName string) ([]models.Documentation, error) {
	// 1. Pastikan MultipartForm sudah di-parse dan tersedia
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, fmt.Errorf("multipart form tidak tersedia atau belum di-parse")
	}

	files, ok := r.MultipartForm.File[fieldName]
	if !ok || len(files) == 0 {
		return nil, nil // Tidak ada file yang diunggah pada field ini, return aman
	}

	// 2. Tentukan target folder untuk insiden
	folder := "./public/uploads/incidents"
	if err := os.MkdirAll(folder, 0755); err != nil {
		return nil, fmt.Errorf("gagal membuat direktori penyimpanan: %v", err)
	}

	var results []models.Documentation

	// 3. Iterasi file dengan index 'i' untuk mencegah bentrok nama file
	for i, header := range files {
		// Gunakan closure agar resource file langsung di-close di setiap akhir iterasi
		err := func() error {
			// Cek ukuran file (Batas maksimal 2MB)
			if header.Size > 2<<20 {
				return fmt.Errorf("ukuran file %s melebihi batas maksimal 2MB", header.Filename)
			}

			// Buka file sumber
			file, err := header.Open()
			if err != nil {
				return fmt.Errorf("gagal membuka file %s: %v", header.Filename, err)
			}
			defer file.Close()

			// Buat nama file unik (Timestamp Nanosecond + Index + Ekstensi)
			ext := filepath.Ext(header.Filename)
			safeName := fmt.Sprintf("%d_%d%s", time.Now().UnixNano(), i, ext)
			path := filepath.Join(folder, safeName)

			// Buat file tujuan di server
			dst, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("gagal membuat file tujuan %s: %v", header.Filename, err)
			}
			defer dst.Close()

			// Salin konten file
			if _, err := io.Copy(dst, file); err != nil {
				return fmt.Errorf("gagal menyalin konten file %s: %v", header.Filename, err)
			}

			// 4. Bungkus ke dalam objek models.Documentation sesuai kebutuhan transaksi database Anda
			doc := models.Documentation{
				FileURL:  "/public/uploads/incidents/" + safeName,
				FileName: header.Filename,
				FileType: ext,
				FileSize: header.Size,
			}

			results = append(results, doc)
			return nil
		}()

		// Jika salah satu file gagal diproses, batalkan seluruh upload demi konsistensi data
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func deletePhysicalFileIncident(fileURL string) {
	if fileURL == "" {
		return
	}

	// Konversi dari URL database "/public/uploads/hazards/xxx.png"
	// Menjadi file path OS "./public/uploads/hazards/xxx.png"
	localPath := filepath.Clean("." + fileURL)

	err := os.Remove(localPath)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Gagal menghapus file fisik %s: %v", localPath, err)
	}
}

func (s *incidentService) sendUpdateNotification(h models.Hazard, isUpdate bool) {
	// 1. Defer recover untuk mencegah aplikasi crash
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic email:", r)
		}
	}()

	// ==========================================================
	// RE-FETCH DATA (Mendapatkan relasi lengkap)
	// ==========================================================
	var fullHazard models.Hazard
	err := s.db.
		Preload("EventCategory").
		Preload("RiskMatrix.RiskAssessment").
		Preload("ReportBy").
		Preload("Location").
		First(&fullHazard, h.ID).Error

	if err != nil {
		fmt.Println("Gagal memuat data lengkap untuk email:", err)
		return
	}

	// 2. Ambil Email
	picEmail, err := helpers.GetPICEmail(s.db, fullHazard.PicID)
	if err != nil {
		fmt.Println("gagal ambil pic email:", err)
	}

	moderatorEmails, err := helpers.GetModeratorEmails(s.db, fullHazard.EventCategoryID)
	if err != nil {
		fmt.Println("gagal ambil moderator email:", err)
	}

	// 3. Persiapkan Data
	namaPelapor := "Sistem / Anonim"
	if fullHazard.ReportBy != nil {
		namaPelapor = fullHazard.ReportBy.Name
	}

	locationName := "-"
	if fullHazard.Location.Name != "" {
		locationName = fullHazard.Location.Name
	}

	categoryName := "-"
	if fullHazard.EventCategory.Name != "" {
		categoryName = fullHazard.EventCategory.Name
	}

	riskLevel := "-"
	if fullHazard.RiskMatrix.RiskAssessment.Name != "" {
		riskLevel = fullHazard.RiskMatrix.RiskAssessment.Name
	}

	hazardURL := fmt.Sprintf("%s/hazard/edit/%d", config.AppURL(), fullHazard.ID)

	// 4. Generate HTML Table
	tableContent := fmt.Sprintf(`
        <table border="1" cellpadding="8" cellspacing="0" style="border-collapse: collapse; width: 100%%;">
            <tr><td><strong>Kategori</strong></td><td>%s</td></tr>
            <tr><td><strong>Lokasi</strong></td><td>%s</td></tr>
            <tr><td><strong>Deskripsi</strong></td><td>%s</td></tr>
            <tr><td><strong>Risk</strong></td><td>%s</td></tr>
            <tr><td><strong>Pelapor</strong></td><td>%s</td></tr>
            <tr><td><strong>Status</strong></td><td>%s</td></tr>
        </table>
        <p style="margin-top:20px;">
            <a href="%s" style="background:#2563eb; color:white; padding:12px 20px; text-decoration:none; border-radius:6px; display:inline-block; font-weight:bold;">
                Buka Laporan Hazard
            </a>
        </p>`,
		categoryName, locationName, fullHazard.Deskripsi, riskLevel, namaPelapor, fullHazard.Status, hazardURL,
	)

	// 5. Logic Greeting Dinamis
	moderatorGreeting := "Halo Moderator HSE, Terdapat hazard baru pada kategori yang Anda moderasi:"
	if isUpdate {
		moderatorGreeting = "Halo Moderator HSE, Terdapat update hazard pada kategori yang Anda moderasi:"
	}

	picGreeting := "Halo PIC, Anda ditunjuk sebagai PIC hazard baru berikut:"
	if isUpdate {
		picGreeting = "Halo PIC, Terdapat update pada hazard yang Anda tangani:"
	}

	// 6. Fungsi Helper untuk Kirim Email (IMPLEMENTASI LENGKAP)
	sendMail := func(to []string, subject, greeting string) {
		if len(to) == 0 || (len(to) == 1 && to[0] == "") {
			return
		}

		// tableContent digunakan di sini, sehingga error 'declared and not used' hilang
		fullHTML := fmt.Sprintf("<p>%s</p> %s", greeting, tableContent)

		err := config.SendEmail(
			to,
			subject,
			config.EmailTemplate("SENTRY Hazard System", "Notifikasi Hazard", fullHTML),
		)

		if err != nil {
			fmt.Printf("gagal kirim email ke %v: %v\n", to, err)
		}
	}

	// 7. Eksekusi Pengiriman
	if picEmail != "" {
		sendMail([]string{picEmail}, fmt.Sprintf("SENTRY PIC Hazard #%d", fullHazard.ID), picGreeting)
	}

	if len(moderatorEmails) > 0 {
		sendMail(moderatorEmails, fmt.Sprintf("SENTRY Moderator Hazard #%d", fullHazard.ID), moderatorGreeting)
	}
}
