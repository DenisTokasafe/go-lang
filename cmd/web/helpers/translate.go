package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// TranslateText menerjemahkan teks ke bahasa target (misal: "en", "id")
// Menggunakan endpoint publik Google (Gratis & Tanpa API Key)
func TranslateText(text string, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	// 1. LOG AWAL: Memantau data yang masuk ke fungsi
	fmt.Printf("\n================ [TRANSLATE START] ================\n")
	fmt.Printf("Target Bahasa : %s\n", targetLang)
	fmt.Printf("Teks Asli     : %s\n", text)
	fmt.Printf("---------------------------------------------------\n")

	// Encode teks agar aman dikirim melalui URL Query
	encodedText := url.QueryEscape(text)
	apiURL := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=%s&dt=t&q=%s", targetLang, encodedText)

	// Kirim request ke Google Translate
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Printf("[ERROR] Gagal koneksi ke Google: %v\n", err)
		fmt.Printf("================= [TRANSLATE END] =================\n\n")
		return "", fmt.Errorf("gagal menghubungi API translate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[ERROR] Google merespon dengan HTTP Status: %d\n", resp.StatusCode)
		fmt.Printf("================= [TRANSLATE END] =================\n\n")
		return "", fmt.Errorf("API translate mengembalikan status non-200: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gagal membaca body respon: %w", err)
	}

	// Format kembalian Google Translate berupa array bersarang: [[[ "hasil", "asli", ... ]]]
	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("[ERROR] Gagal parsing JSON dari Google\n")
		fmt.Printf("================= [TRANSLATE END] =================\n\n")
		return "", fmt.Errorf("gagal unmarshal json respon: %w", err)
	}

	// Ekstrak teks hasil terjemahan dari array bersarang
	if len(result) > 0 {
		if innerResult, ok := result[0].([]interface{}); ok {
			translatedText := ""
			for _, item := range innerResult {
				if pair, ok := item.([]interface{}); ok && len(pair) > 0 {
					if str, ok := pair[0].(string); ok {
						translatedText += str
					}
				}
			}
			if translatedText != "" {
				// 2. LOG SUKSES: Memantau hasil akhir terjemahan
				fmt.Printf("Hasil Translate: %s\n", translatedText)
				fmt.Printf("================= [TRANSLATE END] =================\n\n")
				return translatedText, nil
			}
		}
	}

	fmt.Printf("[ERROR] Format array JSON Google berubah / tidak dikenali\n")
	fmt.Printf("================= [TRANSLATE END] =================\n\n")
	return "", fmt.Errorf("format respon translate tidak dikenali")
}
