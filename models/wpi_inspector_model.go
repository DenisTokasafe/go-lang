package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// UintString adalah tipe custom agar JSON dari frontend (Alpine.js)
// yang mengirim angka sebagai string (misal "5") bisa di-parse dengan benar.
type UintString struct {
	Val   uint
	Valid bool // Valid adalah false jika nilainya null/kosong
}

func (u *UintString) UnmarshalJSON(data []byte) error {
	// Coba lepas kutip (jika string "5")
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s == "" || s == "0" {
			u.Valid = false
			return nil
		}
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf("UintString: tidak bisa parse string %q: %w", s, err)
		}
		u.Val = uint(n)
		u.Valid = true
		return nil
	}
	// Coba parse langsung sebagai angka (jika number 5)
	var n uint
	if err := json.Unmarshal(data, &n); err == nil {
		u.Val = n
		u.Valid = n > 0
		return nil
	}
	// Null
	u.Valid = false
	return nil
}

func (u UintString) MarshalJSON() ([]byte, error) {
	if !u.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(u.Val)
}

// Value - agar GORM bisa menyimpan ke database sebagai *uint
func (u UintString) Value() (driver.Value, error) {
	if !u.Valid {
		return nil, nil
	}
	return int64(u.Val), nil
}

// Scan - agar GORM bisa membaca dari database
func (u *UintString) Scan(value interface{}) error {
	if value == nil {
		u.Valid = false
		return nil
	}
	switch v := value.(type) {
	case int64:
		u.Val = uint(v)
		u.Valid = true
	case uint64:
		u.Val = uint(v)
		u.Valid = true
	default:
		return fmt.Errorf("UintString: tipe tidak didukung %T", value)
	}
	return nil
}

// DateString digunakan untuk menerima JSON yang mungkin berisi string kosong "" atau format YYYY-MM-DD
type DateString struct {
	Val   time.Time
	Valid bool
}

func (d *DateString) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		d.Valid = false
		return nil
	}
	// Coba parsing format YYYY-MM-DD
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err // Jika gagal parse, akan me-return error
	}
	d.Val = t
	d.Valid = true
	return nil
}

type WpiInspector struct {
	gorm.Model
	WpiReportID uint `gorm:"not null" json:"wpi_report_id"`

	InspectorID *uint `gorm:"column:inspector_id" json:"-"`
	Inspector   *User `gorm:"foreignKey:InspectorID"`

	// Field ini hanya untuk menerima JSON dari frontend (Alpine.js)
	InspectorIDInput UintString `gorm:"-" json:"inspector_id"`

	DepartmentID *uint       `gorm:"column:department_id" json:"-"`
	Department   *Department `gorm:"foreignKey:DepartmentID"`
	DepartmentIDInput UintString `gorm:"-" json:"department_id"`

	ContractorID *uint       `gorm:"column:contractor_id" json:"-"`
	Contractor   *Contractor `gorm:"foreignKey:ContractorID"`
	ContractorIDInput UintString `gorm:"-" json:"contractor_id"`

	InspectorWorkType string `gorm:"type:varchar(20)" json:"inspector_work_type"`
}
