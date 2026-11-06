// Package freezer implementa los comandos freeze, unfreeze y list sobre un proyecto Doodba.
package freezer

import (
	"fmt"
	"os"
	"strings"

	"github.com/javier/odoo-freeze-repos/internal/gitx"
	"github.com/javier/odoo-freeze-repos/internal/miniyaml"
	"github.com/javier/odoo-freeze-repos/internal/project"
	"github.com/javier/odoo-freeze-repos/internal/reposfile"
)

// repoEntry es lo que nos interesa de una entrada de repos.yaml.
type repoEntry struct {
	Target string
	Merges []string
	URL    string // solo para entradas sintetizadas (repos que no estaban en repos.yaml)
}

func entryFromYAML(v any) repoEntry {
	m := miniyaml.AsMap(v) // si v es nil, m es un mapa nil y leerlo da valores cero
	return repoEntry{Target: miniyaml.AsString(m["target"]), Merges: miniyaml.AsList(m["merges"])}
}

// synthesizeEntry crea la entrada mínima para un repo que hoy depende del auto-generado
// via addons.yaml.
func synthesizeEntry(repoPath string) (repoEntry, error) {
	url, err := gitx.RunGit(repoPath, "config", "--get", "remote.origin.url")
	if err != nil {
		return repoEntry{}, err
	}
	return repoEntry{
		Target: "origin $ODOO_VERSION",
		Merges: []string{"origin $ODOO_VERSION"},
		URL:    url,
	}, nil
}

func getOdooVersion(p *project.Project) (string, error) {
	data, err := miniyaml.Load(p.CommonYAML)
	if err != nil {
		return "", err
	}
	version := miniyaml.AsString(miniyaml.Dig(data, "services", "odoo", "build", "args", "ODOO_VERSION"))
	if version == "" {
		return "", fmt.Errorf("%s no define services.odoo.build.args.ODOO_VERSION", p.CommonYAML)
	}
	return version, nil
}

// resolveMerges resuelve cada entrada de forma independiente. Las que fallen quedan sin tocar.
func resolveMerges(name, repoPath string, merges []string, odooVersion string) (resolved, warnings []string) {
	resolved = append([]string(nil), merges...) // copia: en Go `b := a` compartiría memoria
	for i, m := range merges {
		remote, ref, ok := reposfile.SplitMerge(m)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("%s: merges[%d] ('%s') no tiene formato 'remote ref'", name, i, m))
			continue
		}
		if gitx.IsSHA(ref) {
			continue // ya fijado a mano
		}
		sha, err := gitx.ResolveRef(repoPath, remote, ref, odooVersion)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: merges[%d] ('%s') no se pudo resolver: %v", name, i, m, err))
			continue
		}
		resolved[i] = remote + " " + sha
	}
	return resolved, warnings
}

// validateTargets filtra los repos pedidos contra los existentes; sin petición, devuelve todos.
func validateTargets(requested, all []string) []string {
	if len(requested) == 0 {
		return all
	}
	known := make(map[string]bool, len(all))
	for _, name := range all {
		known[name] = true
	}
	var valid []string
	for _, name := range requested {
		if !known[name] {
			fmt.Fprintf(os.Stderr, "[WARN] repo desconocido o sin .git: %s\n", name)
			continue
		}
		valid = append(valid, name)
	}
	return valid
}

// shortRefs devuelve los 10 primeros caracteres de la ref de cada merge.
func shortRefs(merges []string) string {
	parts := make([]string, len(merges))
	for i, m := range merges {
		_, ref, _ := reposfile.SplitMerge(m)
		r := []rune(ref)
		parts[i] = string(r[:min(10, len(r))])
	}
	return strings.Join(parts, ", ")
}
