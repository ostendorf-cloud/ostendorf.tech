# Scope of the first milestone

The home page's hero, Leistungen, Impressionen, Bands, Über mich and Kontakt sections,
the imprint, privacy and AGB pages, crawling (`robots.txt`, `sitemap.xml`), the Instagram
feed (split across the API and the feed service, see `internal/instagram/AGENTS.md`),
the Umami hookup, the three container images and the Helm chart are implemented. Left
out, tracked as later milestones:

- The Impressionen section is rendered by the browser after the page loads, from
  `GET /api/v1/instagram`, not baked into the prerendered HTML. The feed changes while
  the binary does not, and `task prerender` runs at build time with no access token. A
  crawler that executes no JavaScript therefore does not see the section. That is
  deliberate, since the section carries no content worth indexing.
- The nav's Impressionen entry follows the section the same way. The prerendered nav has
  none, because it cannot know whether the section will be there. `InstagramFeed.vue`
  sets `impressionsOnPage` (`web/src/lib/pageSections.ts`) once its section is rendered,
  which makes `AppHeader.vue` add the entry between Leistungen and Bands.
- The entry shifts the links after it sideways when it arrives. The header's height does
  not change.
- It is left out of the row between 640 and 1023 px (`IMPRESSIONS_HIDDEN_QUERY`, the
  `sm` to below `lg` range), where the one row has no room for a fifth German link. It is
  in govue's menu below 640 px and in the row from 1024 px.
