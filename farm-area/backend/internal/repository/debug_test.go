package repository

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sovanna2011/farm-area/backend/internal/domain"
)

func TestDebugScope(t *testing.T) {
	year := 2026
	farm := 1
	f := domain.Filter{CropYear: &year, FarmID: &farm}
	b := areaSource(f, "ba")
	b.applyScope(f, "ba")
	scope := plantingScope(b, f)
	q := fmt.Sprintf(`A %[2]s B %[2]s C %[1]s D %[3]s`, b.source, scope, b.whereSQL())
	for _, line := range strings.Split(q, "\n") {
		t.Log(line)
	}
}
