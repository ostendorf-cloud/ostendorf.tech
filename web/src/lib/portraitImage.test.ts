/**
 * The portrait's variants are listed with a width next to each file. The
 * widths are read back from the files' own headers, so a srcset can never
 * claim more pixels than a file has, and the original's 1200 is the ceiling.
 */
import { readFileSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

import { PORTRAIT_VARIANTS, portraitAvif, portraitFallbackSet, portraitWebp } from './heroImage'

/** Pixel size from the file header: JPEG SOF, WebP VP8/VP8L/VP8X, AVIF ispe. */
function dimensions(file: Buffer): { width: number; height: number } {
  if (file[0] === 0xff && file[1] === 0xd8) {
    let i = 2
    while (i + 9 < file.length) {
      if (file[i] !== 0xff) {
        i += 1
        continue
      }
      const marker = file[i + 1] ?? 0
      if (marker >= 0xc0 && marker <= 0xcf && ![0xc4, 0xc8, 0xcc].includes(marker)) {
        return { width: file.readUInt16BE(i + 7), height: file.readUInt16BE(i + 5) }
      }
      i += 2 + file.readUInt16BE(i + 2)
    }
    throw new Error('no JPEG frame header')
  }
  if (file.toString('ascii', 0, 4) === 'RIFF' && file.toString('ascii', 8, 12) === 'WEBP') {
    const kind = file.toString('ascii', 12, 16)
    if (kind === 'VP8 ') {
      return { width: file.readUInt16LE(26) & 0x3fff, height: file.readUInt16LE(28) & 0x3fff }
    }
    if (kind === 'VP8X') {
      return { width: file.readUIntLE(24, 3) + 1, height: file.readUIntLE(27, 3) + 1 }
    }
    if (kind === 'VP8L') {
      const bits = file.readUInt32LE(21)
      return { width: (bits & 0x3fff) + 1, height: ((bits >> 14) & 0x3fff) + 1 }
    }
    throw new Error(`unknown WebP chunk ${kind}`)
  }
  const ispe = file.indexOf('ispe', 0, 'ascii')
  if (ispe > 0) {
    return { width: file.readUInt32BE(ispe + 8), height: file.readUInt32BE(ispe + 12) }
  }
  throw new Error('unknown image format')
}

/** Vite hands a test the file's path below the project root. */
function read(url: string): Buffer {
  return readFileSync(join(process.cwd(), url.replace(/\?.*$/, '')))
}

const all = Object.entries(PORTRAIT_VARIANTS).flatMap(([format, list]) =>
  list.map(([url, width]) => ({ format, url, width })),
)

describe('the portrait variants', () => {
  it('list the same three widths for every format, none above the original 1200', () => {
    for (const list of Object.values(PORTRAIT_VARIANTS)) {
      expect(list.map(([, width]) => width)).toEqual([460, 920, 1200])
    }
  })

  it.each(all)('$format $width: the file exists and is that many pixels wide', (variant) => {
    const bytes = read(variant.url)
    expect(dimensions(bytes)).toEqual({ width: variant.width, height: variant.width })
    expect(statSync(join(process.cwd(), variant.url)).size).toBeLessThan(150 * 1024)
  })

  it('reach the srcset strings with their widths', () => {
    for (const set of [portraitAvif, portraitWebp, portraitFallbackSet]) {
      expect(set.split(', ').map((entry) => entry.split(' ')[1])).toEqual(['460w', '920w', '1200w'])
    }
  })
})
