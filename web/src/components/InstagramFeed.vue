<script setup lang="ts">
/**
 * The home page's Impressionen section: the latest posts of the business
 * Instagram account.
 *
 * Nothing here talks to Meta. The Go service fetched these posts and their
 * pictures server side (internal/instagram) and serves both from this
 * site's own origin, which is what the privacy policy promises: "Ihr
 * Browser baut dabei keine Verbindung zu Servern von Instagram oder Meta
 * auf". The only requests this component causes are to /api/v1/instagram
 * and /api/v1/instagram/{id}, both on this very domain.
 *
 * The section renders only once there is something to show. A deployment
 * with no access token configured, a feed that has not loaded yet and a
 * request that failed all end the same way: no section at all, and nothing
 * else on the page affected.
 */
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { createClient } from '@ostendorf-cloud/govue'

import { IMPRESSIONS_ID, impressionsOnPage } from '../lib/pageSections'
import ChannelIcon from './ChannelIcon.vue'

/** The account the "follow" button links to, the one the feed shows. */
const PROFILE_URL = 'https://www.instagram.com/simonostendorf.vtech/'

export interface InstagramPost {
  id: string
  alt: string
  permalink: string
}

interface InstagramFeedResponse {
  posts: InstagramPost[]
}

const props = withDefaults(
  defineProps<{
    /** Replaced in tests; defaults to the real API client. */
    client?: Pick<ReturnType<typeof createClient>, 'get'>
  }>(),
  { client: undefined },
)

const { t } = useI18n()

const posts = ref<InstagramPost[]>([])

/** imageUrl is this service's own copy of the picture, never Meta's CDN. */
function imageUrl(post: InstagramPost): string {
  return `/api/v1/instagram/${encodeURIComponent(post.id)}`
}

// The header's Impressionen entry follows the section: it is there while the
// section is rendered, and only then. flush 'post' runs after the DOM update,
// so the entry never points at an id that is not on the page yet.
watch(
  () => posts.value.length > 0,
  (present) => {
    impressionsOnPage.value = present
  },
  { flush: 'post' },
)
onBeforeUnmount(() => {
  impressionsOnPage.value = false
})

onMounted(async () => {
  const client = props.client ?? createClient()
  try {
    const feed = await client.get<InstagramFeedResponse>('/instagram')
    posts.value = feed?.posts ?? []
  } catch {
    // A feed that cannot be loaded is a section that is not shown. It is
    // never an error a visitor is told about, and never anything the rest
    // of the page notices.
    posts.value = []
  }
})
</script>

<template>
  <section
    v-if="posts.length > 0"
    :id="IMPRESSIONS_ID"
    aria-labelledby="impressions-title"
    class="py-8 sm:py-14"
  >
    <header class="mb-6 flex flex-col gap-2">
      <p class="text-sm font-semibold tracking-wide text-text-muted uppercase">
        {{ t('home.impressions.eyebrow') }}
      </p>
      <h2 id="impressions-title" class="text-2xl font-semibold text-text">
        {{ t('home.impressions.title') }}
      </h2>
      <p class="text-lg text-text-muted">{{ t('home.impressions.lead') }}</p>
    </header>

    <ul class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4" role="list">
      <li v-for="post in posts" :key="post.id">
        <a :href="post.permalink" target="_blank" rel="noopener noreferrer" class="insta-tile">
          <img
            :src="imageUrl(post)"
            :alt="post.alt"
            width="540"
            height="540"
            loading="lazy"
            decoding="async"
            class="aspect-square w-full object-cover"
          />
          <!-- Decorative: the link is named by the picture's alt text. -->
          <span class="insta-overlay" aria-hidden="true">
            <ChannelIcon name="instagram" />
          </span>
        </a>
      </li>
    </ul>

    <p class="mt-6">
      <a
        :href="PROFILE_URL"
        target="_blank"
        rel="noopener noreferrer"
        class="inline-flex min-h-11 items-center justify-center gap-2 rounded-md border border-border-strong bg-surface px-4 text-base font-medium text-text transition-colors hover:bg-surface-hover"
      >
        {{ t('home.impressions.follow') }}
      </a>
    </p>
  </section>
</template>

<style scoped>
/*
 * A tile as on the old site: on hover or keyboard focus the picture grows
 * to 1.06 over 600 ms while the brand gradient fades in over it, 250 ms,
 * with the Instagram glyph in the middle. The overlay's two colours are
 * --ig-overlay-from and --ig-overlay-to from the theme, not values of
 * their own.
 */
.insta-tile {
  position: relative;
  display: block;
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: 14px;
  background: var(--color-surface);
}

.insta-tile img {
  transition: transform 600ms var(--ease-out);
}

.insta-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: linear-gradient(135deg, var(--ig-overlay-from), var(--ig-overlay-to));
  color: var(--color-accent-on);
  opacity: 0;
  pointer-events: none;
  transition: opacity 250ms ease;
}

.insta-overlay svg {
  width: 34px;
  height: 34px;
}

.insta-tile:focus-visible img {
  transform: scale(1.06);
}

.insta-tile:focus-visible .insta-overlay {
  opacity: 1;
}

/* Hover only where the pointer can really hover: on a touch screen a tap
   would otherwise leave the tile stuck in its hovered state. */
@media (hover: hover) {
  .insta-tile:hover img {
    transform: scale(1.06);
  }

  .insta-tile:hover .insta-overlay {
    opacity: 1;
  }
}

/* No scale and no fade: the overlay simply appears. */
@media (prefers-reduced-motion: reduce) {
  .insta-tile img,
  .insta-overlay {
    transition: none;
  }

  .insta-tile:hover img,
  .insta-tile:focus-visible img {
    transform: none;
  }
  /* (the hover rules above sit in a media query of their own, so this
     reduced motion rule, which comes later with equal specificity, wins) */
}
</style>
