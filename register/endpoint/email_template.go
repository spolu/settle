package endpoint

import "text/template"

// EmailData is the data required to execute the email template.
type EmailData struct {
	Env      string
	From     string
	Username string
	Email    string
	Mint     string
	CredsURL string
	Secret   string
}

var emailTemplate *template.Template

func init() {
	emailTemplate = template.New("email")
	emailTemplate.Parse(
		"From: Mint Registration <{{.From}}>\r\n" +
			"To: {{.Email}}\r\n" +
			"Subject: Credentials for {{.Username}}@{{.Mint}}\r\n" +
			"\r\n" +
			"Hi {{.Username}}!\n" +
			"\n" +
			"Please click on the link below to verify your address and retrieve your credentials to access the mint at {{.Mint}}[0]:\n" +
			"\n" +
			"{{.CredsURL}}#?qa={{.Env}}&secret={{.Secret}}\n" +
			"\n" +
			"Keep this link safe and secure as this your only way to retrieve or roll your credentials.\n" +
			"\n" +
			"-settle\n" +
			"\n" +
			"[0] required to run `settle login`\n",
	)
}
