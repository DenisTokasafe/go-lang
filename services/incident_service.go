package services

import (
	"latihan1/models"

	"gorm.io/gorm"
)

type IncidentService interface {
	GetFormDataReferences(page int) (map[string]interface{}, error)
	CreateIncident(incident *models.IncidentReport) error
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
func (s *incidentService) CreateIncident(incident *models.IncidentReport) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&incident).Error; err != nil {
			return err
		}
		return nil
	})
}
