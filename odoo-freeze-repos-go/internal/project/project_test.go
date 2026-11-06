package project

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDiscoverReposIgnoresPrivateAndNonGit(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "odoo", "custom", "src")
	for _, d := range []string{"web/.git", "account/.git", "private/.git", "sin-git"} {
		if err := os.MkdirAll(filepath.Join(src, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	p, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.DiscoverRepos()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"account", "web"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestValidate(t *testing.T) {
	p, _ := New(t.TempDir())
	if err := p.Validate(false); err == nil {
		t.Error("un directorio vacio no es un proyecto Doodba")
	}
}
