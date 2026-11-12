package contact

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"io"
	"math/big"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/textproto"
	"regexp"
	"strings"
	"testing"
	"time"
)

func valid() Request {
	return Request{Name: "Erika Mustermann", Email: "erika@example.de", Message: "Hallo,\nich brauche Technik.", Consent: true}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*Request)
		field  string
		code   string
	}{
		{"valid", func(*Request) {}, "", ""},
		{"umlauts in name", func(r *Request) { r.Name = "Jörg Größmann"; r.Email = "joerg@xn--mnchen-3ya.de" }, "", ""},
		{"umlauts in email", func(r *Request) { r.Email = "jörg@example.de" }, "email", CodeNonASCII},
		{"umlauts in email domain", func(r *Request) { r.Email = "joerg@münchen.de" }, "email", CodeNonASCII},
		{"missing name", func(r *Request) { r.Name = "" }, "name", CodeRequired},
		{"name too long", func(r *Request) { r.Name = strings.Repeat("ä", MaxName+1) }, "name", CodeTooLong},
		{"name at limit", func(r *Request) { r.Name = strings.Repeat("ä", MaxName) }, "", ""},
		{"CRLF in name", func(r *Request) { r.Name = "Eve\r\nBcc: victim@example.com" }, "name", CodeInvalidChars},
		{"LF in name", func(r *Request) { r.Name = "Eve\nBcc: x@example.com" }, "name", CodeInvalidChars},
		{"NUL in name", func(r *Request) { r.Name = "Eve\x00" }, "name", CodeInvalidChars},
		{"missing email", func(r *Request) { r.Email = "" }, "email", CodeRequired},
		{"email without domain dot", func(r *Request) { r.Email = "a@localhost" }, "email", CodeInvalid},
		{"email with display name", func(r *Request) { r.Email = "Eve <eve@example.com>" }, "email", CodeInvalid},
		{"email with CRLF", func(r *Request) { r.Email = "eve@example.com\r\nBcc: x@example.com" }, "email", CodeInvalid},
		{"email with comma", func(r *Request) { r.Email = "a@example.com,b@example.com" }, "email", CodeInvalid},
		{"email two ats", func(r *Request) { r.Email = "a@b@example.com" }, "email", CodeInvalid},
		{"email too long", func(r *Request) { r.Email = strings.Repeat("a", 64) + "@" + strings.Repeat("b", 190) + ".de" }, "email", CodeInvalid},
		{"email bad label", func(r *Request) { r.Email = "a@-example.de" }, "email", CodeInvalid},
		{"missing message", func(r *Request) { r.Message = "" }, "message", CodeRequired},
		{"message too long", func(r *Request) { r.Message = strings.Repeat("x", MaxMessage+1) }, "message", CodeTooLong},
		{"message at limit", func(r *Request) { r.Message = strings.Repeat("ü", MaxMessage) }, "", ""},
		{"message with escape", func(r *Request) { r.Message = "hi\x1b[31m" }, "message", CodeInvalidChars},
		{"missing consent", func(r *Request) { r.Consent = false }, "consent", CodeRequired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := valid()
			tc.mutate(&r)
			err := r.Validate()
			switch {
			case tc.field == "" && err != nil:
				t.Fatalf("unexpected error %v", err)
			case tc.field != "" && err == nil:
				t.Fatalf("accepted, want error on %s", tc.field)
			case tc.field != "" && (err.Field != tc.field || err.Code != tc.code):
				t.Fatalf("error on %s/%s, want %s/%s", err.Field, err.Code, tc.field, tc.code)
			}
		})
	}
}

func TestNonASCIIEmailRejectedWithItsOwnCode(t *testing.T) {
	t.Parallel()

	r := valid()
	r.Email = "jörg@example.de"
	fe := r.Validate()
	if fe == nil || fe.Field != "email" || fe.Code != CodeNonASCII {
		t.Errorf("error = %v, want email/%s", fe, CodeNonASCII)
	}
	if ValidEmail("jörg@example.de") || ValidEmail("a@bü.de") {
		t.Error("ValidEmail accepted a non ASCII address")
	}
	r.Email = "eve@example.com\r\nBcc: x@example.com"
	if fe := r.Validate(); fe == nil || fe.Code != CodeInvalid {
		t.Errorf("CRLF address: %v, want the generic invalid code", fe)
	}
}

func TestGermanTimeHasSummerTime(t *testing.T) {
	t.Parallel()

	july := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC).In(germanTime)
	if got := july.Format("15:04 MST -07:00"); got != "14:00 CEST +02:00" {
		t.Errorf("July = %s, want 14:00 CEST +02:00", got)
	}
	january := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC).In(germanTime)
	if got := january.Format("15:04 MST -07:00"); got != "13:00 CET +01:00" {
		t.Errorf("January = %s, want 13:00 CET +01:00", got)
	}
}

