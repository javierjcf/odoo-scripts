package reposfile

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func TestFindBlock(t *testing.T) {
	lines := strings.SplitAfter("# cabecera\n./odoo:\n  target: x\n\naccount:\n  target: y\n", "\n")
	lines = lines[:len(lines)-1]

	start, end, ok := FindBlock(lines, "odoo")
	if !ok || start != 1 || end != 4 {
		t.Errorf("odoo: got (%d,%d,%v)", start, end, ok)
	}
	start, end, ok = FindBlock(lines, "account")
	if !ok || start != 4 || end != 6 {
		t.Errorf("account: got (%d,%d,%v)", start, end, ok)
	}
	if _, _, ok := FindBlock(lines, "nope"); ok {
		t.Error("nope no deberia existir")
	}
}

func TestFindMergeLine(t *testing.T) {
	block := []string{
		"repo:\n",
		"  merges:\n",
		"    # - origin comentado\n",
		"    - origin $ODOO_VERSION  #frozen:x\n",
		"    - origin 123\n",
	}
	if got := FindMergeLine(block, "origin $ODOO_VERSION"); got != 3 {
		t.Errorf("got %d, want 3", got)
	}
	if got := FindMergeLine(block, "origin 123"); got != 4 {
		t.Errorf("got %d, want 4", got)
	}
	if got := FindMergeLine(block, "origin comentado"); got != -1 {
		t.Errorf("las lineas comentadas no deben coincidir, got %d", got)
	}
}

func TestStampRoundTrip(t *testing.T) {
	lines := []string{"a:\n"}
	lines = SetStamp(lines, time.Date(2026, 3, 6, 9, 5, 0, 0, time.UTC))
	if lines[0] != "# Congelado con odoo-freeze-repos el 2026-03-06 09:05\n" {
		t.Errorf("sello: %q", lines[0])
	}
	lines = SetStamp(lines, time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC)) // actualiza, no duplica
	if len(lines) != 2 {
		t.Errorf("el sello se duplico: %v", lines)
	}
	if got := RemoveStamp(lines); !slices.Equal(got, []string{"a:\n"}) {
		t.Errorf("got %v", got)
	}
}

func TestSplitMerge(t *testing.T) {
	remote, ref, ok := SplitMerge("origin  refs/pull/1/head ")
	if !ok || remote != "origin" || ref != "refs/pull/1/head" {
		t.Errorf("got %q %q %v", remote, ref, ok)
	}
	if _, _, ok := SplitMerge("solo"); ok {
		t.Error("sin espacio no es un merge valido")
	}
}
