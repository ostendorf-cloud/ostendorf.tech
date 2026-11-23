/** escapeAttribute writes value for use inside a double or single quoted
 *  HTML attribute. */
export function escapeAttribute(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

/** THEME_BOOT_TEXT is the text of the theme boot script. It is one
 *  constant: nothing is filled in per page or per value. The two values
 *  it needs are read at run time from the data-storage-key and
 *  data-theme-color-light attributes of its own script element, through
 *  document.currentScript, which is set while a classic inline script
 *  runs. When the element or either attribute is missing the script does
 *  nothing.
 *
 *  What it does, in order: read the stored choice under the storage key,
 *  ignore anything but light, dark or system, set or clear data-mode on
 *  the root element, resolve system through prefers-color-scheme, and for
 *  a light result set the theme-color meta to the light colour. */
export const THEME_BOOT_TEXT =
  "(function(){try{var s=document.currentScript;var key=s&&s.getAttribute('data-storage-key');var light=s&&s.getAttribute('data-theme-color-light');if(!key||!light){return}var raw=localStorage.getItem(key);if(!raw){return}var stored=JSON.parse(raw);var mode=stored&&stored.mode;if(mode!=='light'&&mode!=='dark'&&mode!=='system'){return}var root=document.documentElement;var resolved=mode;if(mode==='system'){root.removeAttribute('data-mode');resolved=window.matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light'}else{root.setAttribute('data-mode',mode)}if(resolved==='light'){var meta=document.querySelector('meta[name=\"theme-color\"]');if(meta){meta.setAttribute('content',light)}}}catch(e){}})();"

/** themeBootElement is the complete script element: the constant text,
 *  with the storage key and the light theme colour as data attributes. */
export function themeBootElement(storageKey: string, lightColor: string): string {
  return `<script id="theme-boot" data-storage-key="${escapeAttribute(storageKey)}" data-theme-color-light="${escapeAttribute(lightColor)}">${THEME_BOOT_TEXT}</script>`
}
