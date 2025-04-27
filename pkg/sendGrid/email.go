package sendGrid

import (
	"context"
	"fmt"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type EmailService interface {
	Send(ctx context.Context, req *models.EmailNotificationRequest) error
	GetSendGridClient() *sendgrid.Client
}

type emailService struct {
	client    *sendgrid.Client
	fromEmail string
	fromName  string
}

func NewEmailService(apiKey string, fromEmail string, fromName string) EmailService {
	return &emailService{client: sendgrid.NewSendClient(apiKey), fromEmail: fromEmail, fromName: fromName}
}

// Send implements EmailService.
func (e *emailService) Send(ctx context.Context, req *models.EmailNotificationRequest) error {

	from := mail.NewEmail(e.fromName, e.fromEmail)
	to := mail.NewEmail("", req.To)

	message := mail.NewV3Mail()
	message.SetFrom(from)

	personalization := mail.NewPersonalization()
	personalization.AddTos(to)

	for _, cc := range req.CC {
		personalization.AddCCs(mail.NewEmail("", cc))
	}

	for _, bcc := range req.BCC {
		personalization.AddBCCs(mail.NewEmail("", bcc))
	}

	personalization.Subject = req.Subject
	message.AddPersonalizations(personalization)

	message.AddContent(mail.NewContent("text/plain", req.Content))
	message.AddContent(mail.NewContent("text/html", req.HTMLContent))

	// send the email
	response, err := e.client.Send(message)

	if err != nil {
		return err
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("failed to send email, status code: %d", response.StatusCode)
	}

	return nil
}

// GetSendGridClient provides access to the internal sendgrid.Client.
func (e *emailService) GetSendGridClient() *sendgrid.Client {
	return e.client
}
