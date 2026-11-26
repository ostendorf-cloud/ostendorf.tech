# Generated assets

Every file below is generated and never hand edited. Commands that need ImageMagick use
the version from `mise.toml`.

## Logo wordmark

`web/src/assets/logo-wordmark-light.svg` and `logo-wordmark-dark.svg` are generated from
`Logo_Name.pdf` in the organisation's `old_logo` repository.

The source file bakes in two colours (white for "Simon Ostendorf", a mid blue for
"Veranstaltungstechnik"), meant for the dark stage background the old site always
rendered on. Both are overridden to one flat colour per variant, navy (`#0f1a2e`, this
theme's own `--color-text`) for light mode and white for dark mode, matching the two
variants the design calls for. Moving that token means regenerating the light variant: it
carries the value, not a reference to it. The command, repeated once per variant:

```sh
pdftocairo -svg Logo_Name.pdf logo-wordmark-light.svg
# then replace every fill="rgb(...)" with fill="#0f1a2e" (light) or fill="#ffffff" (dark)
svgo --multipass -i logo-wordmark-light.svg -o logo-wordmark-light.svg
```

## Hero

`web/src/assets/hero-*.{avif,webp,jpg}` and `portrait-*.{avif,webp,jpg}` are generated
from `hero.jpg` (2400x1800) and `portrait.jpg` (1200x1200) in old_simon-ostendorf.de's
`cmd/website/static/img`, never re-cropped. Both are the owner's own photographs.

The build pipeline does not resize or re-encode anything, on purpose: these two pictures
change about as often as the logo does, so paying for an image pipeline on every
`task web:build` would buy nothing. `web/src/lib/heroImage.ts` is the one place the
variants are listed, with the width of each next to its import, and Vite content hashes
them from there. The commands:

```sh
magick hero.jpg -resize 1280x -strip -quality 50 web/src/assets/hero-1280.avif
magick hero.jpg -resize 1280x -strip -quality 72 -define webp:method=6 web/src/assets/hero-1280.webp
```

Repeat for 640, 960, 1280, 1600, 1920 and 2400. The hero's JPEG fallback is the 1280
width at `-quality 78 -sampling-factor 4:2:0 -interlace JPEG`.

## Portrait

The portrait is the exception: it is shown large and close, so it keeps higher quality
and the original's own width. Its source is `portrait.jpg` of the old site
(`https://www.simon-ostendorf.de/static/img/portrait.jpg`, 1200x1200, 133 KB, the
owner's own photograph). It is not committed: put it at
`web/src/assets/portrait-source.jpg`, where `.gitignore` keeps it out. Every variant is
made from that file and none is scaled up. The commands, run in `web/src/assets/`, for W
in 460, 920 and 1200:

```sh
magick portrait-source.jpg -resize Wx -strip -quality 80 portrait-W.avif
magick portrait-source.jpg -resize Wx -strip -quality 90 -define webp:method=6 portrait-W.webp
magick portrait-source.jpg -resize Wx -strip -quality 85 -sampling-factor 4:2:0 \
  -interlace JPEG portrait-W.jpg
```

The JPEG commands run for 460 and 920 only: `portrait-1200.jpg` is the source file
copied unchanged, so the widest fallback loses nothing. Each file stays under 150 KB.
`web/src/lib/heroImage.ts` lists all nine with their widths, and
`web/src/lib/portraitImage.test.ts` reads every file's header and fails when a declared
width differs from the file's pixels.

## Favicon and social image

`web/public/favicon.svg` is the one hand kept source of the icon: the earlier site's
32 by 32 fader mark on a dark rounded square. `web/public/favicon-32x32.png`,
`apple-touch-icon.png` (180x180, opaque, the square's own dark colour fills the corners)
and `og-image.png` (1200x630, the mark and the white wordmark on that same colour) are
generated from it and from `web/src/assets/logo-wordmark-dark.svg`.

- The file names are govue's favicon contract (its Go package `favicon`), plus
  `og-image.png`.
- Vite copies `web/public` to the root of `dist/`.
- govue's frontend serves the three icon files and `og-image.png` (a `PublicFiles` entry)
  from there, refuses to start without one, and injects the icon links into every page
  itself. `web/src/lib/icons.ts` writes only the social preview image's tags.
- `cmd/frontend/build_test.go` runs govue's `favicon.CheckPages` over the pages as the
  server sends them.

Run again whenever the SVG or the wordmark changes:

```sh
task favicon:generate
```

The icon is a dark square with a bright gradient mark, so it reads on a light and on a
dark tab bar without a colour scheme switch. The SVG carries literal colours because a
favicon cannot read the page's design tokens.

The fader mark alone (the icon, `LogoMark.vue`) is not generated. It is inline Vue markup
ported from old_simon-ostendorf.de's `{{define "logo"}}` template block, with a gradient
whose stops read the theme's `--logo-from` and `--logo-to` tokens: the favicon's
`#0ff0ff` and `#803cff`, the same in light and dark mode and independent of the palette's
accent roles, so it never needs regenerating when the theme changes.

## Band pictures

`web/src/assets/bands/herrenjedeck-logo.webp` is generated from the band's own logo,
`https://www.herrenjedeck.de/assets/logo-dark.png` (1182x945, transparent background).
The source is not committed: download it to
`web/src/assets/bands/herrenjedeck-logo-dark.png`, which `.gitignore` keeps out. The
command trims the transparent margin and scales to 480 px wide (480x489, about 28 KB):

```sh
magick herrenjedeck-logo-dark.png -trim +repage -resize 480x -strip \
  -define webp:method=6 -define webp:alpha-quality=90 -quality 80 herrenjedeck-logo.webp
```

The mark is a dark badge on a transparent ground, so `BandLogo.vue` sets it on a plate of
`--color-accent-on`, which is near white in both modes. Further band logos follow the same
pattern; see the comment in `web/src/data/bands.ts`.

`web/src/assets/bands/bademeister.webp` and `rubik.webp` are generated from the bands' own
pictures: `bademeister-source.jpg` (the band's profile picture, 1080x1080) and
`rubik-source.jpg` (the band's photo, 315x315). The sources are not committed: put them in
`web/src/assets/bands/`, where `.gitignore` keeps `*-source.jpg` out. Rubik's file is not
scaled up, so it stays 315x315 and looks soft on high density screens. Both stay under
60 KB. The commands:

```sh
magick bademeister-source.jpg -resize 640x640 -strip -define webp:method=6 -quality 80 bademeister.webp
magick rubik-source.jpg -strip -define webp:method=6 -quality 85 rubik.webp
```

They are square pictures without transparency, so `BandPhoto.vue` shows them as rounded
squares with no plate, through the `photo` field in `web/src/data/bands.ts`.
