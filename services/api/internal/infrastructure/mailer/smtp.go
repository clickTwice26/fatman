package mailer

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// SMTPMailer delivers transactional onboarding OTP emails over SMTP (Mailpit in dev, SMTP relay in prod).
type SMTPMailer struct {
	host string
	port int
	from string
}

// NewSMTPMailer constructs a new SMTPMailer.
func NewSMTPMailer(host string, port int, from string) *SMTPMailer {
	return &SMTPMailer{
		host: host,
		port: port,
		from: from,
	}
}

// SendOTP emails a 6-digit Fatman verification code to recipient.
func (m *SMTPMailer) SendOTP(_ context.Context, toEmail, code string, expiresIn time.Duration) error {
	addr := net.JoinHostPort(m.host, strconv.Itoa(m.port))
	mins := int(expiresIn.Minutes())
	if mins < 1 {
		mins = 1
	}

	subject := "Your Fatman sign-in verification code"
	body := fmt.Sprintf(
		"Welcome to Fatman!\r\n\r\n"+
			"Your 6-digit verification code is: %s\r\n\r\n"+
			"This code expires in %d minutes and can only be used once.\r\n"+
			"If you didn't request this code, you can safely ignore this email.\r\n",
		code,
		mins,
	)

	msg := strings.Join([]string{
		"From: Fatman Community <" + m.from + ">",
		"To: " + toEmail,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	if err := smtp.SendMail(addr, nil, m.from, []string{toEmail}, []byte(msg)); err != nil {
		return fmt.Errorf("send otp smtp mail: %w", err)
	}
	return nil
}
