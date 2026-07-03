package config

import (
	"fmt"
	"latihan1/models"
	"latihan1/seeders"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {

	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal menghubungkan ke database:", err)
	}

	db.AutoMigrate(
		&models.Company{},
		&models.Role{},
		&models.Contractor{},
		&models.Department{},
		&models.BusinessUnit{},
		&models.Custodian{},
		&models.Group{},
		&models.DepartmentGroup{},
		&models.Manhours{},
		&models.Location{},
		&models.RiskLikelihood{},
		&models.RiskConsequence{},
		&models.RiskAssessmentCode{},
		&models.RiskMatrix{},
		&models.ScatOption{},
		&models.BodyPart{},
		&models.EventCategory{},
		&models.User{},
		&models.Hazard{},
		&models.Documentation{},
		&models.HazardDocumentation{},
		&models.HazardAudit{},
		&models.UserEventCategory{},
		&models.CorrectiveActionHazard{},
		&models.IncidentReport{},
		&models.InvolvedParty{},
		&models.IncidentDocumentation{}, // Pastikan tabel pivot ini ikut dimigrasi
		&models.IncidentReportedAudit{},
		&models.InvestigationParticipant{},
		&models.PeepoFactor{},
		&models.Timeline{},
		&models.IncidentCause{},
		&models.CorrectiveActionIncident{},
		&models.IncidentReview{},
	)
	if err != nil {
		log.Fatalf("MIGRASI DATABASE GAGAL: %v", err)
	} else {
		fmt.Println("Migrasi sukses!")
	}

	fmt.Println("Database terkoneksi & Migrasi berhasil!")

	seeders.SeedScatOptions(db)
	seeders.SeedAdminUser(db)

	return db
}
