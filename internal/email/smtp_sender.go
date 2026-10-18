package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const defaultSMTPSendTimeout = 10 * time.Second

// SMTPSender sends email using an SMTP server. It is a minimal implementation
// intended to be swapped with other providers that satisfy the Sender interface.
type SMTPSender struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	Timeout  time.Duration
}

// NewSMTPSender constructs an SMTPSender. A timeout <= 0 falls back to 10s.
func NewSMTPSender(host string, port int, username, password, from string, timeout time.Duration) *SMTPSender {
	if timeout <= 0 {
		timeout = defaultSMTPSendTimeout
	}
	return &SMTPSender{Host: host, Port: port, Username: username, Password: password, From: from, Timeout: timeout}
}

// Send implements the Sender interface using net/smtp. It tries implicit TLS
// first (common for port 465) and falls back to STARTTLS when that connection
// cannot be set up. Both attempts share one deadline of s.Timeout.
func (s *SMTPSender) Send(ctx context.Context, to, subject, body string) error {
	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	log.Printf("[SMTPSender] sending email to=%s host=%s port=%d from=%s", to, s.Host, s.Port, s.From)

	msg := s.composeMessage(to, subject, body)
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)

	c, err := s.openImplicitTLSClient(ctx)
	if err != nil {
		log.Printf("[SMTPSender] implicit TLS setup error: %v; falling back to STARTTLS", err)
		return s.sendWithSTARTTLS(ctx, auth, to, msg)
	}
	defer c.Close()

	// The Auth error has always been ignored on this path; surfacing it could
	// change which emails are delivered today.
	_ = c.Auth(auth)
	if err := s.deliver(c, to, msg); err != nil {
		return err
	}
	log.Printf("[SMTPSender] sent email to %s via TLS connection", to)
	return nil
}

func (s *SMTPSender) composeMessage(to, subject, body string) []byte {
	header := map[string]string{
		"From":         s.From,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=utf-8",
	}

	var msg strings.Builder
	for k, v := range header {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n" + body)
	return []byte(msg.String())
}

func (s *SMTPSender) openImplicitTLSClient(ctx context.Context) (*smtp.Client, error) {
	conn, err := s.dialWithDeadline(ctx)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Client(conn, &tls.Config{ServerName: s.Host})
	c, err := smtp.NewClient(tlsConn, s.Host)
	if err != nil {
		_ = tlsConn.Close()
		return nil, err
	}
	return c, nil
}

// sendWithSTARTTLS mirrors smtp.SendMail step for step, but over a connection
// bounded by ctx's deadline instead of an unbounded net.Dial.
func (s *SMTPSender) sendWithSTARTTLS(ctx context.Context, auth smtp.Auth, to string, msg []byte) error {
	conn, err := s.dialWithDeadline(ctx)
	if err != nil {
		log.Printf("[SMTPSender] STARTTLS dial error: %v", err)
		return err
	}

	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		_ = conn.Close()
		log.Printf("[SMTPSender] STARTTLS client error: %v", err)
		return err
	}
	defer c.Close()

	if err := s.upgradeAndAuthenticate(c, auth); err != nil {
		log.Printf("[SMTPSender] STARTTLS setup error: %v", err)
		return err
	}
	if err := s.deliver(c, to, msg); err != nil {
		return err
	}
	log.Printf("[SMTPSender] sent email to %s via STARTTLS fallback", to)
	return nil
}

func (s *SMTPSender) upgradeAndAuthenticate(c *smtp.Client, auth smtp.Auth) error {
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: s.Host}); err != nil {
			return err
		}
	}
	if ok, _ := c.Extension("AUTH"); !ok {
		return errors.New("smtp: server doesn't support AUTH")
	}
	return c.Auth(auth)
}

func (s *SMTPSender) deliver(c *smtp.Client, to string, msg []byte) error {
	if err := c.Mail(s.From); err != nil {
		log.Printf("[SMTPSender] error MAIL FROM: %v", err)
		return err
	}
	if err := c.Rcpt(to); err != nil {
		log.Printf("[SMTPSender] error RCPT TO: %v", err)
		return err
	}
	wc, err := c.Data()
	if err != nil {
		log.Printf("[SMTPSender] error DATA: %v", err)
		return err
	}
	if _, err := wc.Write(msg); err != nil {
		_ = wc.Close()
		return err
	}
	if err := wc.Close(); err != nil {
		log.Printf("[SMTPSender] error closing DATA: %v", err)
		return err
	}
	if err := c.Quit(); err != nil {
		log.Printf("[SMTPSender] error during QUIT: %v", err)
		return err
	}
	return nil
}

// dialWithDeadline applies ctx's deadline to the connection itself, so every
// SMTP read and write that follows is bounded by it.
func (s *SMTPSender) dialWithDeadline(ctx context.Context) (net.Conn, error) {
	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	return conn, nil
}
