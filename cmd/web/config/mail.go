package config

import (
	"crypto/tls"
	"os"
	"strconv"
	"strings" // Diperlukan untuk replacer template HTML

	"gopkg.in/gomail.v2"
)

// SendEmail menangani pengiriman email mentah menggunakan konfigurasi .env / Mailtrap
func SendEmail(
	to []string,
	subject string,
	htmlBody string,
) error {

	mailHost := os.Getenv("MAIL_HOST")
	mailPortStr := os.Getenv("MAIL_PORT")

	mailUsername := os.Getenv("MAIL_USERNAME")
	mailPassword := os.Getenv("MAIL_PASSWORD")

	mailFromAddress := os.Getenv("MAIL_FROM_ADDRESS")
	mailFromName := os.Getenv("MAIL_FROM_NAME")

	mailPort, _ := strconv.Atoi(
		mailPortStr,
	)

	if mailPort == 0 {

		mailPort = 587
	}

	m := gomail.NewMessage()

	m.SetHeader(
		"From",
		m.FormatAddress(
			mailFromAddress,
			mailFromName,
		),
	)

	m.SetHeader("To", to...)

	m.SetHeader(
		"Subject",
		subject,
	)

	m.SetBody(
		"text/html",
		htmlBody,
	)

	d := gomail.NewDialer(
		mailHost,
		mailPort,
		mailUsername,
		mailPassword,
	)

	d.TLSConfig = &tls.Config{
		ServerName:         mailHost,
		InsecureSkipVerify: true,
	}

	return d.DialAndSend(m)
}

// EmailTemplate adalah pembungkus (wrapper) HTML master agar tampilan email terlihat elegan dan modern
func EmailTemplate(title string, previewText string, contentHTML string) string {
	template := `
	<!DOCTYPE html>
	<html lang="id">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>{{TITLE}}</title>
	</head>
	<body style="margin: 0; padding: 0; background-color: #f6f8fa; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; -webkit-font-smoothing: antialiased; color: #333333;">
		
		<div style="display: none; max-height: 0px; overflow: hidden;">
			{{PREVIEW_TEXT}}
		</div>

		<table width="100%" border="0" cellspacing="0" cellpadding="0" style="background-color: #f6f8fa; padding: 20px 0;">
			<tr>
				<td align="center">
					
					<table width="100%" max-width="600" border="0" cellspacing="0" cellpadding="0" style="max-width: 600px; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05); border: 1px solid #e1e4e8;">
						
						<tr>
							<td style="background-color: #1e293b; padding: 30px 40px; text-align: left;">
								<h1 style="margin: 0; color: #ffffff; font-size: 22px; font-weight: 600; letter-spacing: -0.5px;">
									{{TITLE}}
								</h1>
							</td>
						</tr>

						<tr>
							<td style="padding: 40px; font-size: 15px; line-height: 1.6; color: #444444;">
								{{CONTENT}}
							</td>
						</tr>

						<tr>
							<td style="background-color: #f8fafc; padding: 20px 40px; text-align: center; font-size: 12px; color: #64748b; border-top: 1px solid #e2e8f0;">
								<p style="margin: 0 0 8px 0; font-weight: 500;">SENTRY Interlock System</p>
								<p style="margin: 0; font-size: 11px; color: #94a3b8;">Email ini dikirim otomatis oleh sistem. Mohon tidak membalas email ini.</p>
							</td>
						</tr>

					</table>

				</td>
			</tr>
		</table>

	</body>
	</html>
	`

	// Proses replacing tag penanda dengan konten real
	r := strings.NewReplacer(
		"{{TITLE}}", title,
		"{{PREVIEW_TEXT}}", previewText,
		"{{CONTENT}}", contentHTML,
	)

	return r.Replace(template)
}
