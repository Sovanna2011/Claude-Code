package api

import _ "embed"

// openAPISpec is the API contract, embedded so the running binary always serves
// the specification it actually implements. The contract test in this package
// parses it and asserts that every documented path exists in the router.
//
//go:embed openapi.yaml
var openAPISpec []byte

// OpenAPISpec exposes the document for tooling and tests.
func OpenAPISpec() []byte { return openAPISpec }
