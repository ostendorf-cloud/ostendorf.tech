/**
 * Which theme this site ships and which mode it opens in.
 *
 * Both are read from here by the two sides that have to agree on them:
 * src/main.ts hands them to useTheme, and web/scripts/prerender.ts writes
 * the same two values into the data-theme and data-mode attributes of
 * every page it renders, and picks the browser chrome colour by the mode
 * named here. A prerendered page therefore paints in the mode it will
 * still be in once the bundle has loaded, rather than in one the
 * composable then changes under the visitor.
 *
 * That last part is why the mode is pinned in the markup at all. govue's
 * useTheme treats the absence of data-mode as its third state, "follow
 * the operating system", so a prerendered page that left the attribute
 * off would paint light for a visitor whose system is light and then go
 * dark the moment the bundle ran.
 */
import type { ColorMode } from '@ostendorf-cloud/govue'

/** The theme never varies: this site ships exactly one brand. */
export const THEME_NAME = 'vtech'

/**
 * Dark, not the visitor's system setting. This is event production work
 * and the stage ground is the brand, so dark is the site rather than an
 * accommodation; light mode is there for the people who want it. The
 * toggle still offers all three of useTheme's states and still remembers
 * the choice, so a visitor who picks light or system keeps it.
 *
 * Typed as the two modes that name a real palette rather than as the full
 * ColorMode: "system" is not something prerendered markup can resolve,
 * and the chrome colour is looked up by this value.
 */
export const DEFAULT_MODE: Extract<ColorMode, 'light' | 'dark'> = 'dark'

/**
 * Where the visitor's theme and mode choice is stored. govue's useTheme
 * defaults to this exact same key when no storageKey option is given, but
 * that default is not exported, so three places need to agree on it
 * without being able to import it from govue: this file, main.ts's
 * useTheme call, and the prerendered <head>'s own blocking boot script
 * (web/scripts/prerender.ts), which has to read the stored choice the
 * same way useTheme does, not a second guess at the key. Pinning the
 * value here, explicitly, rather than leaving it to govue's unexported
 * default, is what keeps those three in agreement on purpose rather than
 * by coincidence.
 */
export const STORAGE_KEY = 'govue-theme'
