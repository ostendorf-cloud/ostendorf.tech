# Legal pages review

This is an informational review, not legal advice. It does not replace a lawyer. Everything
below is marked with the source it relies on; anything not verifiable is named as such.

## Scope

Checked: `web/src/views/ImprintView.vue`, `PrivacyView.vue`, `AgbView.vue` and
`web/src/i18n/locales/{de,en}.json`, against current German law and against this repository's
code. Two generations of that code check: the first milestone, in this worktree; and, once it
existed, the second milestone's real, pushed implementation on the `feat/go-live` branch of the
main checkout (`internal/webapp`, `internal/contact`, `internal/pow`, and `govue`'s
`middleware/accesslog.go` and `session` package), read directly rather than inferred from
`old_simon-ostendorf.de`. Earlier findings based on the old repository's forward-looking code are
marked below where `feat/go-live`'s actual code turned out to differ from it.

Fixed directly, not listed here: the dispute resolution statement in the imprint (section 36
VSBG), applied in German and English.

Superseded later: the imprint no longer carries any dispute resolution section, German or
English. The owner decided that no site of the organisation states one, and the pages moved onto
govue's legal routes (`GovueLegalPage`, `language.HandleLegal`, `govue-check-legal`) without any
other change to the imprint, privacy or AGB text. The pages are now served at the exact paths
`/de/imprint`, `/en/imprint`, `/de/privacy`, `/en/privacy`, `/de/terms` and `/en/terms`; the
former AGB path `/<code>/agb/` redirects to the terms page.

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
- **General, site-wide rate limiting.** The owner asked for it to be mentioned. It is now
  described in both languages of the privacy policy, as a processing activity separate from the
  contact form's own limiting. See "Fixed: general rate limiting" below.

### VAT ID versus Kleinunternehmerregelung (resolved, with research)

The owner confirmed he holds the VAT identification number stated in the imprint and applies the
Kleinunternehmerregelung under section 19 UStG. Both at once is lawful and does not need to be
reconciled in the text:

