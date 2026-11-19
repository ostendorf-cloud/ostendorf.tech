package pow

// Known values for salt "00112233445566778899aabb?expires=1790000000&issued=1789998200&",
// number 4242 and key "test-key". web/src/lib/pow-solve.test.ts checks the same
// challenge against the browser solver.
const (
	knownSalt      = "00112233445566778899aabb?expires=1790000000&issued=1789998200&"
	knownChallenge = "0a07febdfddcf8cff1953cbe2187efd37439e2d4bf0d78e7ed16fb4e58fdf810"
	knownSignature = "d6c028d44050be1f0920e9b4a5a42602c8b57d31da9e548fff9682ed6756662e"
)
