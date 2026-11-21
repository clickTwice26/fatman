package mailer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/infrastructure/mailer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSMTPMailer_SendsOTPEmailToMailpit(t *testing.T) {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		host = "localhost"
	}
	port := 1025
	if rawPort := os.Getenv("SMTP_PORT"); rawPort != "" {
		if parsed, err := strconv.Atoi(rawPort); err == nil {
			port = parsed
		}
	}

	m := mailer.NewSMTPMailer(host, port, "no-reply@fatman.local")
	recipient := fmt.Sprintf("mailtest.%s@example.com", uuid.NewString()[:8])
	code := "739204"

	require.NoError(t, m.SendOTP(context.Background(), recipient, code, 10*time.Minute))

	// Query Mailpit API to verify the real email arrived with the OTP code.
	resp, err := http.Get(fmt.Sprintf("http://localhost:8025/api/v1/search?query=%s", recipient))
	require.NoError(t, err, "Docker Mailpit must be running (make infra-up)")
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var searchRes struct {
		Messages []struct {
			ID      string `json:"ID"`
			Subject string `json:"Subject"`
			Snippet string `json:"Snippet"`
		} `json:"messages"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&searchRes))
	require.NotEmpty(t, searchRes.Messages)
	assert.Contains(t, searchRes.Messages[0].Subject, "Fatman")
	assert.Contains(t, searchRes.Messages[0].Snippet, code)
}
