package webapp

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ostendorf-cloud/govue/api"
)

// pageMessages are the site's own user facing texts of a refused page
// request, in one locale. The contact form's texts come from govue's
// catalogue in the frontend, by the stable code each answer carries.
type pageMessages struct {
	tooMany string
}

var pageMessagesDE = pageMessages{
	tooMany: "Zu viele Anfragen in kurzer Zeit. Bitte versuchen Sie es später noch einmal oder schreiben Sie mir an simon@ostendorf.tech.",
}

var pageMessagesEN = pageMessages{
	tooMany: "Too many requests in a short time. Please try again later or write to me at simon@ostendorf.tech.",
}

// messagesFor returns the page message set for locale, German for anything
// that is not exactly "en".
func messagesFor(locale string) pageMessages {
	if locale == "en" {
		return pageMessagesEN
	}
	return pageMessagesDE
}

// pageRateLimitRespond answers a denied page, asset, robots.txt,
// sitemap.xml or API request. A page request resolves to a known locale
// through splitLocale, the same way pageHandler itself does, and gets back a
// small HTML document carrying the localized tooMany message: a browser
// loading a page expects HTML, not govue's JSON error envelope, which is
// what it would otherwise see on a 429 here. A request that resolves to no
// locale, meaning an asset, robots.txt, sitemap.xml, an API route or an
// unknown path, keeps govue's default JSON shape.
func (a *App) pageRateLimitRespond(w http.ResponseWriter, r *http.Request, wait time.Duration) {
	locale, _, ok := splitLocale(r.URL.Path)
	if !ok && r.URL.Path == "/" {
		locale, ok = preferredLocale(r.Header.Get("Accept-Language")), true
	}
	if !ok {
		api.WriteError(w, r, api.TooManyRequests(wait))
		return
	}

	messages := messagesFor(locale)
	// The same floor api.WriteError applies to a RetryAfter: a wait under
	// one second would otherwise round down to Retry-After: 0, which asks
	// the browser to retry immediately.
	seconds := int(wait.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Retry-After", strconv.Itoa(seconds))
	w.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintf(w, "<!doctype html><html lang=%q><meta charset=\"utf-8\"><title>%s</title><p>%s</p>",
		locale, messages.tooMany, messages.tooMany)
}
