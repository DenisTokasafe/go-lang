package models

import (
	"time"
)

// IncidentReview menyimpan data Part 9 - Komentar & Penerimaan
type IncidentReview struct {
	ID               uint            `gorm:"primarykey" json:"id"`
	IncidentReportID uint            `gorm:"index" json:"incident_report_id"`
	IncidentReport   *IncidentReport `gorm:"foreignKey:IncidentReportID;constraint:OnDelete:CASCADE;" json:"-"`

	// 1. Project Manager (Jika Kontraktor)
	PMComment  string `gorm:"type:text" json:"pm_comment"`
	PMUserID   *uint  `json:"pm_user_id"`
	PMUser     *User  `gorm:"foreignKey:PMUserID" json:"-"`
	PMAccepted *bool  `json:"pm_accepted"`

	// 2. Dept Penanggung Jawab
	DeptComment  string `gorm:"type:text" json:"dept_comment"`
	DeptUserID   *uint  `json:"dept_user_id"`
	DeptUser     *User  `gorm:"foreignKey:DeptUserID" json:"-"`
	DeptAccepted *bool  `json:"dept_accepted"`

	// 3. OHS Dept Head
	OHSComment  string `gorm:"type:text" json:"ohs_comment"`
	OHSUserID   *uint  `json:"ohs_user_id"`
	OHSUser     *User  `gorm:"foreignKey:OHSUserID" json:"-"`
	OHSAccepted *bool  `json:"ohs_accepted"`

	// 4. Direktur Operasi (Hanya untuk Level 3, 4, 5)
	DirOpsComment  string `gorm:"type:text" json:"dirops_comment"`
	DirOpsUserID   *uint  `json:"dirops_user_id"`
	DirOpsUser     *User  `gorm:"foreignKey:DirOpsUserID" json:"-"`
	DirOpsAccepted *bool  `json:"dirops_accepted"`

	// 5. KTT (Hanya untuk Level 3, 4, 5)
	KTTComment  string `gorm:"type:text" json:"ktt_comment"`
	KTTUserID   *uint  `json:"ktt_user_id"`
	KTTUser     *User  `gorm:"foreignKey:KTTUserID" json:"-"`
	KTTAccepted *bool  `json:"ktt_accepted"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
