package contact

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

// ErrNoStartTLS means the server did not offer STARTTLS; the sender then
// stops before any credentials or form data leave the pod.
var ErrNoStartTLS = errors.New("smtp: server does not offer STARTTLS")

// Credentials log in to the submission port.
type Credentials struct {
	Username string
	Password string
}

// Sender submits mails over SMTP with STARTTLS required and AUTH PLAIN.
type Sender struct {
	// Addr is host:port of the submission server.
	Addr string
	// HeloName is the name sent in EHLO.
	HeloName string
	// Timeout bounds the whole conversation.
	Timeout time.Duration
	// Dial opens the connection; tests replace it with net.Pipe.
	Dial func(ctx context.Context, network, addr string) (net.Conn, error)
	// TLSConfig overrides the TLS settings; tests use it to trust their CA.
	TLSConfig *tls.Config
}

// NewSender returns a sender with a 20 second timeout.
func NewSender(addr, heloName string) *Sender {
	d := &net.Dialer{Timeout: 10 * time.Second}
	return &Sender{Addr: addr, HeloName: heloName, Timeout: 20 * time.Second, Dial: d.DialContext}
}

// Send delivers msg from one address to another.
func (s *Sender) Send(ctx context.Context, creds Credentials, from, to string, msg []byte) (err error) {
	host, _, err := net.SplitHostPort(s.Addr)
	if err != nil {
		return fmt.Errorf("smtp: address %q: %w", s.Addr, err)
	}
	conn, err := s.Dial(ctx, "tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("smtp: dial: %w", err)
	}
	deadline := time.Now().Add(s.Timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	// Cancelling the context unblocks a pending read or write at once.
	stop := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Unix(1, 0)) })
	defer func() {
		// The deadline error alone would hide why the conversation ended.
		if !stop() && err != nil {
			err = fmt.Errorf("%w (%w)", err, context.Cause(ctx))
		}
	}()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			return fmt.Errorf("smtp: greeting: %w (close: %w)", err, closeErr)
		}
		return fmt.Errorf("smtp: greeting: %w", err)
	}
	defer func() {
		closeErr := c.Close()
		// A successful Quit below already closed the connection; this
		// second Close then only reports that it is already closed, which
		// carries no information about the send itself.
		if closeErr == nil || errors.Is(closeErr, net.ErrClosed) {
			return
		}
		if err != nil {
			err = fmt.Errorf("%w (close: %w)", err, closeErr)
		} else {
			err = fmt.Errorf("smtp: close: %w", closeErr)
		}
	}()

	if err := c.Hello(s.HeloName); err != nil {
		return fmt.Errorf("smtp: ehlo: %w", err)
	}
	if ok, _ := c.Extension("STARTTLS"); !ok {
		return ErrNoStartTLS
	}
	cfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if s.TLSConfig != nil {
		cfg = s.TLSConfig.Clone()
		if cfg.ServerName == "" {
			cfg.ServerName = host
		}
	}
	if err := c.StartTLS(cfg); err != nil {
		return fmt.Errorf("smtp: starttls: %w", err)
	}
	// PlainAuth itself refuses to send the password without TLS.
	if err := c.Auth(smtp.PlainAuth("", creds.Username, creds.Password, host)); err != nil {
		return fmt.Errorf("smtp: auth: %w", err)
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp: mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp: rcpt to: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp: data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp: end of data: %w", err)
	}
	// The server accepted the mail with the reply to the end of data; a
	// failing QUIT, often a close_notify racing the server's close, changes
	// nothing about that.
	_ = c.Quit()
	return nil
}
