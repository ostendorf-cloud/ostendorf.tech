/**
 * Solves the contact form's proof of work challenge: find n with
 * sha256(salt + n) == challenge. Ported from old_simon-ostendorf.de's
 * cmd/website/static/worker.js, kept as a plain function so it can run
 * inside the Web Worker (workers/pow.worker.ts) and be tested directly,
 * with no worker involved, from the same source.
 */

const encoder = new TextEncoder()

function hexToBytes(hex: string): Uint8Array {
  const bytes = new Uint8Array(hex.length / 2)
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = Number.parseInt(hex.slice(i * 2, i * 2 + 2), 16)
  }
  return bytes
}

// Comparing bytes skips a hex conversion per attempt.
function equal(buffer: ArrayBuffer, expected: Uint8Array): boolean {
  const view = new Uint8Array(buffer)
  for (let i = 0; i < expected.length; i++) {
    if (view[i] !== expected[i]) {
      return false
    }
  }
  return true
}

/** solve searches 0..maxNumber for the number whose sha256(salt + n) hex
 *  digest equals challenge, or null when none is found or the challenge is
 *  not a 64 character hex digest to begin with. */
export async function solve(
  challenge: string,
  salt: string,
  maxNumber: number,
): Promise<number | null> {
  if (challenge.length !== 64) {
    return null
  }
  const expected = hexToBytes(challenge)
  for (let n = 0; n <= maxNumber; n++) {
    const digest = await crypto.subtle.digest('SHA-256', encoder.encode(salt + String(n)))
    if (equal(digest, expected)) {
      return n
    }
  }
  return null
}
