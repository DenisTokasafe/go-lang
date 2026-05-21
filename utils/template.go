package utils

import (
	"bytes"
	"encoding/json"
	"html/template"
)

func ToJSON(v interface{}) template.JS {
	b, _ := json.Marshal(v)
	return template.JS(b)
}

func prettyJSON(input string) string {

	var out bytes.Buffer

	err := json.Indent(&out, []byte(input), "", "  ")
	if err != nil {
		return input
	}

	return out.String()
}
