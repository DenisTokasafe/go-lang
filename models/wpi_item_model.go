package models

import (
	"time"

	"gorm.io/gorm"
)

type OhsRiskCode string

const (
	RiskCodeExtreme  OhsRiskCode = "E"
	RiskCodeHigh     OhsRiskCode = "H"
	RiskCodeModerate OhsRiskCode = "M"
	RiskCodeLow      OhsRiskCode = "L"
)

type WpiItem struct {
	gorm.Model
	WpiReportID uint `gorm:"not null" json:"wpi_report_id"`

	OhsRiskCode      OhsRiskCode `gorm:"type:varchar(5)" json:"ohs_risk_code"`
	UnsafeCondition  string      `gorm:"type:text" json:"unsafe_condition"`
	PreventionAction string      `gorm:"type:text" json:"prevention_action"`
	// JSON array URL dokumen. Pointer menjaga nilai kolom tetap NULL bila belum ada dokumen.
	// Diabaikan saat unmarshal items_json (frontend mengirim array URL), lalu diisi
	// oleh service dari multipart upload agar tetap tersimpan sebagai JSON string DB.
	DocUraianTindakan *string `gorm:"type:text" json:"-"`
	DocJenisTindakan  *string `gorm:"type:text" json:"-"`

	PicID *uint `gorm:"column:pic_id" json:"-"`
	PIC   *User `gorm:"foreignKey:PicID"`

	// Field ini hanya untuk menerima JSON dari frontend (Alpine.js)
	PicIDInput          UintString `gorm:"-" json:"pic_id"`
	DueDateInput        DateString `gorm:"-" json:"due_date"`
	CompletionDateInput DateString `gorm:"-" json:"completion_date"`

	DueDate        *time.Time `gorm:"type:date" json:"-"`
	PrWoNeeded     string     `gorm:"type:varchar(100)" json:"pr_wo_needed"`
	CompletionDate *time.Time `gorm:"type:date" json:"-"`
	InxNumber      string     `gorm:"type:varchar(100)" json:"inx_number"`
}
