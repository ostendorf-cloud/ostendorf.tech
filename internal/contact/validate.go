// Package contact validates contact form requests and turns them into
// mails. Ported from old_simon-ostendorf.de's internal/contact, with one
// change: a FieldError now carries a stable Code instead of a German
// message, because this site answers in German or English depending on the
// visitor's own choice, and internal/webapp maps the code to the message of
// the right locale.
package contact

import (
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Field limits; the form's maxlength attributes use the same values.
const (
	MaxName    = 200
	MaxEmail   = 254
	MaxMessage = 5000
)

// Request is one submitted contact form.
type Request struct {
	Name     string
	Email    string
	Message  string
	Consent  bool
	Honeypot string
}

// Field error codes, stable and machine readable: internal/webapp maps each
// one to the message of the request's locale, so this package never decides
// what language a visitor sees.
const (
	CodeRequired     = "required"
	CodeTooLong      = "too_long"
	CodeInvalidChars = "invalid_chars"
	CodeNonASCII     = "non_ascii_email"
	CodeInvalid      = "invalid"
)

// FieldError names the field that is wrong and the stable code for why.
type FieldError struct {
	Field string
	Code  string
}

func (e *FieldError) Error() string { return e.Field + ": " + e.Code }

// Normalize trims the fields and turns every line break of the message into \n.
func (r *Request) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = strings.TrimSpace(r.Email)
	r.Message = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(r.Message, "\r\n", "\n"), "\r", "\n"))
}

// IsSpam reports whether the honeypot field, invisible to people, was filled.
func (r Request) IsSpam() bool {
	return strings.TrimSpace(r.Honeypot) != ""
}

// Validate checks a normalized request and returns the first problem.
func (r Request) Validate() *FieldError {
	switch {
	case r.Name == "":
		return &FieldError{"name", CodeRequired}
	case !utf8.ValidString(r.Name) || utf8.RuneCountInString(r.Name) > MaxName:
		return &FieldError{"name", CodeTooLong}
	case hasControl(r.Name, false):
		// CR and LF in a name would end up in the mail's Subject header.
		return &FieldError{"name", CodeInvalidChars}
	case r.Email == "":
		return &FieldError{"email", CodeRequired}
	case strings.ContainsFunc(r.Email, func(c rune) bool { return c > unicode.MaxASCII }):
		// The address goes unencoded into Reply-To, which allows ASCII only.
		return &FieldError{"email", CodeNonASCII}
	case len(r.Email) > MaxEmail || !ValidEmail(r.Email):
		return &FieldError{"email", CodeInvalid}
	case r.Message == "":
		return &FieldError{"message", CodeRequired}
	case !utf8.ValidString(r.Message) || utf8.RuneCountInString(r.Message) > MaxMessage:
		return &FieldError{"message", CodeTooLong}
	case hasControl(r.Message, true):
		return &FieldError{"message", CodeInvalidChars}
	case !r.Consent:
		return &FieldError{"consent", CodeRequired}
	}
	return nil
}

// hasControl reports control characters; newlines and tabs are allowed only
// in multi line text.
func hasControl(s string, multiline bool) bool {
	for _, r := range s {
		if multiline && (r == '\n' || r == '\t') {
			continue
		}
		// U+2028 and U+2029, the line and paragraph separators, are not in
		// unicode.IsControl's own category but still do not belong in a mail
		// header or a plain text body.
		if unicode.IsControl(r) || r == ' ' || r == ' ' {
			return true
		}
	}
	return false
}

// ValidEmail accepts a bare ASCII address local@domain.tld without quoting,
// comments or display names, and with a plausible domain. Internationalised
// domains work in their xn-- form.
func ValidEmail(s string) bool {
	if s == "" || len(s) > MaxEmail || !printableASCII(s) || strings.ContainsAny(s, " \t\r\n\"<>()[]\\,;:") {
		return false
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Name != "" || addr.Address != s {
		return false
	}
	local, domain, ok := strings.Cut(s, "@")
	if !ok || local == "" || len(local) > 64 || strings.Contains(domain, "@") {
		return false
	}
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, r := range label {
			//nolint:staticcheck // De Morgan's form negates three range checks separately instead of one grouped "is alphanumeric" disjunction, which reads worse
			if r != '-' && !('a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9') {
				return false
			}
		}
	}
	return true
}
