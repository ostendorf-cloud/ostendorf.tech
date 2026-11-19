import type { ApiClient } from '@ostendorf-cloud/govue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { createChallengeSolver } from './contactChallenge'

/** fakeClient answers GET /challenge with a canned, never-expiring
 *  challenge; nothing in this suite calls POST /contact. */
function fakeClient(overrides: Partial<{ maxnumber: number; csrfToken: string }> = {}): ApiClient {
  const now = Math.floor(Date.now() / 1000)
  const salt = `00112233445566778899aabb?expires=${now + 600}&issued=${now}&`
  return {
    get: vi.fn().mockResolvedValue({
      algorithm: 'SHA-256',
      challenge: 'c'.repeat(64),
      maxnumber: overrides.maxnumber ?? 100,
      salt,
      signature: 'deadbeef',
      csrfToken: overrides.csrfToken ?? 'token-1',
    }),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    del: vi.fn(),
  }
}

/** FakeWorker answers with a fixed number on the next microtask, instead
 *  of solving anything: the solving algorithm itself is pow-solve.test.ts's
 *  job, this suite is only about the fetch/solve/wait orchestration. */
class FakeWorker {
  onmessage: ((event: MessageEvent<{ number: number }>) => void) | null = null
  onerror: (() => void) | null = null
  posted: unknown

  postMessage(message: unknown): void {
    this.posted = message
    queueMicrotask(() =>
      this.onmessage?.({ data: { number: 7 } } as MessageEvent<{ number: number }>),
    )
  }

  terminate(): void {}
}

beforeEach(() => {
  vi.stubGlobal('Worker', FakeWorker)
  vi.useFakeTimers()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe('createChallengeSolver', () => {
  it('waits out the minimum age before resolving', async () => {
    const client = fakeClient()
    const solver = createChallengeSolver(client, () => 'de')

    const result = solver.solved()
    // The minimum age has not passed yet: the promise must not settle.
    await vi.advanceTimersByTimeAsync(1000)
    let settled = false
    void result.then(() => (settled = true))
    await Promise.resolve()
    expect(settled).toBe(false)

    await vi.advanceTimersByTimeAsync(3000)
    const solution = await result
    expect(solution.pow.number).toBe(7)
    expect(solution.csrfToken).toBe('token-1')
  })

  it('passes the locale given at call time to every fetch', async () => {
    const client = fakeClient()
    let current = 'de'
    const solver = createChallengeSolver(client, () => current)

    current = 'en'
    const pending = solver.solved()
    await vi.advanceTimersByTimeAsync(4000)
    await pending

    expect(client.get).toHaveBeenCalledWith('/challenge', { query: { locale: 'en' } })
  })

  it('reuses a prepared challenge instead of fetching twice', async () => {
    const client = fakeClient()
    const solver = createChallengeSolver(client, () => 'de')

    solver.prepare()
    await Promise.resolve()
    const pending = solver.solved()
    await vi.advanceTimersByTimeAsync(4000)
    await pending

    expect(client.get).toHaveBeenCalledTimes(1)
  })
})
