package email

import (
	"bytes"
	"fmt"
	"formhub/internal/config"
	"html/template"
	"strings"

	"gopkg.in/gomail.v2"
)

type SMTPService struct {
	config config.SMTPConfig
	dialer *gomail.Dialer
}

type EmailData struct {
	FormName    string
	Subject     string
	FromEmail   string
	ToEmails    []string
	CCEmails    []string
	SubmissionData map[string]interface{}
	IPAddress   string
	Timestamp   string
}

func NewSMTPService(cfg config.SMTPConfig) (*SMTPService, error) {
	dialer := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	
	return &SMTPService{
		config: cfg,
		dialer: dialer,
	}, nil
}

func (s *SMTPService) SendFormSubmission(data EmailData) error {
	m := gomail.NewMessage()
	
	// Set headers
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail))
	m.SetHeader("To", data.ToEmails...)
	if len(data.CCEmails) > 0 {
		m.SetHeader("Cc", data.CCEmails...)
	}
	
	subject := data.Subject
	if subject == "" {
		subject = fmt.Sprintf("New form submission from %s", data.FormName)
	}
	m.SetHeader("Subject", subject)
	
	// Generate HTML body
	htmlBody, err := s.generateHTMLBody(data)
	if err != nil {
		return fmt.Errorf("failed to generate HTML body: %w", err)
	}
	
	// Generate text body
	textBody := s.generateTextBody(data)
	
	m.SetBody("text/plain", textBody)
	m.AddAlternative("text/html", htmlBody)
	
	// Send email
	if err := s.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	return nil
}

func (s *SMTPService) generateHTMLBody(data EmailData) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>New Form Submission</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .submission-data { background: white; padding: 15px; border-radius: 8px; margin: 15px 0; }
        .field { margin-bottom: 10px; }
        .field-label { font-weight: bold; color: #555; }
        .field-value { margin-left: 10px; }
        .footer { padding: 20px; text-align: center; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>New Form Submission</h1>
        <p>{{.FormName}}</p>
    </div>
    
    <div class="content">
        <div class="submission-data">
            <h3>Submission Details:</h3>
            {{range $key, $value := .SubmissionData}}
            <div class="field">
                <span class="field-label">{{$key}}:</span>
                <span class="field-value">{{$value}}</span>
            </div>
            {{end}}
        </div>
        
        <div class="submission-data">
            <h3>Technical Details:</h3>
            <div class="field">
                <span class="field-label">IP Address:</span>
                <span class="field-value">{{.IPAddress}}</span>
            </div>
            <div class="field">
                <span class="field-label">Timestamp:</span>
                <span class="field-value">{{.Timestamp}}</span>
            </div>
        </div>
    </div>
    
    <div class="footer">
        <p>This email was sent by FormHub - Form Backend Service</p>
    </div>
</body>
</html>`

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *SMTPService) generateTextBody(data EmailData) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("New Form Submission - %s\n", data.FormName))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	sb.WriteString("Submission Details:\n")
	sb.WriteString(strings.Repeat("-", 20) + "\n")
	for key, value := range data.SubmissionData {
		sb.WriteString(fmt.Sprintf("%s: %v\n", key, value))
	}
	
	sb.WriteString("\nTechnical Details:\n")
	sb.WriteString(strings.Repeat("-", 20) + "\n")
	sb.WriteString(fmt.Sprintf("IP Address: %s\n", data.IPAddress))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", data.Timestamp))
	
	sb.WriteString("\n" + strings.Repeat("=", 50) + "\n")
	sb.WriteString("This email was sent by FormHub - Form Backend Service\n")
	
	return sb.String()
}