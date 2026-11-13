/**
 * Solves the contact form's proof of work challenge off the main thread.
 * Bundled as its own chunk through Vite's `new Worker(new URL(...))`
 * convention (see lib/pow.ts); never imported directly by application
 * code. Cookie-free and third-party-free: this file never touches
 * document, fetch or anything outside the one message it answers.
 */
import { solve } from '../lib/pow-solve'

interface Request {
  challenge: string
  salt: string
  maxnumber: number
}

interface Response {
  number: number | null
  took?: number
  error?: string
}

self.onmessage = (event: MessageEvent<Request>) => {
  const { challenge, salt, maxnumber } = event.data
  const started = Date.now()
  solve(challenge, salt, maxnumber).then(
    (number) => {
      self.postMessage({ number, took: Date.now() - started } satisfies Response)
    },
    (error: unknown) => {
      self.postMessage({ number: null, error: String(error) } satisfies Response)
    },
  )
}
