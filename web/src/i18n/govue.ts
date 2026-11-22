import { createGovueI18n } from '@ostendorf-cloud/govue'

/**
 * The site's German texts for govue's contact form. govue's German catalogue
 * addresses the visitor with "du"; this site says "Sie" everywhere, so every
 * contact string that addresses the visitor is replaced here, with the same
 * meaning and the same placeholders as govue's text. The site's privacy text
 * speaks as "wir", so privacyIntro does too.
 *
 * govueI18n.test.ts proves every key exists in govue's German catalogue and
 * that every other contact key is listed in NEUTRAL_CONTACT_KEYS, so a new or
 * renamed govue string fails a test instead of showing "du" on the site.
 */
export const GOVUE_CONTACT_DE = {
  privacyIntro: 'So gehen wir mit Ihren Angaben um:',
  sending: 'Ihre Nachricht wird gesendet',
  success: 'Danke. Ihre Nachricht wurde gesendet.',
  error: {
    required: 'Bitte füllen Sie dieses Feld aus.',
    nonAsciiEmail:
      'Bitte nutzen Sie eine E-Mail-Adresse ohne Umlaute und andere Sonderzeichen, etwa ä, ö, ü, ß oder é.',
    consentRequired: 'Bitte stimmen Sie zu, bevor Sie senden.',
    invalidRequest:
      'Das Formular konnte nicht angenommen werden. Bitte prüfen Sie Ihre Eingaben und versuchen Sie es erneut.',
    notConfigured:
      'Das Kontaktformular ist noch nicht eingerichtet. Bitte versuchen Sie es später erneut.',
    powRequired:
      'Die Spam-Prüfung fehlt. Es wurde nichts gesendet. Bitte senden Sie das Formular erneut.',
    powInvalid:
      'Die Spam-Prüfung ist fehlgeschlagen. Es wurde nichts gesendet. Bitte senden Sie das Formular erneut.',
    powExpired:
      'Die Spam-Prüfung ist abgelaufen. Es wurde nichts gesendet. Bitte senden Sie das Formular erneut.',
    powTooFast:
      'Das Formular wurde zu schnell gesendet. Bitte warten Sie einen Moment und senden Sie es erneut.',
    powReplay:
      'Die Spam-Prüfung wurde schon verwendet. Es wurde nichts gesendet. Bitte senden Sie das Formular erneut.',
    powBusy:
      'Der Server ist gerade ausgelastet. Es wurde nichts gesendet. Bitte versuchen Sie es gleich erneut.',
    powBusyWait:
      'Der Server ist gerade ausgelastet. Es wurde nichts gesendet. Bitte versuchen Sie es {wait} erneut.',
    mailLimit:
      'Das Formular hat sein Limit an Nachrichten vorerst erreicht. Es wurde nichts gesendet. Bitte versuchen Sie es später erneut.',
    mailLimitWait:
      'Das Formular hat sein Limit an Nachrichten vorerst erreicht. Es wurde nichts gesendet. Bitte versuchen Sie es {wait} erneut.',
    sendFailed:
      'Ihre Nachricht wurde möglicherweise nicht gesendet. Bitte prüfen Sie das, bevor Sie sie erneut senden, oder schreiben Sie auf einem anderen Weg.',
    rateLimited:
      'Zu viele Nachrichten. Bitte warten Sie ein paar Minuten und versuchen Sie es erneut.',
    rateLimitedWait: 'Zu viele Nachrichten. Bitte versuchen Sie es {wait} erneut.',
    refused:
      'Die Anfrage wurde abgelehnt. Bitte laden Sie die Seite neu und versuchen Sie es erneut.',
    solver:
      'Die Spam-Prüfung konnte in diesem Browser nicht laufen. Es wurde nichts gesendet. Bitte laden Sie die Seite neu oder nutzen Sie einen anderen Browser.',
    network:
      'Keine Verbindung zum Server. Bitte prüfen Sie Ihr Netzwerk und versuchen Sie es erneut.',
    generic: 'Etwas ist schiefgegangen. Bitte versuchen Sie es erneut.',
  },
}

/**
 * The contact keys whose German govue text addresses nobody and stays as
 * govue ships it. Reviewed by hand: a key govue adds later is in neither list
 * and fails govueI18n.test.ts until it is sorted into one of the two.
 */
export const NEUTRAL_CONTACT_KEYS = [
  'title',
  'field.name',
  'field.email',
  'field.message',
  'field.honeypot',
  'wait.minute',
  'wait.minutes',
  'wait.hour',
  'wait.hours',
  'privacy',
  'submit',
  'another',
  'failure',
  'unavailable',
  'error.tooLong',
  'error.invalidChars',
  'error.invalid',
  'error.tooLarge',
]

/**
 * createSiteGovueI18n is govue's language layer with the site's German
 * contact texts merged over govue's own. main.ts, the prerender script and
 * the tests install it, so the browser and the prerendered HTML say the same.
 */
export function createSiteGovueI18n(options: Parameters<typeof createGovueI18n>[0]) {
  return createGovueI18n({
    ...options,
    messages: { de: { govue: { contact: GOVUE_CONTACT_DE } } },
  })
}
