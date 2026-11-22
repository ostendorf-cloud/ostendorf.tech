import { ref } from 'vue'

/** The id of the Impressionen section, which the header's entry links to. */
export const IMPRESSIONS_ID = 'impressions'

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