- Holding a USt-IdNr while applying section 19 UStG is a normal, lawful combination. A
  Kleinunternehmer typically applies for one to deal with intra-Community transactions: quoting
  it to an EU supplier lets the supplier invoice net (reverse charge / "Erwerbsbesteuerung"), and
  a Kleinunternehmer who has reportable intra-Community acquisitions or reverse-charge supplies
  can have declaration duties (including a Zusammenfassende Meldung) that a USt-IdNr makes
  possible in the first place, even though no domestic VAT is charged on the business's own
  invoices (sources:
  [kleinunternehmer.de](https://www.kleinunternehmer.de/ausland.htm),
  [IHK Hannover](https://www.ihk.de/hannover/hauptnavigation/recht/steuerrecht/umsatzsteuer/kleinunternehmer-im-ust-5213920),
  [existenzgruendungsportal.de (BMWK)](https://www.existenzgruendungsportal.de/Redaktion/DE/BMWK-Infopool/Antworten/Steuern/Kleinunternehmer/Kleinunternehmer-Umgang-mit-USt-IdNr-Umsatzsteuer-Identifikationsnumme)).
- The imprint may, and under section 5 DDG must, keep stating the VAT ID. The duty to list it
  applies whenever a business holds one, "soweit vorhanden" (if one has been issued), regardless
  of Kleinunternehmer status; a VAT ID is only ever omitted when none was issued at all. Section 5
  DDG also accepts a Wirtschafts-Identifikationsnummer (W-IdNr.) as an alternative, but a trader
  who has a USt-IdNr must disclose that one (source:
  [e-recht24.de](https://www.e-recht24.de/impressum/13097-impressum-fuer-kleinunternehmer.html):
  "Führen Sie sie ab, wird Ihnen eine Umsatzsteuer-Identifikationsnummer (USt-ID) zugeteilt. Dann
  müssen Sie diese auch zwingend im Impressum angeben.").
- Repository search for a contradiction: grepped the whole worktree for VAT, price and rate
  wording ("MwSt", "Umsatzsteuer", "zzgl", "VAT", "value added tax", "€", "Preis", "Honorar",
  "Tagessatz") across `web/src`. No page states or implies a VAT amount, shows a price "plus
  statutory VAT", or otherwise contradicts the Kleinunternehmer status. The only VAT-related
  sentence anywhere is `AgbView.vue` section 4, "Gemäß § 19 UStG wird aufgrund der
  Kleinunternehmerregelung keine Umsatzsteuer erhoben" ("No value added tax is charged..." in the
  English translation), which is itself correct and is not silence: the AGB does say something
  about VAT, and what it says is right. No fix was needed and none was made.

This closes the finding. Nothing in the imprint or the AGB needs to change.

### Chamber and supervisory authority (resolved, with a source)

The owner described ordinary mandatory IHK membership that follows from registering a trade with
the tax office, not registration with a chamber for a regulated profession. That description, on
its own, does not settle whether a disclosure is owed, so this was checked against the rule
section 5 DDG actually applies:

- Section 5 DDG's duty to name a supervisory authority, and the further duty to name a chamber,
  the statutory professional title and the professional rules, attaches only to an activity that
  needs specific official government approval ("besondere behördliche Zulassung") to practise at
  all, or to a profession tied to a chamber for that reason, examples given being lawyers, estate
  agents, chimney sweeps, gastronomy, banks, insurance, security companies and auctioneers. Being
  entered in a trade register, or being an ordinary member of the IHK as every registered trader
  automatically is, does not trigger that duty by itself (source:
  [e-recht24.de](https://www.e-recht24.de/impressum/10574-impressum-aufsichtsbehoerde.html)).
- "Fachkraft für Veranstaltungstechnik" is a recognised vocational training occupation
  (Ausbildungsberuf) under the Berufsbildungsgesetz, regulated by the IHK for training purposes,
  not a regulated profession in the sense section 5 DDG means (source:
  [Wikipedia: Fachkraft für Veranstaltungstechnik](https://de.wikipedia.org/wiki/Fachkraft_f%C3%BCr_Veranstaltungstechnik)).
  The one role inside event technology that does need a specific qualification is the
  "verantwortliche Person für Veranstaltungstechnik" the Musterversammlungsstättenverordnung
  requires for building or striking a stage over 200 m², who must hold a Meister or
  Diplom-Ingenieur qualification. `AgbView.vue` section 2 already excludes exactly that role
  ("Sicherheitsbeauftragter für Veranstaltungstechnik (SiMA)... Der Auftragnehmer ist keine
  zugelassene Fachkraft für diese Bereiche") from this business's scope of services.
- Conclusion: no chamber, supervisory authority or professional title line is missing from the
  imprint. The business's actual trade (event sound, light and video services, with the
  regulated safety-officer role contracted out of scope) is not a regulated profession, so
  section 5 DDG imposes no further disclosure here.

This closes the finding. Nothing in the imprint needs to change.

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

## Fixed: general rate limiting

The owner asked for the site's general, every-request rate limiting to be described in the
privacy policy, not only the contact form's own limiting. Added as a new "Ratenbegrenzung" /
"Rate limiting" subsection in both languages of `PrivacyView.vue`, right after "Server-Log-Dateien"
/ "Server log files", describing what the code in this worktree actually does:

- `internal/webapp/webapp.go` wires one `govue/ratelimit.Limiter` (`a.rateLimit`), applied ahead
  of every page, asset, `robots.txt` and `sitemap.xml` request, keyed by the resolved client
  address (`clientip.Key`: IPv6 collapsed to its /64, IPv4 kept exact).
- What is stored: per-key, an in-memory token count and the time it was last touched
  (`govue/ratelimit/ratelimit.go`, `type bucket`). Nothing is written to disk and nothing is
  transmitted to any third party.
- For how long: a key's bucket is dropped automatically once it has refilled all the way back to
  its configured burst, which happens on its own once the key stops being used; the limiter also
  caps the number of distinct addresses it tracks at once (`maxRateLimitKeys = 100_000` in
  `webapp.go`) and evicts an arbitrary existing entry if a new address arrives while the map is
  full. There is no fixed retention period to name, because nothing survives a restart or an
  idle period.
- Legal basis: Art. 6(1)(f) GDPR, the same legitimate interest already named for server log
  files, protecting the site against overload and abuse.
- Separate processing: this general limiter is independent of the contact form's own rate
  limiting described in the "Kontaktformular" / "Contact form" section; the two do not share a
  bucket or a key space, so being rate limited on one does not count against the other.

## Fixed: the server log field list, with real data

The owner's own sample log line was from the old site and does not describe this one; verified
against `feat/go-live`'s actual code instead of inferring from `old_simon-ostendorf.de`, the
fields genuinely differ from what the ported "Server-Log-Dateien" text claimed, and in the more
privacy-friendly direction, so the old text was understating how little this site logs. Rewrote
the section in both languages of `PrivacyView.vue` to list exactly, and only, what is logged:

- Read `middleware/accesslog.go` in `govue` and `internal/webapp/webapp.go` on `feat/go-live`
  directly. `AccessLog.log` writes: the request method, `Route(r)` (the matched net/http mux
  *pattern*, never the raw path), the HTTP status, the byte count, the duration, and, because
  `internal/webapp/webapp.go` constructs `middleware.AccessLog{ClientIP: clientIPs}` without
  setting `UserAgent`, the resolved client address but never the user agent. `RequestID`
  middleware additionally attaches a per-request id to the same log line.
- The two points made explicit in both languages, as asked: the full requested address is never
  stored, only an internal category of the request; for an ordinary page view that category does
  not even distinguish which page was viewed, because `internal/webapp/webapp.go` registers the
  entire document handler on the single catch-all mux pattern `"/"` (`pages.go`'s own routing
  happens inside that one handler, never as further mux patterns) — the home page, the imprint,
  the privacy policy, the AGB and the 404 page all produce the identical logged value. The
  browser's user agent is not logged at all.
- The legal basis and retention sentences were left exactly as they were, since the 30 day
  retention is already recorded above as a standing operational requirement rather than something
  this section needed to restate.

This closes the finding.

## Closed: the stale address, checked where it actually lives now

`old_simon-ostendorf.de/internal/contact/validate.go`'s unported code was the one carrying
`info@simon-ostendorf.de`. That package is now ported on `feat/go-live` as `internal/contact`,
alongside `internal/webapp/messages.go`, which holds the German and English strings a field
error or a service outage shows a visitor. Checked both for the old domain and the old address:

- `grep -rn "simon-ostendorf\.de\|info@simon-ostendorf" internal/contact/` on `feat/go-live`
  matches only a source comment naming the old repository
  (`internal/contact/validate.go:2`), never a live address.
- Every visitor-facing message in `internal/webapp/messages.go`, in both languages, including the
  one about an email address the form cannot process and the one about the form being
  temporarily unavailable, already says `simon@ostendorf.tech`. `internal/contact/message.go`'s
  mail body also already says "ostendorf.tech", not the old domain.

The ported package is clean. Nothing needed fixing, and nothing was changed on `feat/go-live`,
which is outside this worktree.

## Fixed: a cookie the privacy policy denied exists

Reading `feat/go-live`'s real code surfaced a mismatch the ported text could not have caught,
because it goes the other way round from every other finding in this review: the privacy policy
said less processing happens than actually does, not more.

- `internal/webapp/session.go`'s `ensureSession` runs on every page request
  (`routeSession` wraps `mux.Handle("/", ...)` in `webapp.go`'s `buildHandler`), and starts a
  fresh, cookie-backed session for any visitor who does not already have one
  (`govue/session/session.go`, `Manager.Start` → `setCookie` → `http.SetCookie`). That is a real
  browser cookie (`__Host-session`, or the same name without the prefix on a plain HTTP
  development run, per `govue/session/session.go`'s `DefaultCookieName` and `New`), set for
  every visitor of every page, not only someone who uses the contact form.
- What is in it: `govue/session/session.go`'s `cookie()` sets `HttpOnly`, `Secure` (unless
  plain HTTP is explicitly allowed) and `SameSite=Lax`, with `Path=/` and no `Domain`, and no
  `Max-Age` unless an application asks for a persistent cookie, which this one does not
  (`session.Config{Store: ..., AllowPlainHTTP: cfg.AllowPlainHTTP}` in `webapp.go`, `Persistent`
  left false). The cookie's value is a random 256 bit id; the session itself (`session.Session{}`,
  empty here: no login exists on this site) lives server-side in an in-process memory store
  (`session.NewMemoryStore(nil)`). `internal/webapp/contact.go` confirms the session's only use:
  deriving the contact form's CSRF token (`csrf.Token(a.csrfKey, s.ID)`), nothing else.
- This directly contradicted the privacy policy's opening sentence and its "Keine Cookies" / "No
  cookies" section, both carried over from the old site, which claimed this website uses no
  cookies at all. Fixed in both languages of `PrivacyView.vue`: the opening summary now says the
  site sets one, purely technical cookie for the contact form; the section (renamed "Cookies" /
  "Cookies") describes what the cookie contains, that it is session-only, and its `HttpOnly`,
  `Secure` and `SameSite` attributes; and the "Kontaktformular" / "Contact form" section's
  existing sentence about the proof-of-work step itself setting no cookie was kept (it is still
  true, the PoW mechanism itself sets nothing) and given a cross-reference to the session cookie
  that already exists by the time a visitor reaches the form.
- What was not invented: whether this cookie actually falls under the "technically necessary"
  exemption from consent (Section 25(2) no. 2 TDDDG) is a genuinely contested point in German
  commentary specifically for CSRF cookies, not a settled one. Some sources treat a security
  cookie with no tracking purpose as covered by the exemption the same way a login or shopping
  cart session is; others argue a CSRF cookie is a security enhancement rather than something the
  requested service cannot function without, and would need consent (source:
  [dr-dsgvo.de](https://dr-dsgvo.de/csrf-cookies-zur-gefahrenabwehr-bedeutung-und-problem-mit-dem-datenschutz/)).
  The fixed text states the "technically necessary, no banner" position plainly, as "nach unserer
  Einschätzung" / "in our assessment", which is the mainstream, commonly applied view for a
  session cookie used solely for security with no tracking purpose, and is also simply the
  operator's own classification to make; it is not asserted here as a settled point of law.

This is now a finding, not a question answered by itself:

1. **Confirm the "technically necessary, no cookie banner" classification for the session
   cookie.** Why it matters: the fixed text takes a defensible, mainstream position on a
   genuinely contested question, in the text on the owner's behalf. Source:
   [dr-dsgvo.de](https://dr-dsgvo.de/csrf-cookies-zur-gefahrenabwehr-bedeutung-und-problem-mit-dem-datenschutz/)
   and [cortina-consult.com](https://cortina-consult.com/web-compliance/wissen/tdddg/) on the
   Section 25(2) TDDDG exemption generally. Owner should have this one specific point confirmed,
   for instance by whoever already helped with the Kleinunternehmer and chamber questions, before
   the contact form milestone goes live; if the conclusion changes, both the German and the
   English "Cookies" section need a cookie banner added, not just different wording.

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

What follows was first checked against `old_simon-ostendorf.de`'s forward-looking implementation,
then re-checked directly against `feat/go-live`'s real, pushed code once it existed. One of the
four points changed as a result, the cookie point; the other three held up unchanged:

- The proof-of-work challenge itself is cookie-free: `internal/pow/pow.go` on `feat/go-live` is
  the same algorithm as `old_simon-ostendorf.de`'s, unchanged, keeping state only in an in-memory
  map on the server (`Service.used`) and in the challenge/salt/signature values the browser
  already receives and returns with the form. This specific mechanism still writes nothing to
  the visitor's browser; see "Fixed: a cookie the privacy policy denied exists" above for the
  separate, sitewide session cookie that exists alongside it.
- The Umami hookup is not yet part of this codebase on either branch (the milestone brief lists
  it as a later milestone); `internal/analytics/analytics.go` only exists in
  `old_simon-ostendorf.de`, so the privacy text's Umami section remains forward-looking and
  unverified against real code, consistent with the standing operational requirement above.
  Update: the hookup has since landed as govue's `analytics` package (issue #87, which
  removed the site's own `internal/analytics`), and the Umami section's description of what
  is counted was changed to match it: route pattern, referrer origin and address shortened
  to /24 or /48 instead of the path, the referrer without parameters and the full address.
- The Instagram feed likewise does not exist on `feat/go-live` yet (also a later milestone); the
  privacy text's Instagram section remains forward-looking, based on `old_simon-ostendorf.de`'s
  `internal/instagram/client.go` and `feed.go`, and needs re-checking once that code lands.
- The site is not cookie-free. `feat/go-live` sets one real cookie, on every page view, for the
  reason and with the attributes described above; the privacy policy no longer claims otherwise.
