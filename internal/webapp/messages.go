package webapp

import "github.com/ostendorf-cloud/ostendorf.tech/internal/contact"

// contactMessages are the contact form's own user facing outcomes, in one
// locale. German text is ported verbatim from old_simon-ostendorf.de's
// cmd/website/form.go, with the address it points a visitor at updated to
// simon@ostendorf.tech; English is a translation.
type contactMessages struct {
	sent        string
	unavailable string
	sendFailed  string
	tooMany     string
	noPoW       string
	poWFailed   string
	poWExpired  string
	poWTooFast  string
}

var contactMessagesDE = contactMessages{
	sent:        "Vielen Dank für Ihre Nachricht! Ich melde mich so schnell wie möglich bei Ihnen.",
	unavailable: "Das Kontaktformular ist gerade nicht erreichbar. Bitte schreiben Sie mir direkt an simon@ostendorf.tech.",
	sendFailed:  "Die Nachricht konnte nicht gesendet werden. Bitte versuchen Sie es später noch einmal oder schreiben Sie mir an simon@ostendorf.tech.",
	tooMany:     "Zu viele Anfragen in kurzer Zeit. Bitte versuchen Sie es später noch einmal oder schreiben Sie mir an simon@ostendorf.tech.",
	noPoW:       "Der Spamschutz des Formulars braucht JavaScript. Bitte aktivieren Sie JavaScript oder schreiben Sie mir an simon@ostendorf.tech.",
	poWFailed:   "Der Spamschutz konnte Ihre Anfrage nicht bestätigen. Bitte laden Sie die Seite neu und senden Sie das Formular noch einmal.",
	poWExpired:  "Das Formular war zu lange geöffnet. Bitte senden Sie es noch einmal.",
	poWTooFast:  "Das ging zu schnell. Bitte warten Sie einen Moment und senden Sie das Formular noch einmal.",
}

var contactMessagesEN = contactMessages{
	sent:        "Thank you for your message! I will get back to you as soon as possible.",
	unavailable: "The contact form is not available right now. Please write to me directly at simon@ostendorf.tech.",
	sendFailed:  "The message could not be sent. Please try again later or write to me at simon@ostendorf.tech.",
	tooMany:     "Too many requests in a short time. Please try again later or write to me at simon@ostendorf.tech.",
	noPoW:       "This form's spam protection needs JavaScript. Please enable JavaScript or write to me at simon@ostendorf.tech.",
	poWFailed:   "Spam protection could not confirm your request. Please reload the page and submit the form again.",
	poWExpired:  "The form was open too long. Please submit it again.",
	poWTooFast:  "That was too fast. Please wait a moment and submit the form again.",
}

// messagesFor returns the contact form's own message set for locale,
// German for anything that is not exactly "en".
func messagesFor(locale string) contactMessages {
	if locale == "en" {
		return contactMessagesEN
	}
	return contactMessagesDE
}

// fieldMessage resolves what a contact.FieldError's code means to a
// visitor, in locale. The code is internal/contact's: stable and
// language-free, so this is the one place that decides what it reads like.
func fieldMessage(locale, field, code string) string {
	if locale == "en" {
		return fieldMessageEN(field, code)
	}
	return fieldMessageDE(field, code)
}

func fieldMessageDE(field, code string) string {
	switch field {
	case "name":
		switch code {
		case contact.CodeRequired:
			return "Bitte geben Sie Ihren Namen an."
		case contact.CodeTooLong:
			return "Der Name ist zu lang."
		case contact.CodeInvalidChars:
			return "Der Name enthält ungültige Zeichen."
		}
	case "email":
		switch code {
		case contact.CodeRequired:
			return "Bitte geben Sie Ihre E-Mail-Adresse an."
		case contact.CodeNonASCII:
			return "E-Mail-Adressen mit Umlauten oder anderen Sonderzeichen kann das Formular leider nicht verarbeiten. Bitte nutzen Sie eine andere Adresse oder schreiben Sie mir direkt an simon@ostendorf.tech."
		case contact.CodeInvalid:
			return "Bitte geben Sie eine gültige E-Mail-Adresse an."
		}
	case "message":
		switch code {
		case contact.CodeRequired:
			return "Bitte geben Sie eine Nachricht ein."
		case contact.CodeTooLong:
			return "Die Nachricht ist zu lang (höchstens 5000 Zeichen)."
		case contact.CodeInvalidChars:
			return "Die Nachricht enthält ungültige Zeichen."
		}
	case "consent":
		if code == contact.CodeRequired {
			return "Bitte stimmen Sie der Verarbeitung Ihrer Daten zu."
		}
	}
	return "Bitte überprüfen Sie Ihre Angabe."
}

func fieldMessageEN(field, code string) string {
	switch field {
	case "name":
		switch code {
		case contact.CodeRequired:
			return "Please enter your name."
		case contact.CodeTooLong:
			return "The name is too long."
		case contact.CodeInvalidChars:
			return "The name contains invalid characters."
		}
	case "email":
		switch code {
		case contact.CodeRequired:
			return "Please enter your email address."
		case contact.CodeNonASCII:
			return "This form cannot process email addresses with umlauts or other special characters. Please use a different address or write to me directly at simon@ostendorf.tech."
		case contact.CodeInvalid:
			return "Please enter a valid email address."
		}
	case "message":
		switch code {
		case contact.CodeRequired:
			return "Please enter a message."
		case contact.CodeTooLong:
			return "The message is too long (5000 characters at most)."
		case contact.CodeInvalidChars:
			return "The message contains invalid characters."
		}
	case "consent":
		if code == contact.CodeRequired {
			return "Please agree to the processing of your data."
		}
	}
	return "Please check this field."
}
