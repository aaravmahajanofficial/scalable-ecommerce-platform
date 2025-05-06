package sendgrid_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	sendgrid_client "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/sendgrid"
	"github.com/sendgrid/sendgrid-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmailService(t *testing.T) {
	// Arrange
	apiKey := "test-api-key"
	fromEmail := "sender@example.com"
	fromName := "Test Sender"

	// Act
	service := sendgrid_client.NewEmailService(apiKey, fromEmail, fromName)

	// Assert
	assert.NotNil(t, service)
}

type sendgridV3Payload struct {
	Personalizations []struct {
		To      []map[string]string `json:"to"`
		Cc      []map[string]string `json:"cc,omitempty"`
		Bcc     []map[string]string `json:"bcc,omitempty"`
		Subject string              `json:"subject"`
	} `json:"personalizations"`
	From    map[string]string `json:"from"`
	Content []struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"content"`
}

func TestEmailService_Send(t *testing.T) {
	apiKey := "SG.test-api-key"
	fromEmail := "from@example.com"
	fromName := "Test Sender"
	ctx := t.Context()

	var mockServer *httptest.Server

	var lastRequestPayload sendgridV3Payload

	var handlerFunc http.HandlerFunc

	// startMockServer sets up and starts the httptest server with the current handlerFunc.
	startMockServer := func() {
		mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusInternalServerError)

				return
			}

			defer r.Body.Close()

			err = json.Unmarshal(bodyBytes, &lastRequestPayload)
			if err != nil {
				http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)

				return
			}

			handlerFunc(w, r)
		}))
	}

	tests := []struct {
		name          string
		req           *models.EmailNotificationRequest
		handler       http.HandlerFunc                              // Mock server handler for this specific test
		expectedError string                                        // Substring expected in the error message, empty for no error
		checkPayload  func(t *testing.T, payload sendgridV3Payload) // Optional payload validation
	}{
		{
			name: "Success - Simple Email",
			req: &models.EmailNotificationRequest{
				To:          "recipient@example.com",
				Subject:     "Test Subject 1",
				Content:     "Plain text content",
				HTMLContent: "<h1>HTML Content</h1>",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Assert
				assert.Equal(t, http.MethodPost, r.Method, "Expected POST request")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "Bearer "+apiKey, r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusAccepted) // 202 Accepted is typical for SendGrid v3 mail/send
			},
			expectedError: "",
			checkPayload: func(t *testing.T, p sendgridV3Payload) {
				require.Len(t, p.Personalizations, 1, "Expected one personalization block")
				pers := p.Personalizations[0]
				require.Len(t, pers.To, 1, "Expected one TO recipient")
				assert.Equal(t, "recipient@example.com", pers.To[0]["email"])
				assert.Empty(t, pers.Cc, "Expected no CC recipients")
				assert.Empty(t, pers.Bcc, "Expected no BCC recipients")
				assert.Equal(t, "Test Subject 1", pers.Subject)

				assert.Equal(t, fromEmail, p.From["email"])
				assert.Equal(t, fromName, p.From["name"])

				require.Len(t, p.Content, 2, "Expected two content blocks (text and html)")
				assert.Equal(t, "text/plain", p.Content[0].Type)
				assert.Equal(t, "Plain text content", p.Content[0].Value)
				assert.Equal(t, "text/html", p.Content[1].Type)
				assert.Equal(t, "<h1>HTML Content</h1>", p.Content[1].Value)
			},
		},
		{
			name: "Success - With CC and BCC",
			req: &models.EmailNotificationRequest{
				To:          "recipient@example.com",
				CC:          []string{"cc1@example.com", "cc2@example.com"},
				BCC:         []string{"bcc1@example.com"},
				Subject:     "Test Subject 2",
				Content:     "Another plain text",
				HTMLContent: "<p>HTML</p>",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			expectedError: "",
			checkPayload: func(t *testing.T, p sendgridV3Payload) {
				require.Len(t, p.Personalizations, 1)
				pers := p.Personalizations[0]
				require.Len(t, pers.To, 1)
				assert.Equal(t, "recipient@example.com", pers.To[0]["email"])
				require.Len(t, pers.Cc, 2, "Expected two CC recipients")
				assert.Equal(t, "cc1@example.com", pers.Cc[0]["email"])
				assert.Equal(t, "cc2@example.com", pers.Cc[1]["email"])
				require.Len(t, pers.Bcc, 1, "Expected one BCC recipient")
				assert.Equal(t, "bcc1@example.com", pers.Bcc[0]["email"])
				assert.Equal(t, "Test Subject 2", pers.Subject)

				require.Len(t, p.Content, 2)
				assert.Equal(t, "Another plain text", p.Content[0].Value)
				assert.Equal(t, "<p>HTML</p>", p.Content[1].Value)
			},
		},
		{
			name: "Failure - SendGrid API Error (4xx)",
			req: &models.EmailNotificationRequest{
				To:      "bad@example.com",
				Subject: "Test Subject 3",
				Content: "Content",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest) // 400 Bad Request
				_, _ = w.Write([]byte(`{"errors": [{"message": "Invalid email"}]}`))
			},
			expectedError: "failed to send email, status code: 400",
			checkPayload: func(t *testing.T, p sendgridV3Payload) {
				require.Len(t, p.Personalizations, 1)
				assert.Equal(t, "bad@example.com", p.Personalizations[0].To[0]["email"])
			},
		},
		{
			name: "Failure - SendGrid API Error (5xx)",
			req: &models.EmailNotificationRequest{
				To:      "recipient@example.com",
				Subject: "Test Subject 4",
				Content: "Content",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError) // 500 Internal Server Error
			},
			expectedError: "failed to send email, status code: 500",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lastRequestPayload = sendgridV3Payload{} // Reset payload capture
			handlerFunc = tc.handler                 // Set the handler for this test

			startMockServer() // Start the server for this test case

			serviceImpl := sendgrid_client.NewEmailService(apiKey, fromEmail, fromName).(sendgrid_client.EmailService)

			sgClient := serviceImpl.GetSendGridClient()

			sgClient.Request.BaseURL = mockServer.URL

			// Act
			err := serviceImpl.Send(ctx, tc.req)

			// Assert
			if tc.expectedError == "" {
				assert.NoError(t, err, "Expected no error")
			} else {
				assert.Error(t, err, "Expected an error")
				assert.Contains(t, err.Error(), tc.expectedError, "Error message mismatch")
			}

			if tc.checkPayload != nil {
				tc.checkPayload(t, lastRequestPayload)
			}

			mockServer.Close()
		})
	}

	t.Run("Failure - Network Error", func(t *testing.T) {
		// Arrange
		startMockServer()

		serviceImpl := sendgrid_client.NewEmailService(apiKey, fromEmail, fromName).(sendgrid_client.EmailService)
		sgClient := serviceImpl.GetSendGridClient()
		sgClient.Request.BaseURL = mockServer.URL
		mockServer.Close()

		req := &models.EmailNotificationRequest{
			To:      "recipient@example.com",
			Subject: "Network Error Test",
			Content: "Content",
		}

		// Act
		err := serviceImpl.Send(ctx, req)

		// Assert
		assert.Error(t, err, "Expected a network error")
		assert.True(t, strings.Contains(err.Error(), "connect: connection refused") || strings.Contains(err.Error(), "dial tcp"), "Expected connection refused or dial tcp error")
	})
}

type emailService = sendgrid_client.EmailService

type testEmailService struct {
	client *sendgrid.Client
}

func (e *testEmailService) Send(ctx context.Context, req *models.EmailNotificationRequest) error {
	return nil
}

func (e *testEmailService) GetSendGridClient() *sendgrid.Client {
	if e.client == nil {
		e.client = sendgrid.NewSendClient("dummy-key-for-test-struct")
	}

	return e.client
}

var _ interface {
	sendgrid_client.EmailService
	GetSendGridClient() *sendgrid.Client
} = &testEmailService{}
