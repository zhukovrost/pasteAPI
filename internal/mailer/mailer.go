package mailer

import (
	"bytes"
	"embed"
	"github.com/go-mail/mail"
	"html/template"
	"sync"
	"time"
)

//go:embed "templates"
var templateFS embed.FS

type Mailer struct {
	dialer *mail.Dialer
	sender string
}

var (
	once     sync.Once
	myMailer *Mailer
)

func New(host string, port int, username, password, sender string) *Mailer {
	once.Do(func() {
		myMailer = new(Mailer)
		dialer := mail.NewDialer(host, port, username, password)
		dialer.Timeout = 5 * time.Second
		myMailer.dialer = dialer
		myMailer.sender = sender

	})

	return myMailer
}

func (m *Mailer) Send(recipient, templateFile string, data interface{}) error {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	// trying to send email 3 times
	for i := 1; i <= 3; i++ {
		err = m.dialer.DialAndSend(msg)
		if nil == err {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return err
}
