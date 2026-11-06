package gitx

import (
	"strings"
	"testing"
)

func TestIsSHA(t *testing.T) {
	for in, want := range map[string]bool{
		strings.Repeat("a", 40): true,
		strings.Repeat("A", 40): true,
		strings.Repeat("g", 40): false,
		"abc":                   false,
		"$ODOO_VERSION":         false,
	} {
		if got := IsSHA(in); got != want {
			t.Errorf("IsSHA(%q) = %v", in, got)
		}
	}
}