func TestNormalizeAndHoneypot(t *testing.T) {
	t.Parallel()

	r := Request{Name: "  Max ", Email: " max@example.de\t", Message: "\r\n a\r\nb\rc \n"}
	r.Normalize()
	if r.Name != "Max" || r.Email != "max@example.de" || r.Message != "a\nb\nc" {
		t.Errorf("normalized = %+v", r)
	}
	if r.IsSpam() {
		t.Error("empty honeypot flagged")
	}
	r.Honeypot = "https://spam.example"
	if !r.IsSpam() {
		t.Error("filled honeypot not flagged")
	}
}

var env = Envelope{
	From: &mail.Address{Name: "ostendorf.tech", Address: "no-reply@ostendorf.cloud"},
	To:   &mail.Address{Address: "simon@ostendorf.tech"},
}

var sendTime = time.Date(2026, 9, 30, 14, 5, 0, 0, time.UTC)

func parse(t *testing.T, raw []byte) (*mail.Message, string) {
	t.Helper()
	if bytes.Contains(bytes.ReplaceAll(raw, []byte("\r\n"), nil), []byte("\n")) {
		t.Fatal("bare LF in message")
	}
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parse message: %v", err)
	}
	body, err := io.ReadAll(quotedprintable.NewReader(msg.Body))
	if err != nil {
		t.Fatal(err)
	}
	return msg, string(body)
}

func TestBuildMessage(t *testing.T) {
	t.Parallel()

	req := valid()
	req.Name = "Jörg Größmann"
	req.Message = "Grüße aus Aachen!\n.\nZweite Zeile mit = Zeichen und einer sehr langen Zeile, die länger als sechsundsiebzig Zeichen ist, damit weich umbrochen wird."
	raw, err := Build(env, req, sendTime, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	msg, body := parse(t, raw)
	h := msg.Header

	dec := new(mime.WordDecoder)
	subject, err := dec.DecodeHeader(h.Get("Subject"))
	if err != nil || subject != "Kontaktanfrage von Jörg Größmann" {
		t.Errorf("Subject = %q (%v)", subject, err)
	}
	if got := h.Get("From"); got != `"ostendorf.tech" <no-reply@ostendorf.cloud>` {
		t.Errorf("From = %q", got)
	}
	if got := h.Get("To"); got != "<simon@ostendorf.tech>" {
		t.Errorf("To = %q", got)
	}
	replyTo, err := mail.ParseAddress(h.Get("Reply-To"))
	if err != nil || replyTo.Name != "Jörg Größmann" || replyTo.Address != "erika@example.de" {
		t.Errorf("Reply-To = %q -> %+v (%v)", h.Get("Reply-To"), replyTo, err)
	}
	if date, err := h.Date(); err != nil || !date.Equal(sendTime) {
		t.Errorf("Date = %q (%v)", h.Get("Date"), err)
	}
	if id := h.Get("Message-ID"); !strings.HasPrefix(id, "<") || !strings.HasSuffix(id, "@ostendorf.cloud>") || len(id) != 50 {
		t.Errorf("Message-ID = %q", id)
	}
	if h.Get("Content-Type") != "text/plain; charset=utf-8" || h.Get("Content-Transfer-Encoding") != "quoted-printable" || h.Get("MIME-Version") != "1.0" {
		t.Errorf("MIME headers = %v", h)
	}
	for _, want := range []string{
		"Name: Jörg Größmann\r\n",
		"E-Mail: erika@example.de\r\n",
		"erteilt am 30.09.2026 um 16:05 Uhr",
		"Nachricht:\r\nGrüße aus Aachen!\r\n.\r\nZweite Zeile mit = Zeichen",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body lacks %q:\n%s", want, body)
		}
	}
	for _, line := range strings.Split(string(raw), "\r\n") {
		if len(line) > 998 {
			t.Errorf("line of %d bytes", len(line))
		}
	}
}

func TestBuildLongNonASCIIName(t *testing.T) {
	t.Parallel()

	req := valid()
	req.Name = strings.Repeat("漢", MaxName)
	raw, err := Build(env, req, sendTime, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(raw), "\r\n") {
		if len(line) > 998 {
			t.Fatalf("header line of %d bytes", len(line))
		}
	}
	msg, _ := parse(t, raw)
	replyTo, err := mail.ParseAddress(msg.Header.Get("Reply-To"))
	if err != nil || replyTo.Name != req.Name {
		t.Errorf("folded Reply-To does not round trip: %v", err)
	}
}

// headerLines returns the header block's lines, continuation lines included.
func headerLines(raw []byte) []string {
	head, _, _ := strings.Cut(string(raw), "\r\n\r\n")
	return strings.Split(head, "\r\n")
}

