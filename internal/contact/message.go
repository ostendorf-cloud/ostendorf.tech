package contact

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	// The scratch image has no zoneinfo. Imported here, not only in main:
	// Go initialises a package's imports before its variables, but unrelated
	// packages by path, so germanTime would be computed before main's import.
	_ "time/tzdata"
)

// subjectNameRunes shortens long names in the subject; the body has the full name.
const subjectNameRunes = 80

const (
	// maxLine is the line length RFC 5322 asks for, without CRLF.
	maxLine = 78
	// maxWord is the RFC 2047 limit for one encoded word.
	maxWord = 75
)

// germanTime prints times in the mail body the way Simon reads them, with
// summer time: the mail this package builds is always for him, whatever
// language the visitor who sent it saw on the page. The embedded zoneinfo
// makes a failure a build bug.
var germanTime = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic("contact: " + err.Error())
	}
	return loc
}()

// Envelope holds the fixed addresses of every contact mail.
type Envelope struct {
	From *mail.Address
	To   *mail.Address
}

// plainHeader reports whether text can stand in a header as it is: ASCII,
// and nothing a mail client would decode as an encoded word.
func plainHeader(text string) bool {
	return printableASCII(text) && !strings.Contains(text, "=?")
}

func printableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

// qSafe are the bytes RFC 2047 allows unencoded in a phrase.
func qSafe(b byte) bool {
	return 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || '0' <= b && b <= '9' || strings.IndexByte("!*+-/", b) >= 0
}

// encodeWords turns s into Q encoded words of at most maxWord characters,
// each holding whole runes, folded onto continuation lines. first is the
// room left on the header's first line.
func encodeWords(s string, first int) string {
	const prefix, suffix = "=?utf-8?q?", "?="
	var out, word, enc strings.Builder
	room := min(first, maxWord)
	flush := func() {
		if out.Len() > 0 {
			out.WriteString("\r\n ")
		}
		out.WriteString(prefix + word.String() + suffix)
		word.Reset()
		room = maxWord
	}
	var buf [utf8.UTFMax]byte
	for _, r := range s {
		enc.Reset()
		for _, b := range buf[:utf8.EncodeRune(buf[:], r)] {
			switch {
			case b == ' ':
				enc.WriteByte('_')
			case qSafe(b):
				enc.WriteByte(b)
			default:
				fmt.Fprintf(&enc, "=%02X", b)
			}
		}
		if word.Len() > 0 && len(prefix)+word.Len()+enc.Len()+len(suffix) > room {
			flush()
		}
		word.WriteString(enc.String())
	}
	if word.Len() > 0 || out.Len() == 0 {
		flush()
	}
	return out.String()
}

// lastLine is the length of the last line of a header, name included.
func lastLine(name, value string) int {
	if i := strings.LastIndexByte(value, '\n'); i >= 0 {
		return len(value) - i - 1
	}
	return len(name) + 2 + len(value)
}

// subjectHeader keeps a short ASCII subject readable and encodes the rest.
func subjectHeader(text string) string {
	if plainHeader(text) && lastLine("Subject", text) <= maxLine {
		return text
	}
	return encodeWords(text, maxLine-len("Subject: "))
}

// replyToHeader encodes the name when it is not short ASCII; the address
// is ASCII (ValidEmail) and gets a line of its own when it does not fit.
func replyToHeader(name, address string) string {
	plain := (&mail.Address{Name: name, Address: address}).String()
	if plainHeader(name) && lastLine("Reply-To", plain) <= maxLine {
		return plain
	}
	words := encodeWords(name, maxLine-len("Reply-To: "))
	sep := " "
	if lastLine("Reply-To", words)+len(" <"+address+">") > maxLine {
		sep = "\r\n "
	}
	return words + sep + "<" + address + ">"
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// Build renders the mail for a validated request. It refuses CR or LF in
// any header value as a second line of defence against header injection.
func Build(env Envelope, req Request, now time.Time, random io.Reader) ([]byte, error) {
	if env.From == nil || env.To == nil {
		return nil, errors.New("contact: envelope incomplete")
	}
	if strings.ContainsAny(req.Name+req.Email, "\r\n") {
		return nil, errors.New("contact: line break in a header field")
	}
	_, domain, ok := strings.Cut(env.From.Address, "@")
	if !ok {
		return nil, errors.New("contact: from address without domain")
	}
	var id [16]byte
	if _, err := io.ReadFull(random, id[:]); err != nil {
		return nil, fmt.Errorf("contact: message id: %w", err)
	}

	var buf bytes.Buffer
	headers := [][2]string{
		{"From", env.From.String()},
		{"To", env.To.String()},
		{"Reply-To", replyToHeader(req.Name, req.Email)},
		{"Subject", subjectHeader("Kontaktanfrage von " + truncateRunes(req.Name, subjectNameRunes))},
		{"Date", now.Format(time.RFC1123Z)},
		{"Message-ID", "<" + hex.EncodeToString(id[:]) + "@" + domain + ">"},
		{"MIME-Version", "1.0"},
		{"Content-Type", "text/plain; charset=utf-8"},
		{"Content-Transfer-Encoding", "quoted-printable"},
		{"Auto-Submitted", "auto-generated"},
	}
	for _, h := range headers {
		fmt.Fprintf(&buf, "%s: %s\r\n", h[0], h[1])
	}
	buf.WriteString("\r\n")

	var body strings.Builder
	fmt.Fprintf(&body, "Neue Kontaktanfrage über das Formular auf ostendorf.tech\n\n")
	fmt.Fprintf(&body, "Name: %s\n", req.Name)
	fmt.Fprintf(&body, "E-Mail: %s\n", req.Email)
	fmt.Fprintf(&body, "Datenschutz-Einverständnis: erteilt am %s\n\n", now.In(germanTime).Format("02.01.2006 um 15:04 Uhr"))
	fmt.Fprintf(&body, "Nachricht:\n%s\n", req.Message)

	qp := quotedprintable.NewWriter(&buf)
	if _, err := qp.Write([]byte(body.String())); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
