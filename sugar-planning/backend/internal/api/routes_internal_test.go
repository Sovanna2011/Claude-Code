package api

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/kss/sugarplan/internal/store/memory"
)

// TestEveryRouteIsDocumented holds the OpenAPI document to the routing table.
//
// The public test suite already checks the other direction - that every
// documented path is routed - which catches a specification that has drifted
// ahead of the code. This one catches the commoner and quieter failure: an
// endpoint that was added and never written down. Together they mean the
// document describes the API rather than describing what somebody remembered.
func TestEveryRouteIsDocumented(t *testing.T) {
	s := &Server{store: memory.New()}
	s.routes(http.NewServeMux())
	if len(s.patterns) == 0 {
		t.Fatal("no routes were registered")
	}

	documented := documentedOperations(string(openAPISpec))
	if len(documented) == 0 {
		t.Fatal("no paths were read out of the specification; the parser is wrong, not the document")
	}

	for _, pattern := range s.patterns {
		if !documented[pattern] {
			t.Errorf("%s is routed but not documented in openapi.yaml", pattern)
		}
	}
}

// documentedOperations returns the "METHOD /api/v1/path" of every operation in
// the specification. Paths are the two-space indented keys under `paths:`;
// methods are the four-space keys beneath them.
func documentedOperations(spec string) map[string]bool {
	pathLine := regexp.MustCompile(`(?m)^  (/[^\s:]*):`)
	out := map[string]bool{}
	matches := pathLine.FindAllStringSubmatchIndex(spec, -1)
	for i, m := range matches {
		path := spec[m[2]:m[3]]
		end := len(spec)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		body := spec[m[1]:end]
		for _, method := range []string{"get", "post", "put", "delete"} {
			if strings.Contains(body, "\n    "+method+":\n") {
				out[strings.ToUpper(method)+" /api/v1"+path] = true
			}
		}
	}
	return out
}
