package bootstrap

import (
	"latihan1/cmd/web/helpers"
	"latihan1/controllers"
	"latihan1/services"
	"net/http"

	"gorm.io/gorm"
)

type Controllers struct {
	AuthController            *controllers.AuthController
	CompanyController         *controllers.CompanyController
	RoleController            *controllers.RoleController
	ContractorController      *controllers.ContractorController
	DepartmentController      *controllers.DepartmentController
	BusinessUnitController    *controllers.BusinessUnitController
	CustodianController       *controllers.CustodianController
	GroupController           *controllers.GroupController
	DepartmentGroupController *controllers.DepartmentGroupController
	ManhoursController        *controllers.ManhoursController
	LocationController        *controllers.LocationController
	RiskLikelihoodController  *controllers.RiskLikelihoodController
	RiskConsequenceController *controllers.RiskConsequenceController
	RiskAssessmentController  *controllers.RiskAssessmentController
	RiskMatrixController      *controllers.RiskMatrixController
	ScatOptionController      *controllers.ScatOptionController
	BodyPartController        *controllers.BodyPartController
	EventCategoryController   *controllers.EventCategoryController
	UserController            *controllers.UserController
	HazardController          *controllers.HazardController
	IncidentController        *controllers.IncidentController
	DashboardController       *controllers.DashboardController
}

func InitControllers(db *gorm.DB) *Controllers {

	render := func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{}) {
		helpers.RenderTemplate(db, w, r, tmpl, data)
	}

	return &Controllers{

		// =========================
		// AUTH
		// =========================
		AuthController: &controllers.AuthController{
			DB:     db,
			Render: render,
		},

		// =========================
		// COMPANY
		// =========================
		CompanyController: &controllers.CompanyController{
			DB:     db,
			Render: render,
		},

		// =========================
		// ROLE
		// =========================
		RoleController: &controllers.RoleController{
			DB:     db,
			Render: render,
		},

		// =========================
		// CONTRACTOR
		// =========================
		ContractorController: &controllers.ContractorController{
			DB:     db,
			Render: render,
		},

		// =========================
		// DEPARTMENT
		// =========================
		DepartmentController: &controllers.DepartmentController{
			DB:     db,
			Render: render,
		},

		// =========================
		// BUSINESS UNIT
		// =========================
		BusinessUnitController: &controllers.BusinessUnitController{
			DB:     db,
			Render: render,
		},

		// =========================
		// CUSTODIAN
		// =========================
		CustodianController: &controllers.CustodianController{
			DB:     db,
			Render: render,
		},

		// =========================
		// GROUP
		// =========================
		GroupController: &controllers.GroupController{
			DB:     db,
			Render: render,
		},

		// =========================
		// DEPARTMENT GROUP
		// =========================
		DepartmentGroupController: &controllers.DepartmentGroupController{
			DB:     db,
			Render: render,
		},

		// =========================
		// MANHOURS
		// =========================
		ManhoursController: &controllers.ManhoursController{
			DB:     db,
			Render: render,
		},

		// =========================
		// LOCATION
		// =========================
		LocationController: &controllers.LocationController{
			DB:     db,
			Render: render,
		},

		// =========================
		// RISK LIKELIHOOD
		// =========================
		RiskLikelihoodController: &controllers.RiskLikelihoodController{
			DB:     db,
			Render: render,
		},

		// =========================
		// RISK CONSEQUENCE
		// =========================
		RiskConsequenceController: &controllers.RiskConsequenceController{
			DB:     db,
			Render: render,
		},

		// =========================
		// RISK ASSESSMENT
		// =========================
		RiskAssessmentController: &controllers.RiskAssessmentController{
			DB:     db,
			Render: render,
		},

		// =========================
		// RISK MATRIX
		// =========================
		RiskMatrixController: &controllers.RiskMatrixController{
			DB:     db,
			Render: render,
		},

		// =========================
		// SCAT OPTION
		// =========================
		ScatOptionController: &controllers.ScatOptionController{
			DB:     db,
			Render: render,
		},

		// =========================
		// BODY PART
		// =========================
		BodyPartController: &controllers.BodyPartController{
			DB:     db,
			Render: render,
		},

		// =========================
		// EVENT CATEGORY
		// =========================
		EventCategoryController: &controllers.EventCategoryController{
			DB:     db,
			Render: render,
		},

		// =========================
		// USER
		// =========================
		UserController: &controllers.UserController{
			DB:     db,
			Render: render,
		},

		// =========================
		// HAZARD
		// =========================
		HazardController: &controllers.HazardController{
			DB: db,
			Service: &services.HazardService{
				DB: db,
			},
			Render: render,
		},

		// =========================
		// INCIDENT
		// =========================
		IncidentController: &controllers.IncidentController{
			DB: db,
			// Gunakan function constructor NewIncidentService agar interface-nya terikat dengan benar
			ServiceIncident: services.NewIncidentService(db),
			Render:          render,
		},

		// =========================
		// HAZARD DASHBOARD
		// =========================
		DashboardController: &controllers.DashboardController{
			DB: db,
			Service: &services.DashboardService{
				DB: db,
			},
			Render: render,
		},
	}
}
