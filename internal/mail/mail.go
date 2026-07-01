package mail

import (
	"html/template"
	"os"
	"path"
	"strings"

	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"gopkg.in/gomail.v2"
)

type Mailer struct {
	User     string
	Password string
}

func NewMailer(user, password string) *Mailer {
	return &Mailer{
		User:     user,
		Password: password,
	}
}

type SendMailOpts struct {
	To      string
	Subject string
	Body    string
}

func (m *Mailer) SendMail(to, subject, body string, templateName models.EmailTemplateName, context interface{}) error {
	fromEmail := m.User

	msg := gomail.NewMessage()

	msg.SetHeader("From", fromEmail)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)

	body, err := renderTemplate(string(templateName), context)
	if err != nil {
		return err
	}

	msg.SetBody("text/html", body)

	dialer := gomail.NewDialer("smtp.gmail.com", 587, m.User, m.Password)

	return dialer.DialAndSend(msg)
}

func renderTemplate[T interface{}](templateName string, context T) (string, error) {
	cwd, err := os.Getwd()

	if err != nil {
		return "", err
	}
	templatePath := path.Join(cwd, "internal/mail/templates", string(templateName))
	tmpl, err := template.ParseFiles(templatePath)

	if err != nil {
		return "", err
	}

	var body strings.Builder

	err = tmpl.Execute(&body, context)

	if err != nil {
		return "", err
	}

	return body.String(), nil
}
