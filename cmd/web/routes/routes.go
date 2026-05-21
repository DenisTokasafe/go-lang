package routes

import (
	"latihan1/cmd/web/bootstrap"
	"latihan1/cmd/web/helpers"
	"latihan1/middlewares"
	"net/http"

	"gorm.io/gorm"
)

func RegisterRoutes(db *gorm.DB, c *bootstrap.Controllers) {

	// =========================
	// MIDDLEWARE HELPER
	// =========================
	withAuth := func(handler http.HandlerFunc) http.HandlerFunc {
		return middlewares.AuthMiddleware(db, handler)
	}

	isAdmin := func(handler http.HandlerFunc) http.HandlerFunc {
		return middlewares.AuthMiddleware(
			db,
			middlewares.RoleMiddleware("Administrator", handler),
		)
	}

	// =========================
	// STATIC FILE
	// =========================
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	distFs := http.FileServer(http.Dir("public/dist"))
	http.Handle("/dist/", http.StripPrefix("/dist/", distFs))

	uploadsFs := http.FileServer(http.Dir("uploads"))
	http.Handle("/uploads/", http.StripPrefix("/uploads/", uploadsFs))

	// =========================
	// AUTH
	// =========================
	http.HandleFunc("/login", c.AuthController.ShowLogin)
	http.HandleFunc("/login/process", c.AuthController.Login)
	http.HandleFunc("/register", c.AuthController.ShowRegister)
	http.HandleFunc("/register/process", c.AuthController.Register)
	http.HandleFunc("/logout", c.AuthController.Logout)

	// =========================
	// DASHBOARD
	// =========================
	http.HandleFunc("/", withAuth(func(w http.ResponseWriter, r *http.Request) {
		helpers.RenderTemplate(db, w, r, "/dashboard.html", nil)
	}))

	// =========================
	// COMPANY
	// =========================
	http.HandleFunc("/administration/company", isAdmin(c.CompanyController.Index))
	http.HandleFunc("/administration/company/store", isAdmin(c.CompanyController.Store))
	http.HandleFunc("/administration/company/update", isAdmin(c.CompanyController.Update))
	http.HandleFunc("/administration/company/delete", isAdmin(c.CompanyController.Delete))
	http.HandleFunc("/administration/company/upload", isAdmin(c.CompanyController.UploadExcel))

	// =========================
	// ROLE
	// =========================
	http.HandleFunc("/administration/role", isAdmin(c.RoleController.Index))
	http.HandleFunc("/administration/role/store", isAdmin(c.RoleController.Store))
	http.HandleFunc("/administration/role/update", isAdmin(c.RoleController.Update))
	http.HandleFunc("/administration/role/delete", isAdmin(c.RoleController.Delete))
	http.HandleFunc("/administration/role/upload", isAdmin(c.RoleController.UploadExcel))

	// =========================
	// CONTRACTOR
	// =========================
	http.HandleFunc("/administration/contractor", isAdmin(c.ContractorController.Index))
	http.HandleFunc("/administration/contractor/store", isAdmin(c.ContractorController.Store))
	http.HandleFunc("/administration/contractor/update", isAdmin(c.ContractorController.Update))
	http.HandleFunc("/administration/contractor/delete", isAdmin(c.ContractorController.Delete))
	http.HandleFunc("/administration/contractor/upload", isAdmin(c.ContractorController.UploadExcel))

	// =========================
	// DEPARTMENT
	// =========================
	http.HandleFunc("/administration/department", isAdmin(c.DepartmentController.Index))
	http.HandleFunc("/administration/department/store", isAdmin(c.DepartmentController.Store))
	http.HandleFunc("/administration/department/update", isAdmin(c.DepartmentController.Update))
	http.HandleFunc("/administration/department/delete", isAdmin(c.DepartmentController.Delete))
	http.HandleFunc("/administration/department/upload", isAdmin(c.DepartmentController.UploadExcel))

	// =========================
	// BUSINESS UNIT
	// =========================
	http.HandleFunc("/administration/business-unit", isAdmin(c.BusinessUnitController.Index))
	http.HandleFunc("/administration/business-unit/store", isAdmin(c.BusinessUnitController.Store))
	http.HandleFunc("/administration/business-unit/update", isAdmin(c.BusinessUnitController.Update))
	http.HandleFunc("/administration/business-unit/delete", isAdmin(c.BusinessUnitController.Delete))
	http.HandleFunc("/administration/business-unit/upload", isAdmin(c.BusinessUnitController.UploadExcel))

	// =========================
	// CUSTODIAN
	// =========================
	http.HandleFunc("/administration/custodian", isAdmin(c.CustodianController.Index))
	http.HandleFunc("/administration/custodian/store", isAdmin(c.CustodianController.Store))
	http.HandleFunc("/administration/custodian/update", isAdmin(c.CustodianController.Update))
	http.HandleFunc("/administration/custodian/delete", isAdmin(c.CustodianController.Delete))
	http.HandleFunc("/administration/custodian/upload", isAdmin(c.CustodianController.UploadExcel))

	// =========================
	// GROUP
	// =========================
	http.HandleFunc("/administration/department-group/group", isAdmin(c.GroupController.Index))
	http.HandleFunc("/administration/department-group/group/store", isAdmin(c.GroupController.Store))
	http.HandleFunc("/administration/department-group/group/update", isAdmin(c.GroupController.Update))
	http.HandleFunc("/administration/department-group/group/delete", isAdmin(c.GroupController.Delete))
	http.HandleFunc("/administration/department-group/group/upload", isAdmin(c.GroupController.UploadExcel))

	// =========================
	// LOCATION
	// =========================
	http.HandleFunc("/administration/location", isAdmin(c.LocationController.Index))
	http.HandleFunc("/administration/location/store", isAdmin(c.LocationController.Store))
	http.HandleFunc("/administration/location/update", isAdmin(c.LocationController.Update))
	http.HandleFunc("/administration/location/delete", isAdmin(c.LocationController.Delete))
	http.HandleFunc("/administration/location/upload", isAdmin(c.LocationController.UploadExcel))

	// =========================
	// EVENT CATEGORY
	// =========================
	http.HandleFunc("/administration/event-category", isAdmin(c.EventCategoryController.Index))
	http.HandleFunc("/administration/event-category/store", isAdmin(c.EventCategoryController.Store))
	http.HandleFunc("/administration/event-category/update", isAdmin(c.EventCategoryController.Update))
	http.HandleFunc("/administration/event-category/delete", isAdmin(c.EventCategoryController.Delete))

	// =========================
	// BODY PART
	// =========================
	http.HandleFunc("/administration/body-part", isAdmin(c.BodyPartController.Index))
	http.HandleFunc("/administration/body-part/store", isAdmin(c.BodyPartController.Store))
	http.HandleFunc("/administration/body-part/update", isAdmin(c.BodyPartController.Update))
	http.HandleFunc("/administration/body-part/delete", isAdmin(c.BodyPartController.Delete))
	http.HandleFunc("/administration/body-part/upload", isAdmin(c.BodyPartController.UploadExcel))

	// =========================
	// SCAT OPTION
	// =========================
	http.HandleFunc("/administration/scat-option", isAdmin(c.ScatOptionController.Index))
	http.HandleFunc("/administration/scat-option/store", isAdmin(c.ScatOptionController.Store))
	http.HandleFunc("/administration/scat-option/update", isAdmin(c.ScatOptionController.Update))
	http.HandleFunc("/administration/scat-option/delete", isAdmin(c.ScatOptionController.Delete))
	http.HandleFunc("/administration/scat-option/upload", isAdmin(c.ScatOptionController.UploadExcel))

	// =========================
	// RISK LIKELIHOOD
	// =========================
	http.HandleFunc("/administration/risk/likelihood", isAdmin(c.RiskLikelihoodController.Index))
	http.HandleFunc("/administration/risk/likelihood/store", isAdmin(c.RiskLikelihoodController.Store))
	http.HandleFunc("/administration/risk/likelihood/update", isAdmin(c.RiskLikelihoodController.Update))
	http.HandleFunc("/administration/risk/likelihood/delete", isAdmin(c.RiskLikelihoodController.Delete))
	http.HandleFunc("/administration/risk/likelihood/upload", isAdmin(c.RiskLikelihoodController.UploadExcel))

	// =========================
	// RISK CONSEQUENCE
	// =========================
	http.HandleFunc("/administration/risk/consequence", isAdmin(c.RiskConsequenceController.Index))
	http.HandleFunc("/administration/risk/consequence/store", isAdmin(c.RiskConsequenceController.Store))
	http.HandleFunc("/administration/risk/consequence/update", isAdmin(c.RiskConsequenceController.Update))
	http.HandleFunc("/administration/risk/consequence/delete", isAdmin(c.RiskConsequenceController.Delete))
	http.HandleFunc("/administration/risk/consequence/upload", isAdmin(c.RiskConsequenceController.UploadExcel))

	// =========================
	// RISK ASSESSMENT
	// =========================
	http.HandleFunc("/administration/risk/assessment", isAdmin(c.RiskAssessmentController.Index))
	http.HandleFunc("/administration/risk/assessment/store", isAdmin(c.RiskAssessmentController.Store))
	http.HandleFunc("/administration/risk/assessment/update", isAdmin(c.RiskAssessmentController.Update))
	http.HandleFunc("/administration/risk/assessment/delete", isAdmin(c.RiskAssessmentController.Delete))
	http.HandleFunc("/administration/risk/assessment/upload", isAdmin(c.RiskAssessmentController.UploadExcel))

	// =========================
	// RISK MATRIX
	// =========================
	http.HandleFunc("/administration/risk/matrix", isAdmin(c.RiskMatrixController.Index))
	http.HandleFunc("/administration/risk/matrix/store", isAdmin(c.RiskMatrixController.Store))
	http.HandleFunc("/administration/risk/matrix/update", isAdmin(c.RiskMatrixController.Update))
	http.HandleFunc("/administration/risk/matrix/delete", isAdmin(c.RiskMatrixController.Delete))
	http.HandleFunc("/administration/risk/matrix/upload", isAdmin(c.RiskMatrixController.UploadExcel))

	// =========================
	// DEPARTMENT GROUP
	// =========================
	http.HandleFunc("/administration/department-group", isAdmin(c.DepartmentGroupController.Index))
	http.HandleFunc("/administration/department-group/store", isAdmin(c.DepartmentGroupController.Store))
	http.HandleFunc("/administration/department-group/update", isAdmin(c.DepartmentGroupController.Update))
	http.HandleFunc("/administration/department-group/delete", isAdmin(c.DepartmentGroupController.Delete))
	http.HandleFunc("/administration/department-group/upload", isAdmin(c.DepartmentGroupController.UploadExcel))
	http.HandleFunc("/department-groups/export", isAdmin(c.DepartmentGroupController.ExportExcel))

	// =========================
	// MANHOURS
	// =========================
	http.HandleFunc("/manhours", withAuth(c.ManhoursController.Index))
	http.HandleFunc("/manhours/store", withAuth(c.ManhoursController.Store))
	http.HandleFunc("/manhours/update", withAuth(c.ManhoursController.Update))
	http.HandleFunc("/manhours/delete", withAuth(c.ManhoursController.Delete))
	http.HandleFunc("/manhours/upload", withAuth(c.ManhoursController.UploadExcel))

	// =========================
	// PEOPLE
	// =========================
	http.HandleFunc("/people", withAuth(c.UserController.Index))
	http.HandleFunc("/people/store", withAuth(c.UserController.Store))
	http.HandleFunc("/people/update", withAuth(c.UserController.Update))
	http.HandleFunc("/people/delete", withAuth(c.UserController.Delete))
	http.HandleFunc("/people/upload", withAuth(c.UserController.UploadExcel))

	// =========================
	// HAZARD
	// =========================
	http.HandleFunc("/hazards/sync", withAuth(c.HazardController.SyncField))
	http.HandleFunc("/hazard", withAuth(c.HazardController.Index))
	http.HandleFunc("/hazard/create", withAuth(c.HazardController.Create))
	http.HandleFunc("/hazard/store", withAuth(c.HazardController.Store))
	http.HandleFunc("GET /hazard/edit/{id}", withAuth(c.HazardController.Edit))
	http.HandleFunc("POST /hazard/update/{id}", withAuth(c.HazardController.Update))
}
