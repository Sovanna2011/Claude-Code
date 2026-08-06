package memory_test

import (
	"testing"

	"github.com/kss/sugarplan/internal/store"
	"github.com/kss/sugarplan/internal/store/memory"
	"github.com/kss/sugarplan/internal/store/storetest"
)

// The in-memory store must satisfy the same contract as PostgreSQL, because
// the demo profile and the unit tests run against it.
func TestConformance(t *testing.T) {
	storetest.Run(t, func(t *testing.T) store.Store { return memory.New() })
}
