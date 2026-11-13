/**
 * Fetches the contact form's proof of work challenge and solves it in a
 * Web Worker, mirroring old_simon-ostendorf.de's cmd/website/static/main.js
 * contactForm() behaviour: a challenge is fetched as soon as the form is
 * focused, solved ahead of time, and re-fetched if it is about to expire
 * or already spent. The server also returns the current session's CSRF
 * token on the same response, which is why solving is always paired with
 * fetching here rather than split into two independent steps.
 */
import type { ApiClient } from '@ostendorf-cloud/govue'

export interface ChallengePayload {
  algorithm: string
  challenge: string
  number: number
  salt: string
  signature: string
}

export interface SolvedChallenge {
  receivedAt: number
  expiresAt: number
  csrfToken: string
  pow: ChallengePayload
}

/** fetched carries maxnumber alongside the solution in progress: the
 *  worker needs it as its search bound, but it is not part of the
 *  Solution the server expects back. */
interface FetchedChallenge extends SolvedChallenge {
  maxNumber: number
}

interface ChallengeResponse {
  algorithm: string
  challenge: string
  maxnumber: number
  salt: string
  signature: string
  csrfToken: string
}

// The server enforces a 3 second minimum age between issuing a challenge
// and accepting its solution; this waits slightly longer so client clock
// granularity and the round trip never make a legitimate submission look
// too fast.
const minAgeMs = 3500
// A solution within this long of its challenge's expiry is not worth
// sending: a slow connection could still land it past the server's own
// deadline, so a fresh challenge is prepared instead.
const expiryHeadroomMs = 60_000

function saltSeconds(salt: string, key: string): number {
  const match = new RegExp(`[?&]${key}=(\\d+)`).exec(salt)
  return match ? Number(match[1]) : 0
}

export interface ChallengeSolver {
  /** Starts fetching and solving a challenge, without waiting for it. Call
   *  this as early as a visitor's intent to submit is clear, typically on
   *  the form's first focus. */
  prepare: () => void
  /** Resolves to a challenge solution that is old enough to submit and not
   *  about to expire, fetching and solving a fresh one first if needed. */
  solved: () => Promise<SolvedChallenge>
}

/** createChallengeSolver builds a solver bound to client and locale, read
 *  fresh on every fetch so a locale switch mid-visit is honoured. */
export function createChallengeSolver(client: ApiClient, locale: () => string): ChallengeSolver {
  let pending: Promise<SolvedChallenge> | null = null

  async function fetchChallenge(): Promise<FetchedChallenge> {
    const data = await client.get<ChallengeResponse>('/challenge', { query: { locale: locale() } })
    const lifetimeMs = (saltSeconds(data.salt, 'expires') - saltSeconds(data.salt, 'issued')) * 1000
    const receivedAt = Date.now()
    return {
      receivedAt,
      expiresAt: receivedAt + lifetimeMs,
      csrfToken: data.csrfToken,
      maxNumber: data.maxnumber,
      pow: {
        algorithm: data.algorithm,
        challenge: data.challenge,
        number: -1,
        salt: data.salt,
        signature: data.signature,
      },
    }
  }

  function solveInWorker(challenge: FetchedChallenge): Promise<SolvedChallenge> {
    return new Promise((resolve, reject) => {
      const worker = new Worker(new URL('../workers/pow.worker.ts', import.meta.url), {
        type: 'module',
      })
      worker.onmessage = (event: MessageEvent<{ number: number | null }>) => {
        worker.terminate()
        const { number } = event.data
        if (!Number.isInteger(number)) {
          reject(new Error('the proof of work worker found no solution'))
          return
        }
        resolve({ ...challenge, pow: { ...challenge.pow, number: number as number } })
      }
      worker.onerror = () => {
        worker.terminate()
        reject(new Error('the proof of work worker failed'))
      }
      worker.postMessage({
        challenge: challenge.pow.challenge,
        salt: challenge.pow.salt,
        maxnumber: challenge.maxNumber,
      })
    })
  }

  function prepare(): Promise<SolvedChallenge> {
    const next = fetchChallenge().then(solveInWorker)
    pending = next
    // Surfaced again by solved(); this only silences the unhandled rejection
    // a prepare-and-forget call would otherwise log.
    next.catch(() => {})
    return next
  }

  return {
    prepare: () => {
      if (!pending) {
        void prepare()
      }
    },
    solved: async () => {
      let solution = await (pending ?? prepare())
      if (solution.expiresAt - Date.now() < expiryHeadroomMs) {
        solution = await prepare()
      }
      const wait = Math.min(minAgeMs, solution.receivedAt + minAgeMs - Date.now())
      if (wait > 0) {
        await new Promise((resolve) => setTimeout(resolve, wait))
      }
      // Every solution is good for one submission only.
      pending = null
      return solution
    },
  }
}