func TestBuildHeadersFoldedAt78(t *testing.T) {
	t.Parallel()

	for name, fullName := range map[string]string{
		"80 umlauts":       strings.Repeat("ä", 80),
		"80 cjk":           strings.Repeat("漢", 80),
		"mixed with space": string([]rune(strings.Repeat("Jörg Größmann ", 6))[:80]),
		"long ascii":       strings.Repeat("Max Mustermann ", 13),
		"ascii specials":   `Eve "=?utf-8?q?x?=" _Q_`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := valid()
			req.Name = strings.TrimSpace(fullName)
			raw, err := Build(env, req, sendTime, rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range headerLines(raw) {
				if len(line) > maxLine {
					t.Errorf("header line of %d characters: %q", len(line), line)
				}
			}
			for _, word := range regexp.MustCompile(`=\?[^ ]*?\?=`).FindAllString(strings.Join(headerLines(raw), "\n"), -1) {
				if len(word) > maxWord {
					t.Errorf("encoded word of %d characters: %q", len(word), word)
				}
			}
			msg, _ := parse(t, raw)
			replyTo, err := mail.ParseAddress(msg.Header.Get("Reply-To"))
			if err != nil || replyTo.Name != req.Name || replyTo.Address != req.Email {
				t.Errorf("Reply-To %q -> %+v (%v)", msg.Header.Get("Reply-To"), replyTo, err)
			}
			subject, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
			if err != nil || subject != "Kontaktanfrage von "+truncateRunes(req.Name, subjectNameRunes) {
				t.Errorf("Subject %q -> %q (%v)", msg.Header.Get("Subject"), subject, err)
			}
		})
	}
}

func TestBuildShortASCIIHeadersStayPlain(t *testing.T) {
	t.Parallel()

	req := valid()
	req.Name = "Max Mustermann"
	raw, err := Build(env, req, sendTime, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	msg, _ := parse(t, raw)
	if got := msg.Header.Get("Subject"); got != "Kontaktanfrage von Max Mustermann" {
		t.Errorf("Subject = %q", got)
	}
	if got := msg.Header.Get("Reply-To"); got != `"Max Mustermann" <erika@example.de>` {
		t.Errorf("Reply-To = %q", got)
	}
}

func TestBuildRefusesHeaderInjection(t *testing.T) {
	t.Parallel()

	for _, req := range []Request{
		{Name: "Eve\r\nBcc: victim@example.com", Email: "eve@example.com", Message: "x", Consent: true},
		{Name: "Eve", Email: "eve@example.com\nBcc: victim@example.com", Message: "x", Consent: true},
	} {
		if _, err := Build(env, req, sendTime, rand.Reader); err == nil {
			t.Errorf("built a message for %q / %q", req.Name, req.Email)
		}
	}
	// A message body may hold anything that looks like a header.
	req := valid()
	req.Message = "Bcc: victim@example.com\n\nSubject: spam"
	raw, err := Build(env, req, sendTime, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	msg, _ := parse(t, raw)
	if msg.Header.Get("Bcc") != "" || len(msg.Header["Subject"]) != 1 {
		t.Error("body text became a header")
	}
}

// ---------- SMTP over net.Pipe ----------

func testCert(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "mail.test"},
		DNSNames:              []string{"mail.test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, pool
}

// session records what the fake server saw.
type session struct {
	commands []string
	auth     string
	data     []byte
}

// fakeSMTP speaks just enough SMTP for net/smtp on one side of a pipe.
func fakeSMTP(conn net.Conn, cert *tls.Certificate, s *session) error {
	defer conn.Close()
	tp := textproto.NewConn(conn)
	if err := tp.PrintfLine("220 mail.test ESMTP"); err != nil {
		return err
	}
	secure := false
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return err
		}
		verb, arg, _ := strings.Cut(line, " ")
		s.commands = append(s.commands, verb)
		switch strings.ToUpper(verb) {
		case "EHLO":
			switch {
			case secure:
				err = tp.PrintfLine("250-mail.test\r\n250 AUTH PLAIN")
			case cert != nil:
				err = tp.PrintfLine("250-mail.test\r\n250 STARTTLS")
			default:
				err = tp.PrintfLine("250-mail.test\r\n250 AUTH PLAIN")
			}
		case "STARTTLS":
			if err := tp.PrintfLine("220 go ahead"); err != nil {
				return err
			}
			tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{*cert}})
			if err := tlsConn.Handshake(); err != nil {
				return err
			}
			conn, tp, secure = tlsConn, textproto.NewConn(tlsConn), true
			continue
		case "AUTH":
			s.auth = arg
			err = tp.PrintfLine("235 ok")
		case "MAIL", "RCPT":
			s.commands[len(s.commands)-1] = line
			err = tp.PrintfLine("250 ok")
		case "DATA":
			if err := tp.PrintfLine("354 go"); err != nil {
				return err
			}
			if s.data, err = tp.ReadDotBytes(); err != nil {
				return err
			}
			err = tp.PrintfLine("250 queued")
		case "QUIT":
			_ = tp.PrintfLine("221 bye")
			return nil
		default:
			err = tp.PrintfLine("502 unknown")
		}
		if err != nil {
			return err
		}
	}
}

