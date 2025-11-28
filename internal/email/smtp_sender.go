package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// SMTPSender sends email using an SMTP server. It is a minimal implementation
// intended to be swapped with other providers that satisfy the Sender interface.
type SMTPSender struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// NewSMTPSender constructs an SMTPSender.
func NewSMTPSender(host string, port int, username, password, from string) *SMTPSender {
	return &SMTPSender{Host: host, Port: port, Username: username, Password: password, From: from}
}

// Send implements the Sender interface using net/smtp. It supports servers
// that require TLS by establishing a TLS connection when necessary.
func (s *SMTPSender) Send(ctx context.Context, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)

	header := make(map[string]string)
	header["From"] = s.From
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=utf-8"

	var msg strings.Builder
	for k, v := range header {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n" + body)

	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)

	// Try a direct TLS connection first (common for port 465), otherwise use SendMail.
	// Establish raw TCP connection and upgrade to TLS if needed.
	conn, err := net.Dial("tcp", addr)
	if err == nil {
		// try STARTTLS-like upgrade if server supports it by wrapping in TLS
		tlsConn := tls.Client(conn, &tls.Config{ServerName: s.Host})
		if tlsConn != nil {
			// Use smtp.NewClient over the TLS connection then authenticate and send
			c, cerr := smtp.NewClient(tlsConn, s.Host)
			if cerr == nil {
				defer c.Close()
				if auth != nil {
					_ = c.Auth(auth)
				}
				if err := c.Mail(s.From); err != nil {
					return err
				}
				if err := c.Rcpt(to); err != nil {
					return err
				}
				wc, err := c.Data()
				if err != nil {
					return err
				}
				_, err = wc.Write([]byte(msg.String()))
				if err != nil {
					_ = wc.Close()
					return err
				}
				_ = wc.Close()
				return c.Quit()
			}
		}
	}

	// Fallback to net/smtp.SendMail (this will do STARTTLS when supported)
	return smtp.SendMail(addr, auth, s.From, []string{to}, []byte(msg.String()))
}
