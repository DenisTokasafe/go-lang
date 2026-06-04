package config

import "os"

func AppURL() string {

	url := os.Getenv("APP_URL")

	if url == "" {

		return "http://localhost:8080"
	}

	return url
}
