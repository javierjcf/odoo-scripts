package main

import (
	"bytes"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/javier/odoo-freeze-repos/internal/testutil"
)

func TestSplitRepos(t *testing.T) {
	got := splitRepos([]string{"a,b", "c", ",d,"})
	if want := []string{"a", "b", "c", "d"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRunCLI(t *testing.T) {
	root, _, _ := testutil.NewProject(t)
	repos := filepath.Join(root, "odoo/custom/src/repos.yaml")
	var out, errb bytes.Buffer

	// Los flags pueden ir despues de RUTA.
	if code := run([]string{"freeze", root, "-n"}, &out, &errb); code != 0 {
		t.Fatalf("code %d: %s", code, errb.String())
	}
	if testutil.Read(t, repos) != testutil.ReposYAML {
		t.Error("-n modifico repos.yaml")
	}

	out.Reset()
	if code := run([]string{"list", root}, &out, &errb); code != 0 || !strings.Contains(out.String(), "init") {
		t.Errorf("list: code %d\n%s%s", code, out.String(), errb.String())
	}
	if code := run([]string{"nope", root}, &out, &errb); code != 2 {
		t.Errorf("comando desconocido: code %d", code)
	}
	if code := run([]string{"list", t.TempDir()}, &out, &errb); code != 1 {
		t.Errorf("proyecto invalido: code %d", code)
	}
	if code := run([]string{"freeze"}, &out, &errb); code != 2 {
		t.Errorf("sin RUTA: code %d", code)
	}
}
