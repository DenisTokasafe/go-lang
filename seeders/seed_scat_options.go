package seeders

import (
	"latihan1/models"
	"log"

	"gorm.io/gorm"
)

func SeedScatOptions(db *gorm.DB) {
	options := []models.ScatOption{
		// --- 1. UNSAFE ACTS ---
		{Code: "1.2.1", Type: "unsafe_act", Name: "Mengoperasikan peralatan tanpa izin"},
		{Code: "1.2.2", Type: "unsafe_act", Name: "Gagal / lalai memperingatkan"},
		{Code: "1.2.3", Type: "unsafe_act", Name: "Gagal / lalai mengamankan"},
		{Code: "1.2.3", Type: "unsafe_act", Name: "Gagal / lalai mengamankan"},
		{Code: "1.2.4", Type: "unsafe_act", Name: "Mengoperasikan dengan kecepatan tidak sesuai"},
		{Code: "1.2.5", Type: "unsafe_act", Name: "Membuat alat pengaman tidak berfungsi"},
		{Code: "1.2.6", Type: "unsafe_act", Name: "Memakai alat yang rusak"},
		{Code: "1.2.7", Type: "unsafe_act", Name: "Gagal / lalai menggunakan APD yang semestinya"},
		{Code: "1.2.8", Type: "unsafe_act", Name: "Pembebanan yang tidak sesuai"},
		{Code: "1.2.9", Type: "unsafe_act", Name: "Salah meletakkan / memuat"},
		{Code: "1.2.10", Type: "unsafe_act", Name: "Pengangkatan yang tidak sesuai"},
		{Code: "1.2.11", Type: "unsafe_act", Name: "Berada di tempat / posisi yang terlarang"},
		{Code: "1.2.12", Type: "unsafe_act", Name: "Memperbaiki peralatan yang bekerja / bergerak"},
		{Code: "1.2.13", Type: "unsafe_act", Name: "Bercanda berlebihan"},
		{Code: "1.2.14", Type: "unsafe_act", Name: "Di bawah pengaruh alkohol dan/atau obat terlarang"},
		{Code: "1.2.15", Type: "unsafe_act", Name: "Memakai peralatan yang bukan semestinya"},
		{Code: "1.2.16", Type: "unsafe_act", Name: "Gagal / lalai mengikuti prosedur"},
		{Code: "1.2.17", Type: "unsafe_act", Name: "Lainnya"},
		{Code: "1.2.17", Type: "unsafe_act", Name: "Lainnya"},

		// --- 2. PERSONAL FACTORS ---
		{Code: "2.1.1", Type: "personal_factor", Name: "Tidak memadainya kemampuan fisik / fisiologis"},
		{Code: "2.1.2", Type: "personal_factor", Name: "Keterbatasan mental / Kemampuan psikologi"},
		{Code: "2.1.3", Type: "personal_factor", Name: "Tekanan Fisik atau fisiologis"},
		{Code: "2.1.4", Type: "personal_factor", Name: "Mental atau Tekanan psikologis"},
		{Code: "2.1.5", Type: "personal_factor", Name: "Kurangnya pengetahuan"},
		{Code: "2.1.6", Type: "personal_factor", Name: "Kurangnya keahlian"},
		{Code: "2.1.7", Type: "personal_factor", Name: "Salah Motivasi"},
		{Code: "2.1.8", Type: "personal_factor", Name: "Lainnya"},

		// --- 3. JOB FACTORS ---
		{Code: "2.2.1", Type: "job_factor", Name: "Kepemimpinan dan atau Fungsi pengawasan tidak memadai"},
		{Code: "2.2.2", Type: "job_factor", Name: "Engineering yang tidak memadai"},
		{Code: "2.2.3", Type: "job_factor", Name: "Pembelian yang tidak memadai"},
		{Code: "2.2.4", Type: "job_factor", Name: "Pemeliharaan yang tidak memadai"},
		{Code: "2.2.5", Type: "job_factor", Name: "Alat dan peralatan yang tidak memadai"},
		{Code: "2.2.6", Type: "job_factor", Name: "Standar-standar kerja yang tidak memadai"},
		{Code: "2.2.7", Type: "job_factor", Name: "Pemakaian yang berlebihan"},
		{Code: "2.2.8", Type: "job_factor", Name: "Salah pakai atau penyalahgunaan"},
		{Code: "2.2.9", Type: "job_factor", Name: "Lainnya"},

		// --- 4. CONTROL SYSTEM ---
		{Code: "2.3.1", Type: "control_system", Name: "Perangkat Keras"},
		{Code: "2.3.2", Type: "control_system", Name: "Pelatihan"},
		{Code: "2.3.3", Type: "control_system", Name: "Organisasi"},
		{Code: "2.3.4", Type: "control_system", Name: "Komunikasi"},
		{Code: "2.3.5", Type: "control_system", Name: "Sasaran tidak kompatibel"},
		{Code: "2.3.6", Type: "control_system", Name: "Prosedur"},
		{Code: "2.3.7", Type: "control_system", Name: "Manajemen Pemeliharaan"},
		{Code: "2.3.8", Type: "control_system", Name: "Disain"},
		{Code: "2.3.9", Type: "control_system", Name: "Manajemen Resiko"},
		{Code: "2.3.10", Type: "control_system", Name: "Manajemen Perubahan"},
		{Code: "2.3.11", Type: "control_system", Name: "Manajemen Kontraktor"},
		{Code: "2.3.12", Type: "control_system", Name: "Budaya Organisasi"},
		{Code: "2.3.13", Type: "control_system", Name: "Pengaruh Peraturan"},
		{Code: "2.3.14", Type: "control_system", Name: "Pembelajaran Organisasi"},
		{Code: "2.3.15", Type: "control_system", Name: "Manajemen Kendaraan"},
		{Code: "2.3.16", Type: "control_system", Name: "Sistem Manajemen"},
		{Code: "2.3.17", Type: "control_system", Name: "Lainnya"},
	}

	for _, opt := range options {
		// Logika: Cari berdasarkan Code & Type, jika tidak ada maka buat baru
		err := db.Where(models.ScatOption{Code: opt.Code, Type: opt.Type}).
			Attrs(models.ScatOption{Name: opt.Name}).
			FirstOrCreate(&opt).Error

		if err != nil {
			log.Printf("Gagal seed opsi %s: %v", opt.Code, err)
		}
	}
	log.Println("Seeding ScatOptions berhasil!")
}
