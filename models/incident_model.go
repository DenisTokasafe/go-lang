// File: models/incident_report.go
package models

import (
	"time"

	"gorm.io/gorm"
)

// IncidentReport mewakili BAGIAN 1 – Detil Laporan (Parent Table)
type IncidentReport struct {
	gorm.Model
	// Nomor Referensi Unik Investigasi
	RefNumber string `gorm:"uniqueIndex;size:50;not null" json:"ref_number"`

	// Klasifikasi & Potensi Bahaya
	EventCategoryID uint          `json:"event_category_id"`
	EventCategory   EventCategory `gorm:"foreignKey:EventCategoryID"`

	RiskMatrixID uint       `json:"risk_matrix_id"`
	RiskMatrix   RiskMatrix `gorm:"foreignKey:RiskMatrixID;references:ID"`

	// Status Hazard
	Status             HazardStatus `gorm:"type:enum('submit','in_progress','pending','closed','cancelled');default:'submit'" json:"status"`
	ModeratorComment   string       `json:"moderator_comment" gorm:"type:text"`
	ModeratorCommentEn string       `json:"moderator_comment_en" gorm:"type:text"`
	// Relasi Scat Option
	ScatOptionID uint       `gorm:"column:scat_option_id;after:event_category_id"`
	ScatOption   ScatOption `gorm:"foreignKey:ScatOptionID;references:ID"`

	PotensiLTIFatality    bool   `gorm:"default:false" json:"potensi_lti_fatality"`
	KlasifikasiLingkungan string `gorm:"type:varchar(100)" json:"klasifikasi_lingkungan"` // Tidak Significant, Kecil, Sedang, Berat, Bencana

	// Waktu Kejadian
	TanggalWaktu time.Time `gorm:"not null" json:"tanggal_waktu"`

	// Relasi ke Master Data Lokasi (Konsisten dengan model Hazard)

	AreaKontrakKarya string `gorm:"type:enum('MSM','TTN','Off Site');default:'MSM'" json:"area_kontrak_karya"`

	// Relasi Penanggung Jawab (Menggunakan pointer agar NULLable jika tidak terikat ke salah satu)
	// Penanggung Jawab (Divisi/Unit)
	DepartmentID *uint       `json:"department_id"`
	Department   *Department `gorm:"foreignKey:DepartmentID"`

	ContractorID *uint       `json:"contractor_id"`
	Contractor   *Contractor `gorm:"foreignKey:ContractorID"`

	// PIC (Berdasarkan relasi User ke Dept/Cont)
	PicID uint `gorm:"not null" json:"pic_id"`
	PIC   User `gorm:"foreignKey:PicID"` // Nama Manajer Penanggung Jawab (Manual/Text)
	// Lokasi Kejadian
	LocationID       uint     `json:"location_id"`
	Location         Location `gorm:"foreignKey:LocationID"`
	LocationSpecific string   `gorm:"type:varchar(255)" json:"location_specific"`
	// Deskripsi & Detail Operasional Lapangan
	TugasDijalankan        string `gorm:"type:text" json:"tugas_dijalankan"`
	Deskripsi              string `gorm:"type:text;not null" json:"deskripsi"` // Deskripsi lengkap insiden
	TindakanLangsung       string `gorm:"type:text" json:"tindakan_langsung"`
	PekerjaanBerhenti      bool   `gorm:"default:false" json:"pekerjaan_berhenti"` // true = Berhenti, false = Lanjut
	DetilKerusakanKerugian string `gorm:"type:text" json:"detil_kerusakan_kerugian"`

	// Relasi Pelapor (User yang menginput data ke sistem)
	ReportByID *uint `json:"report_by_id"`
	ReportBy   *User `gorm:"foreignKey:ReportByID"`

	ReporterManual string `gorm:"type:varchar(255)" json:"reporter_manual"`

	// Relasi One-to-Many ke BAGIAN 2 (Pihak Terlibat)
	// constraint:OnDelete:CASCADE memastikan jika data IncidentReport dihapus, data di InvolvedParty juga otomatis ikut terhapus
	InvolvedParties []InvolvedParty         `gorm:"foreignKey:IncidentReportID;constraint:OnDelete:CASCADE" json:"pihak_terlibat"`
	Documentations  []IncidentDocumentation `gorm:"foreignKey:IncidentReportID;constraint:OnDelete:CASCADE" json:"documentations"`
	Audits          []IncidentReportedAudit `gorm:"foreignKey:IncidentReportID"`
}
