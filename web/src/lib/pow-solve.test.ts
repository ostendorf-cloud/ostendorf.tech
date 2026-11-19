import { describe, expect, it } from 'vitest'

import { solve } from './pow-solve'

// The same salt, number and resulting digest internal/pow's vector_test.go
// pins on the Go side, so both implementations are checked against one
// shared, known answer rather than only against each other.
const knownSalt = '00112233445566778899aabb?expires=1790000000&issued=1789998200&'
const knownNumber = 4242
const knownChallenge = '0a07febdfddcf8cff1953cbe2187efd37439e2d4bf0d78e7ed16fb4e58fdf810'

describe('solve', () => {
  it('finds the known vector shared with the Go server', async () => {
    await expect(solve(knownChallenge, knownSalt, knownNumber + 10)).resolves.toBe(knownNumber)
  })

  it('returns null when the bound is too small to reach the answer', async () => {
    await expect(solve(knownChallenge, knownSalt, knownNumber - 1)).resolves.toBeNull()
  })

  it('returns null for a challenge that is not a 64 character hex digest', async () => {
    await expect(solve('too-short', knownSalt, 1000)).resolves.toBeNull()
  })
})
