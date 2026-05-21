package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"reflect"
)

func PrettyJSON(input string) string {

	var out bytes.Buffer

	err := json.Indent(&out, []byte(input), "", "  ")
	if err != nil {
		return input
	}

	return out.String()
}

func DiffJSON(beforeStr, afterStr string) template.HTML {

	var before map[string]interface{}
	var after map[string]interface{}

	_ = json.Unmarshal([]byte(beforeStr), &before)
	_ = json.Unmarshal([]byte(afterStr), &after)

	ignored := map[string]bool{
		"CreatedAt":     true,
		"UpdatedAt":     true,
		"DeletedAt":     true,
		"EventCategory": true,
		"RiskMatrix":    true,
		"ScatOption":    true,
		"Location":      true,
		"PIC":           true,
		"Department":    true,
		"Contractor":    true,
		"ReportBy":      true,
		"Audits":        true,
	}

	html := `
	<div class="overflow-x-auto">
	<table class="table table-xs w-full border rounded-xl">
		<thead class="bg-base-200">
			<tr>
				<th class="w-[180px]">FIELD</th>
				<th class="text-error">BEFORE</th>
				<th class="text-success">AFTER</th>
			</tr>
		</thead>
		<tbody>
	`

	for key, afterVal := range after {

		beforeVal := before[key]

		// =========================
		// HANDLE DOCUMENTATIONS
		// =========================
		if key == "documentations" {

			docHTML := CompareDocumentations(beforeVal, afterVal)

			if docHTML != "" {
				html += docHTML
			}

			continue
		}

		// =========================
		// IGNORE FIELD
		// =========================
		if ignored[key] {
			continue
		}

		// =========================
		// SKIP IF SAME
		// =========================
		if reflect.DeepEqual(beforeVal, afterVal) {
			continue
		}

		html += fmt.Sprintf(`
		<tr class="align-top border-b">

			<td class="font-semibold text-[11px] uppercase bg-base-100">
				%s
			</td>

			<td class="bg-red-50 text-red-700 whitespace-pre-wrap break-words max-w-[300px] text-[11px]">
				%v
			</td>

			<td class="bg-green-50 text-green-700 whitespace-pre-wrap break-words max-w-[300px] text-[11px]">
				%v
			</td>

		</tr>
		`,
			key,
			FormatAuditValue(beforeVal),
			FormatAuditValue(afterVal),
		)
	}

	html += `
		</tbody>
	</table>
	</div>
	`

	return template.HTML(html)
}

func FormatAuditValue(v interface{}) string {

	if v == nil {
		return "-"
	}

	switch val := v.(type) {

	case string:
		if val == "" {
			return "-"
		}
		return val

	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

func CompareDocumentations(beforeVal, afterVal interface{}) string {

	beforeDocs := ExtractDocURLs(beforeVal)
	afterDocs := ExtractDocURLs(afterVal)

	added := []string{}
	removed := []string{}

	beforeMap := map[string]bool{}
	afterMap := map[string]bool{}

	for _, v := range beforeDocs {
		beforeMap[v] = true
	}

	for _, v := range afterDocs {
		afterMap[v] = true
	}

	for _, v := range afterDocs {
		if !beforeMap[v] {
			added = append(added, v)
		}
	}

	for _, v := range beforeDocs {
		if !afterMap[v] {
			removed = append(removed, v)
		}
	}

	if len(added) == 0 && len(removed) == 0 {
		return ""
	}

	html := `
	<tr>
		<td class="font-semibold">DOCUMENTATIONS</td>

		<td class="bg-red-50 text-red-700">
	`

	if len(removed) == 0 {
		html += "-"
	} else {
		for _, v := range removed {
			html += "❌ " + v + "<br>"
		}
	}

	html += `</td><td class="bg-green-50 text-green-700">`

	if len(added) == 0 {
		html += "-"
	} else {
		for _, v := range added {
			html += "✅ " + v + "<br>"
		}
	}

	html += `
		</td>
	</tr>
	`

	return html
}

func ExtractDocURLs(v interface{}) []string {

	result := []string{}

	arr, ok := v.([]interface{})
	if !ok {
		return result
	}

	for _, item := range arr {

		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		doc, ok := obj["Documentation"].(map[string]interface{})
		if !ok {
			continue
		}

		url, ok := doc["file_url"].(string)
		if ok {
			result = append(result, url)
		}
	}

	return result
}
