package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"latihan1/middlewares"
	"latihan1/models"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/kaugesaar/lucide-go"
	"gorm.io/gorm"
)

func RenderTemplate(
	db *gorm.DB,
	w http.ResponseWriter,
	r *http.Request,
	tmpl string,
	data interface{},
) {

	finalData := make(map[string]interface{})

	// =========================
	// AUTH USER
	// =========================
	if userVal := r.Context().Value(middlewares.AuthUserKey); userVal != nil {

		if user, ok := userVal.(models.User); ok {
			finalData["AuthUser"] = user
		}

	} else if val := r.Context().Value("userID"); val != nil {

		userIDStr := val.(string)

		var user models.User

		if err := db.Preload("Role").First(&user, userIDStr).Error; err == nil {
			finalData["AuthUser"] = user
		}
	}

	// =========================
	// MERGE DATA
	// =========================
	if data != nil {

		if m, ok := data.(map[string]interface{}); ok {

			for k, v := range m {
				finalData[k] = v
			}

		} else {
			finalData["Data"] = data
		}
	}

	finalData["CurrentPath"] = r.URL.Path
	finalData["Title"] = "SENTRY Interlock"

	// =========================
	// LAYOUT
	// =========================
	layoutBase := "views/layouts/base.html"

	if strings.HasPrefix(tmpl, "auth/") {
		layoutBase = "views/layouts/auth-base.html"
	}

	// =========================
	// TEMPLATE FILES
	// =========================
	files := []string{
		layoutBase,
		"views/layouts/navbar.html",
		"views/layouts/tabs.gohtml",
		"views/layouts/sidebar.gohtml",
		"views/layouts/card_scroll.html",

		// =========================
		// PARTIAL FIELD
		// =========================
		"views/partials/fild/input-floating.gohtml",
		"views/partials/fild/input.gohtml",
		"views/partials/fild/input-file.gohtml",
		"views/partials/fild/input-date.gohtml",
		"views/partials/fild/input-flatpickr.gohtml",
		"views/partials/fild/input-flatpickr-range.gohtml",
		"views/partials/fild/select-search.gohtml",
		"views/partials/fild/select.gohtml",
		"views/partials/fild/select-dinamic-loop.gohtml",
		"views/partials/fild/select-search-loop.gohtml",
		"views/partials/fild/select-dinamic.gohtml",
		"views/partials/fild/select-user-search.gohtml",
		"views/partials/fild/cd-editor.gohtml",
		"views/partials/fild/ckeditor-loop.gohtml",
		"views/partials/fild/select_serach_2.gohtml",

		// =========================
		// BUTTON
		// =========================
		"views/partials/button/btn-active.gohtml",
		"views/partials/button/btn-squere.gohtml",

		// =========================
		// PAGINATION
		// =========================
		"views/partials/pagination.gohtml",

		// =========================
		// MAIN VIEW
		// =========================
		filepath.Join("views", tmpl),
	}

	// =========================
	// FUNCMAP
	// =========================
	funcMap := lucide.FuncMap()

	funcMap["hasPrefix"] = strings.HasPrefix

	funcMap["htmlattr"] = func(s string) template.HTMLAttr {
		return template.HTMLAttr(s)
	}

	funcMap["add"] = func(a, b int) int {
		return a + b
	}

	funcMap["sub"] = func(a, b int) int {
		return a - b
	}

	funcMap["mul"] = func(a, b int) int {
		return a * b
	}

	funcMap["replace"] = func(input, old, new string) string {
		return strings.ReplaceAll(input, old, new)
	}

	funcMap["json"] = func(v interface{}) string {

		bytes, _ := json.Marshal(v)

		return string(bytes)
	}

	funcMap["iterate"] = func(from, to int) []int {

		var res []int

		for i := from; i <= to; i++ {
			res = append(res, i)
		}

		return res
	}

	funcMap["append"] = func(slice []interface{}, item interface{}) []interface{} {
		return append(slice, item)
	}

	funcMap["dict"] = func(values ...interface{}) (map[string]interface{}, error) {

		if len(values)%2 != 0 {
			return nil, fmt.Errorf("invalid dict call")
		}

		dict := make(map[string]interface{})

		for i := 0; i < len(values); i += 2 {

			key, ok := values[i].(string)

			if !ok {
				return nil, fmt.Errorf("dict keys must be strings")
			}

			dict[key] = values[i+1]
		}

		return dict, nil
	}

	funcMap["list"] = func(items ...interface{}) []interface{} {
		return items
	}

	funcMap["safe"] = func(s string) template.HTML {
		return template.HTML(s)
	}

	funcMap["safeHTML"] = func(s string) template.HTML {
		return template.HTML(s)
	}

	// =========================
	// AUDIT
	// =========================
	funcMap["prettyJSON"] = PrettyJSON
	funcMap["diffJSON"] = DiffJSON

	// =========================
	// RISK MATRIX
	// =========================
	funcMap["getMatrixColor"] = func(
		matrices []models.RiskMatrix,
		consID,
		likeID uint,
	) string {

		for _, m := range matrices {

			if m.RiskConsequenceID == consID &&
				m.RiskLikelihoodID == likeID {

				return m.RiskAssessment.Colour
			}
		}

		return "transparent"
	}

	funcMap["getMatrixName"] = func(
		matrices []models.RiskMatrix,
		consID,
		likeID uint,
	) string {

		for _, m := range matrices {

			if m.RiskConsequenceID == consID &&
				m.RiskLikelihoodID == likeID {

				return m.RiskAssessment.Name
			}
		}

		return "-"
	}

	funcMap["getMatrixID"] = func(
		matrices []models.RiskMatrix,
		consID,
		likeID uint,
	) uint {

		for _, m := range matrices {

			if m.RiskConsequenceID == consID &&
				m.RiskLikelihoodID == likeID {

				return m.ID
			}
		}

		return 0
	}

	funcMap["getMatrixAssessmentID"] = func(
		matrices []models.RiskMatrix,
		consID,
		likeID uint,
	) uint {

		for _, m := range matrices {

			if m.RiskConsequenceID == consID &&
				m.RiskLikelihoodID == likeID {

				return m.RiskAssessmentID
			}
		}

		return 0
	}

	funcMap["firstChar"] = func(s string) string {

		if len(s) == 0 {
			return ""
		}

		return strings.ToUpper(string(s[0]))
	}

	// =========================
	// PARSE TEMPLATE
	// =========================
	t, err := template.New("").
		Funcs(funcMap).
		ParseFiles(files...)

	if err != nil {

		http.Error(
			w,
			"Parse Error: "+err.Error(),
			http.StatusInternalServerError,
		)

		return
	}

	// =========================
	// EXECUTE TEMPLATE
	// =========================
	var buf bytes.Buffer

	err = t.ExecuteTemplate(&buf, "base", finalData)

	if err != nil {

		http.Error(
			w,
			"Execute Error: "+err.Error(),
			http.StatusInternalServerError,
		)

		return
	}

	// =========================
	// RESPONSE
	// =========================
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	buf.WriteTo(w)
}
