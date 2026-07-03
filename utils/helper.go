package utils // Sesuaikan nama package-nya

import (
	"strconv"
)

// ParseUintPointer konversi string ke *uint
func ParseUintPointer(val string) *uint {
	if val == "" || val == "0" {
		return nil
	}
	u64, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return nil
	}
	u := uint(u64)
	return &u
}

// ParseBoolPointer konversi string ke *bool
func ParseBoolPointer(val string) *bool {
	if val == "" {
		return nil
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return nil
	}
	return &b
}
