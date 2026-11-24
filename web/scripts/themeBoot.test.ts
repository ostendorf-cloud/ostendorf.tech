import { describe, expect, it } from 'vitest'

import { escapeAttribute, THEME_BOOT_TEXT, themeBootElement } from './themeBoot'

/** inner is the text between the open and close tags. */
function inner(element: string): string {
  const start = element.indexOf('>') + 1
  return element.slice(start, element.lastIndexOf('</script>'))
}

describe('escapeAttribute', () => {
  it.each([
    ['&', '&amp;'],
    ['<', '&lt;'],
    ['>', '&gt;'],
    ['"', '&quot;'],
    ["'", '&#39;'],
  ])('writes %j as %s', (character, escaped) => {
    expect(escapeAttribute(`a${character}b`)).toBe(`a${escaped}b`)
  })

  it('leaves a plain value alone', () => {
    expect(escapeAttribute('govue-theme')).toBe('govue-theme')
  })
})

describe('themeBootElement', () => {
  it('has the same text whatever the values are', () => {
    const a = themeBootElement('key-one', '#111111')
    const b = themeBootElement('key-two', '#222222')
    expect(inner(a)).toBe(THEME_BOOT_TEXT)
    expect(inner(b)).toBe(THEME_BOOT_TEXT)
    expect(inner(a)).not.toContain('key-one')
    expect(inner(a)).not.toContain('#111111')
  })

  it('carries both values as data attributes of the element with the id', () => {
    const element = themeBootElement('govue-theme', '#f5f5f4')
    expect(element.startsWith('<script id="theme-boot" ')).toBe(true)
    expect(element).toContain('data-storage-key="govue-theme"')
    expect(element).toContain('data-theme-color-light="#f5f5f4"')
  })

  it('writes the attribute values escaped', () => {
    const element = themeBootElement('a"b<c>&\'d', 'x"y')
    expect(element).toContain('data-storage-key="a&quot;b&lt;c&gt;&amp;&#39;d"')
    expect(element).toContain('data-theme-color-light="x&quot;y"')
    expect(element.slice(0, element.indexOf('>'))).not.toContain('<c')
  })

  it('applies a stored light choice when run as an element', () => {
    document.head.innerHTML = '<meta name="theme-color" content="#000000">'
    document.documentElement.setAttribute('data-mode', 'dark')
    localStorage.setItem('boot-test', JSON.stringify({ mode: 'light' }))
    const script = document.createElement('script')
    script.id = 'theme-boot'
    script.setAttribute('data-storage-key', 'boot-test')
    script.setAttribute('data-theme-color-light', '#f5f5f4')
    script.textContent = THEME_BOOT_TEXT
    document.head.appendChild(script)
    expect(document.documentElement.getAttribute('data-mode')).toBe('light')
    expect(document.querySelector('meta[name="theme-color"]')?.getAttribute('content')).toBe(
      '#f5f5f4',
    )
    localStorage.removeItem('boot-test')
  })

  it('does nothing when an attribute is missing', () => {
    document.documentElement.setAttribute('data-mode', 'dark')
    localStorage.setItem('boot-test', JSON.stringify({ mode: 'light' }))
    const script = document.createElement('script')
    script.setAttribute('data-storage-key', 'boot-test')
    script.textContent = THEME_BOOT_TEXT
    document.head.appendChild(script)
    expect(document.documentElement.getAttribute('data-mode')).toBe('dark')
    localStorage.removeItem('boot-test')
  })

  it('reads both values from the element at run time', () => {
    expect(THEME_BOOT_TEXT).toContain("getAttribute('data-storage-key')")
    expect(THEME_BOOT_TEXT).toContain("getAttribute('data-theme-color-light')")
    expect(THEME_BOOT_TEXT).toContain('document.currentScript')
  })
})
