package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"

	"github.com/resend/resend-go/v3"
)

//go:embed templates/*.html
var templateFS embed.FS
var tmpl *template.Template

func init() {
	tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))
}

type Client struct {
	resend    *resend.Client
	fromEmail string
	baseURL   string
}

func New(apiKey, fromEmail, baseURL string) *Client {
	return &Client{
		resend:    resend.NewClient(apiKey),
		fromEmail: fromEmail,
		baseURL:   baseURL,
	}
}

func (c *Client) VerificationURL(code string) string {
	return c.baseURL + "/verify?code=" + code
}

func (c *Client) SendVerification(toEmail, username, verificationURL string) error {
	var body bytes.Buffer
	err := tmpl.ExecuteTemplate(&body, "verification.html", map[string]any{
		"Username":        username,
		"VerificationURL": verificationURL,
	})
	if err != nil {
		return fmt.Errorf("failed to execute verification template: %w", err)
	}

	_, err = c.resend.Emails.Send(&resend.SendEmailRequest{
		From:    c.fromEmail,
		To:      []string{toEmail},
		Subject: "Verify your email address",
		Html:    body.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}
	return nil
}

func (c *Client) PasswordResetURL(token string) string {
	return c.baseURL + "/reset-password?token=" + token
}

func (c *Client) SendPasswordReset(toEmail, username, resetURL string) error {
	var body bytes.Buffer
	err := tmpl.ExecuteTemplate(&body, "password_reset.html", map[string]any{
		"Username": username,
		"ResetURL": resetURL,
	})
	if err != nil {
		return fmt.Errorf("failed to execute password reset template: %w", err)
	}

	_, err = c.resend.Emails.Send(&resend.SendEmailRequest{
		From:    c.fromEmail,
		To:      []string{toEmail},
		Subject: "Reset your password",
		Html:    body.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}
	return nil
}
