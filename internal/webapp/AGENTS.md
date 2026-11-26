# internal/webapp

## Contact form

The contact form is govue's: the Go handlers (`contact`, with `pow` and `mail`) and the
Vue form (`GovueContactForm` with its proof of work worker). The site has no contact,
proof of work or SMTP code of its own.

- `contact.go` only builds govue's `contact.Handler` with the site's own envelope
  (`CONTACT_FROM`, `CONTACT_TO`), subject and body.
- It sets the challenge keys (`CHALLENGE_KEY` first, `CHALLENGE_KEY_PREVIOUS` for a
  rotation), the audience (the host of `SITE_URL`) and the trusted origin of `SITE_URL`.
- It sets the site's limiters (`CHALLENGE_RATE_LIMIT_*`, `CONTACT_RATE_LIMIT_*`,
  `MAIL_RATE_LIMIT_*`).
- It mounts `GET /api/v1/contact/challenge` and `POST /api/v1/contact` behind the
  general rate limit.
- `internal/config` reads SMTP through govue's `mail.LoadConfig` (`SMTP_HOST`,
  `SMTP_PORT`, `SMTP_USERNAME` or `SMTP_USERNAME_FILE`, `SMTP_PASSWORD` or
  `SMTP_PASSWORD_FILE`; the chart mounts both credentials as files) and only when
  `SMTP_HOST` is set.
- Without a key or without `SMTP_HOST` both routes answer 503 `not_configured`.
- The handler runs with `RequireConsent`: the form sends the consent box
  (`requireConsent` in `HomeView.vue`), and a submission without a ticked box is a
  field error before the proof of work.
- The mail body is a German intro line, govue's body with its German labels
  (`contact.LabelsFor("de").Body`: name, `E-Mail:`, `Nachricht:`) and one line stating
  that the browser reported the box as ticked and the server's receive time
  (`Request.Received`), a claim of the client, not a proof.
- Every mail goes to the German owner, so it is German throughout, whatever page
  language the form sends.
- The privacy page says what the mail holds in both languages. Change it with the body.
- `contact_test.go` proves the wiring through the real handler chain with a fake
  `mail.Sender`.

## Feed route

`feed.go` is the one place `internal/feedclient` is read from. Its
`GET /api/v1/instagram` answers every cached post at once. See `internal/feedclient`
and `cmd/api` for the rest.
