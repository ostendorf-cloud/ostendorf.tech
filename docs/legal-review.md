# Legal pages review

This is an informational review, not legal advice. It does not replace a lawyer. Everything
below is marked with the source it relies on; anything not verifiable is named as such.

## Scope

Checked: `web/src/views/ImprintView.vue`, `PrivacyView.vue`, `AgbView.vue` and
`web/src/i18n/locales/{de,en}.json`, against current German law and against this repository's
code (first milestone, this worktree) plus the second milestone's not-yet-ported code in
`old_simon-ostendorf.de` (`internal/pow`, `internal/analytics`, `internal/contact`,
`internal/instagram`, `internal/clientip`, `internal/ratelimit`).

Fixed directly, not listed here: the dispute resolution statement in the imprint (section 36
VSBG), applied in German and English.

Confirmed and left unchanged, because nothing was wrong:

- Neither page cites section 5 TMG or section 5 DDG, so there is no statute reference to update.
  Section 5 DDG replaced section 5 TMG on 14 May 2024 with no change to the mandatory content,
  only "Telemediendienst" became "digitaler Dienst" (sources:
  [IHK Osnabrück](https://www.ihk.de/osnabrueck/recht-und-fair-play/aktuelles/ddg-ersetzt-das-tmg-6220008),
  [IHK Nordschwarzwald](https://www.ihk.de/nordschwarzwald/recht/recht/handels-und-wettbewerbsrecht/telemediengesetz-tmg-ist-nun-digitale-dienste-gesetz-ddg--6171272)).
- Neither page links the EU ODR platform. It was shut down on 20 July 2025 after Regulation (EU)
  2024/3228 repealed the ODR Regulation; a surviving link would now itself be a problem under
  section 5 UWG (misleading commercial practice), not a missing one (sources:
  [nbs-partners](https://www.nbs-partners.de/neuigkeiten/einstellung-der-eu-online-streitbeilegungsplattform-zum-20-juli-2025-was-unternehmen-jetzt-wissen-und-tun-sollten/),
  [lexallawconsultings](https://lexallawconsultings.de/stichtag-27-juli-2025-eu-stoppt-odr-plattform-und-beendet-eine-umstrittene-pflicht/)).
- The privacy policy already cites section 25(1) TDDDG, not the old TTDSG name; that renaming
  happened in the same May 2024 reform, so no change was needed.
- The contact address, phone number and postal address are identical across the imprint, the
  privacy policy and the home page (`simon@ostendorf.tech`, `+49 1573 0971784`, Frankenstr. 8,
  52070 Aachen), and the old domain and the old `info@simon-ostendorf.de` address do not appear
  anywhere in `web/src/i18n/locales/` or `web/src/views/`, only in three source comments naming
  the old repository. The first milestone already carried out this swap correctly.

## Resolved by the owner

- **Hosting and the Hetzner backup address.** The owner confirmed that the hosting section's "in
  unseren eigenen Räumlichkeiten" (our own premises) statement and the Hetzner Online GmbH backup
  address for Umami and the mail server are correct as they stand. No change needed in either
  page. This closes the two earlier findings about hosting location and the backup address; the
  owner owes nothing further here.
- **English privacy policy and AGB.** Both now carry a full English translation, replacing the
  earlier stub that only pointed back to the German version. See "Translated pages" below.

## Standing operational requirement, not a question

**The 30-day server log retention and the 14-month Umami retention are promises the privacy
policy makes to visitors, and nothing in either repository's code enforces them.**
The owner has confirmed both periods are correct as a matter of policy. That settles what the
text should say; it does not settle whether the infrastructure actually does it. A privacy
policy stating a retention period the operator does not actually enforce is a false statement to
a visitor, not a harmless gap, so this stays in the document as a constraint rather than a
question:

- The log pipeline (whatever sits in front of or inside the deployed application once it exists)
  must delete server logs at 30 days at the latest, matching the "Server-Log-Dateien" / "Server
  log files" section of `PrivacyView.vue` in both languages.
- The Umami instance must be configured to delete statistics data after 14 months, matching the
  "Besucherstatistik mit Umami" / "Visitor statistics with Umami" section in both languages.
- Source: repository search for "retention", "30 Tage" and "logrotate" across this worktree and
  `old_simon-ostendorf.de` found no log pipeline configuration or Umami `DATA_RETENTION` setting;
  these periods are an operational property of the cluster the site runs on, not of this
  repository's code.
- Whoever configures the deployed logging pipeline and the Umami instance must set both
  retention periods to match these numbers, and keep them matching if either is ever changed:
  changing one without the other turns a true statement into a false one.

## Findings for the owner to decide or supply

1. **VAT ID versus Kleinunternehmerregelung is not a contradiction by itself, but is unverified.**
   Why it matters: the AGB state no VAT is charged under section 19 UStG (Kleinunternehmer),
   while the imprint lists a VAT identification number under section 27a UStG. A business can
   legitimately hold both, but this review cannot confirm current turnover or Kleinunternehmer
   status.
   Source: `AgbView.vue` section 4; `ImprintView.vue` VAT ID section.
   Owner must confirm: that the Kleinunternehmer status in the AGB still holds, and that the VAT
   ID is still correct and still needed, so both pages keep agreeing with each other.

2. **No chamber, supervisory authority or professional title appears for the trade, and this
   review cannot certify that none applies.**
   Why it matters: section 5 DDG requires naming a supervisory authority, chamber and
   professional title for a regulated profession or a licensed craft (Anlage A Handwerksordnung).
   Research found no indication that event sound, light and video services are listed among the
   licensed crafts, but this was not checked against an authoritative, current Anlage A listing
   specific to this business's trade registration.
   Source: general guidance on section 5 DDG mandatory content for licensed trades; HWK Berlin's
   published Anlage A list did not surface the trade in search results, which is not the same as
   confirming its absence from the current list.
   Owner must confirm: whether the business is registered as a Gewerbe (IHK) or as an entry in
   the Handwerksrolle (HWK), to settle whether a chamber or supervisory authority line is
   missing.

3. **The whole site's general rate limiting is not mentioned in the privacy policy text as
   ported.**
   Why it matters: `internal/webapp` (this worktree) wires govue's rate limiter keyed by client
   address for every page request, not only for the contact form. The privacy policy's existing
   wording only describes IP-based rate limiting for the contact form's abuse protection.
   Source: `internal/webapp/webapp.go` (`a.rateLimit`, wired via `clientip.NewResolver`); the
   ported privacy text's "Kontaktformular" section is the only place rate limiting by IP is
   mentioned.
   Owner may want: a sentence in the "Server-Log-Dateien" or hosting section naming that every
   request, not only the contact form, is rate limited by a short-lived in-memory counter keyed
   to the client address. Low priority, since the existing legitimate-interest basis already
   covers it in substance.

4. **The access log's actual fields are narrower than the "Server-Log-Dateien" list describes,
   and the gap sits at the reverse proxy, which this repository does not contain yet.**
   Why it matters: `govue`'s `middleware.AccessLog`, as wired in `internal/webapp/webapp.go`,
   logs method, matched route, status, byte count, duration and the resolved client address; it
   does not log the user agent or the referrer by default, and this application does not turn
   that on. The ported privacy text's list (IP, date/time, request line, status, bytes, browser
   and OS, referrer) describes a conventional web or reverse-proxy log, which is plausible for
   whatever sits in front of this application once deployed, but that component is not part of
   this repository yet.
   Source: `internal/webapp/webapp.go`; `govue/middleware/accesslog.go` (`AccessLog.log`, no
   `UserAgent` field set here, so it defaults to omitted).
   Owner must confirm: the actual log fields and retention of the edge or ingress component once
   milestone two's Helm chart exists, so the "Server-Log-Dateien" section matches what is really
   logged, not only what the Go application itself logs.

5. **A stale address lives in code that has not been ported yet, not in a legal page.**
   Why it matters: not a defect in the current pages, but worth catching before milestone two
   lands. `old_simon-ostendorf.de/internal/contact/validate.go` has a validation message that
   tells a visitor with a non-ASCII email address to write to `info@simon-ostendorf.de` directly.
   Source: `old_simon-ostendorf.de/internal/contact/validate.go`, the "E-Mail-Adressen mit
   Umlauten..." error message.
   Owner, or whoever implements the contact form milestone, should use `simon@ostendorf.tech`
   there instead when that code is ported.

## Translated pages

`PrivacyView.vue` and `AgbView.vue` now carry a full English translation of the German privacy
policy and the German AGB, in the same place the imprint already did this: a binding-notice
sentence at the top of the English block, reusing the existing `legal.bindingNotice` /
`legal.viewGerman` catalogue strings, followed by the full translated content under the same
headings. The German text stays the source of truth; nothing was added, removed or softened in
translation, and every German statutory reference (GDPR articles, Section 25(1) TDDDG, Section
19 and Section 27a UStG, DGUV, MVStättVO) keeps its German name and section number, with a short
English gloss on first use rather than a substituted domestic equivalent.

One place in the AGB left room for two readings, and the more conservative one was used rather
than picked silently: Section 5's "Der Auftragnehmer ist verpflichtet, anderweitig erzielte
Einnahmen für den abgesagten Zeitraum anzurechnen" was translated as a duty to credit substitute
income earned elsewhere against the cancellation fee, the plain reading of "anrechnen", rather
than as a cap on or a waiver of the fee. If the owner means something stronger (for example, the
fee reduced to the net loss actually suffered), that changes the German text too, not only the
translation.

## Confirmed by reading the code, not assumed

- The proof-of-work challenge is cookie-free as advertised: `internal/pow/pow.go` keeps state
  only in an in-memory map on the server (`Service.used`) and in the challenge/salt/signature
  values the browser already receives and returns with the form; nothing is written to the
  visitor's browser.
- Umami analytics run entirely server to server: `internal/analytics/analytics.go` builds the
  view from the request the server already has and posts it to Umami's `/api/send`; no script
  reaches the browser and no cookie is set.
- The Instagram feed is proxied: `internal/instagram/client.go` and `feed.go` fetch posts and
  images server-side and cache them in memory; a visitor's browser never contacts Meta unless
  they click through to a permalink, and Instagram's API receives only the server's own address
  and access token, never visitor data.
- No `Cookie` or `Set-Cookie` handling exists anywhere in this worktree's `internal/` or `cmd/`
  packages, matching the privacy policy's cookie-free claim for the current milestone.

These four points mean the planned second milestone (contact form, Instagram feed, Umami) is
consistent with what the privacy policy already says, provided it ports the same architecture
unchanged. Re-check this once that code actually lands in this repository rather than relying on
this review of the old repository's implementation.
