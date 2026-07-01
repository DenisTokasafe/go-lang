package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WhyNode adalah "blueprint" untuk struktur hirarki 5 Whys.
// Ini digunakan untuk kebutuhan Marshalling/Unmarshalling (mengubah JSON <-> Struct Go).
// TIDAK PERLU gorm.Model karena ini bukan tabel database.
type WhyNode struct {
	Text     string    `json:"text"`
	Children []WhyNode `json:"children"`
}

type Timeline struct {
	gorm.Model
	// Relasi ke IncidentReport
	IncidentReportID uint            `gorm:"index" json:"incident_report_id"`
	IncidentReport   *IncidentReport `gorm:"foreignKey:IncidentReportID;constraint:OnDelete:CASCADE;" json:"-"`
	// Deskripsi Kejadian
	Event string `gorm:"type:text" json:"event"`
	// Struktur 5 Whys (Data JSONB)
	// Kita simpan sebagai datatypes.JSON agar bisa fleksibel
	Whys datatypes.JSON `gorm:"type:jsonb" json:"whys"`
}