func pipeSender(t *testing.T, cert *tls.Certificate, pool *x509.CertPool, s *session) (*Sender, chan error) {
	t.Helper()
	done := make(chan error, 1)
	sender := &Sender{
		Addr:     "mail.test:587",
		HeloName: "ostendorf.tech",
		Timeout:  5 * time.Second,
		Dial: func(context.Context, string, string) (net.Conn, error) {
			client, server := net.Pipe()
			go func() { done <- fakeSMTP(server, cert, s) }()
			return client, nil
		},
		TLSConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}
	return sender, done
}

func TestSendWithSTARTTLS(t *testing.T) {
	t.Parallel()

	cert, pool := testCert(t)
	var s session
	sender, done := pipeSender(t, &cert, pool, &s)
	raw, err := Build(env, valid(), sendTime, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	creds := Credentials{Username: "no-reply@ostendorf.cloud", Password: "secret"}
	if err := sender.Send(context.Background(), creds, "no-reply@ostendorf.cloud", "simon@ostendorf.tech", raw); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("server: %v", err)
	}
	want := "PLAIN " + base64.StdEncoding.EncodeToString([]byte("\x00no-reply@ostendorf.cloud\x00secret"))
	if s.auth != want {
		t.Errorf("AUTH = %q, want %q", s.auth, want)
	}
	joined := strings.Join(s.commands, "|")
	for _, cmd := range []string{"EHLO|STARTTLS|EHLO|AUTH", "MAIL FROM:<no-reply@ostendorf.cloud>", "RCPT TO:<simon@ostendorf.tech>", "DATA|QUIT"} {
		if !strings.Contains(joined, cmd) {
			t.Errorf("commands %q lack %q", joined, cmd)
		}
	}
	msg, err := mail.ReadMessage(bufio.NewReader(bytes.NewReader(s.data)))
	if err != nil || msg.Header.Get("Reply-To") == "" {
		t.Errorf("delivered data unreadable: %v", err)
	}
}

func TestSendRefusesWithoutSTARTTLS(t *testing.T) {
	t.Parallel()

	var s session
	sender, done := pipeSender(t, nil, nil, &s)
	err := sender.Send(context.Background(), Credentials{Username: "u", Password: "secret"}, "a@example.com", "b@example.com", []byte("x"))
	if !errors.Is(err, ErrNoStartTLS) {
		t.Fatalf("err = %v, want ErrNoStartTLS", err)
	}
	<-done
	if s.auth != "" {
		t.Error("credentials sent without TLS")
	}
	for _, c := range s.commands {
		if strings.HasPrefix(c, "MAIL") || c == "AUTH" {
			t.Errorf("sent %q without TLS", c)
		}
	}
}

func TestSendStopsOnCancel(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	sender := &Sender{
		Addr:     "mail.test:587",
		HeloName: "ostendorf.tech",
		// Far longer than the test may take: only the cancellation can end it.
		Timeout: time.Minute,
		Dial: func(context.Context, string, string) (net.Conn, error) {
			client, server := net.Pipe()
			go func() {
				defer close(done)
				defer server.Close()
				// Greet, then read EHLO and never answer.
				tp := textproto.NewConn(server)
				_ = tp.PrintfLine("220 mail.test ESMTP")
				for {
					if _, err := tp.ReadLine(); err != nil {
						return
					}
				}
			}()
			return client, nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	start := time.Now()
	err := sender.Send(ctx, Credentials{Username: "u", Password: "secret"}, "a@example.com", "b@example.com", []byte("x"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("Send took %s after the cancellation", d)
	}
	<-done
}

func TestSendRejectsUntrustedCertificate(t *testing.T) {
	t.Parallel()

	cert, _ := testCert(t)
	var s session
	sender, done := pipeSender(t, &cert, x509.NewCertPool(), &s)
	// The unbuffered pipe blocks the server's handshake flight once the
	// client stops reading; the deadline ends it.
	sender.Timeout = 300 * time.Millisecond
	err := sender.Send(context.Background(), Credentials{Username: "u", Password: "secret"}, "a@example.com", "b@example.com", []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "starttls") {
		t.Fatalf("err = %v, want a starttls error", err)
	}
	<-done
	if s.auth != "" {
		t.Error("credentials sent to an untrusted server")
	}
}
