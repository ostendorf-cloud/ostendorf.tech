/**
 * The contact channels shown on the home page's Kontakt section and in the
 * footer's social row. The i18n catalogues hold the label and the display
 * value under home.contact.channels.<key>; this file holds everything that
 * does not translate: the icon and the href.
 */
export type ChannelKey = 'email' | 'phone' | 'whatsapp' | 'instagram' | 'facebook'

export interface Channel {
  key: ChannelKey
  href: string
  /** Opens in a new tab: true for a third party site, false for mailto/tel. */
  external: boolean
}

export const channels: Channel[] = [
  { key: 'email', href: 'mailto:simon@ostendorf.tech', external: false },
  { key: 'phone', href: 'tel:+4915730971784', external: false },
  { key: 'whatsapp', href: 'https://wa.me/4915730971784', external: true },
  { key: 'instagram', href: 'https://instagram.com/simonostendorf.vtech', external: true },
  { key: 'facebook', href: 'https://facebook.com/simonostendorf.vtech/', external: true },
]
