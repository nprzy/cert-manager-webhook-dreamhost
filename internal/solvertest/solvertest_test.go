package solvertest

// Minimal test file to satisfy Go 1.25.4 coverage requirements
//
// Upon upgrading from Go 1.24.0 to 1.25.4, testing began to fail with the error:
//
//   ⎿  Error: Exit code 2
//   # github.com/nprzy/cert-manager-webhook-dreamhost/internal/solvertest
//   go: no such tool "covdata"
//   make: *** [Makefile:15: test] Error 1
//
// Adding this empty test solved it. The internal/solvertest package is used by
// the other unit tests and doesn't really need its own tests.

