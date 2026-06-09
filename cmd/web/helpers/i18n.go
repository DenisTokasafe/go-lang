package helpers

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Translations menyimpan semua data terjemahan global di memori
// Contoh akses: Translations["id"]["AccountSettings"]
var Translations = make(map[string]map[string]string)

// LoadTranslations membaca semua file yaml di folder i18n
func LoadTranslations() error {
	languages := []string{"id", "en"}

	for _, lang := range languages {
		// Sesuaikan jalur file dengan struktur folder Anda
		filename := filepath.Join("i18n", "active."+lang+".yaml")

		fileBytes, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		var data map[string]string
		// Proses Unmarshal mengubah isi YAML menjadi map Go
		err = yaml.Unmarshal(fileBytes, &data)
		if err != nil {
			return err
		}

		// Simpan hasil parse ke map global berdasarkan bahasanya
		Translations[lang] = data
	}

	return nil
}
