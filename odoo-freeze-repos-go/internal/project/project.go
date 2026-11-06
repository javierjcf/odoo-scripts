// Package project modela las rutas de un proyecto Doodba a partir de su carpeta raíz.
package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Project agrupa las rutas de un proyecto Doodba.
type Project struct {
	Root       string
	Src        string
	ReposYAML  string
	BackupYAML string
	FrozenYAML string
	CommonYAML string
}

// New calcula las rutas a partir de la raíz. No toca el disco.
func New(root string) (*Project, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	src := filepath.Join(abs, "odoo", "custom", "src")
	return &Project{
		Root:       abs,
		Src:        src,
		ReposYAML:  filepath.Join(src, "repos.yaml"),
		BackupYAML: filepath.Join(src, "bck-repos.yaml"),
		FrozenYAML: filepath.Join(src, "frozen-repos.yaml"),
		CommonYAML: filepath.Join(abs, "common.yaml"),
	}, nil
}

// Rel devuelve la ruta relativa a la raíz del proyecto (para mensajes).
func (p *Project) Rel(path string) string {
	r, err := filepath.Rel(p.Root, path)
	if err != nil {
		return path
	}
	return r
}

// Validate comprueba que existan los ficheros imprescindibles.
func (p *Project) Validate(needCommon bool) error {
	required := []string{p.Src, p.ReposYAML}
	if needCommon {
		required = append(required, p.CommonYAML)
	}
	var missing []string
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			missing = append(missing, path)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s no parece un proyecto Doodba; falta: %v", p.Root, missing)
	}
	return nil
}

// DiscoverRepos devuelve los repos con .git bajo odoo/custom/src (sin private/), ordenados.
func (p *Project) DiscoverRepos() ([]string, error) {
	entries, err := os.ReadDir(p.Src)
	if err != nil {
		return nil, err
	}
	var repos []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "private" {
			continue
		}
		if info, err := os.Stat(filepath.Join(p.Src, e.Name(), ".git")); err == nil && info.IsDir() {
			repos = append(repos, e.Name())
		}
	}
	sort.Strings(repos)
	return repos, nil
}
