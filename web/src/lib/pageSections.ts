import { ref } from 'vue'

/** The id of the Impressionen section, which the header's entry links to. */
export const IMPRESSIONS_ID = 'impressions'

/**
 * The widths, from Tailwind's sm (40rem) up to just below lg (64rem), where
 * the header shows its navigation as one row that has no room for a fifth
 * link in German (about 909 px needed). The Impressionen entry is left out of
 * the row there; below sm it is in govue's menu, from lg in the row.
 */
export const IMPRESSIONS_HIDDEN_QUERY = '(min-width: 40rem) and (max-width: 63.9375rem)'

/**
 * Whether the Impressionen section is on the page right now.
 *
 * The section renders itself in the browser once the feed has posts, so the
 * prerendered header cannot know about it. InstagramFeed sets this once its
 * section exists and clears it when the section goes away, and AppHeader
 * adds its entry only while it is set. It is never set on the server, so
 * the prerendered HTML carries no entry.
 */
export const impressionsOnPage = ref(false)
