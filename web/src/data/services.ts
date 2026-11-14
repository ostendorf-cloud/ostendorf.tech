/**
 * The icon each Leistungen entry carries, in the order the i18n
 * catalogues list home.services.items. The catalogues hold the words, this
 * file holds the one thing about an entry that does not translate.
 *
 * A shorter list than the catalogue's would silently leave the last
 * entries without an icon, so HomeView.vue pairs the two by index and
 * src/data/services.test.ts holds the two lengths together.
 */
export type ServiceIconName = 'faders' | 'microphone' | 'speaker' | 'movingHead' | 'camera' | 'case'

export const serviceIcons: ServiceIconName[] = [
  'faders',
  'microphone',
  'speaker',
  'movingHead',
  'camera',
  'case',
]
