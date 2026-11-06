package freezer

import (
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/javier/odoo-freeze-repos/internal/diff"
	"github.com/javier/odoo-freeze-repos/internal/miniyaml"
	"github.com/javier/odoo-freeze-repos/internal/project"
	"github.com/javier/odoo-freeze-repos/internal/reposfile"
)

// frozenRepo es el resultado de congelar un repo (para el resumen final).
type frozenRepo struct {
	Name     string
	Resolved []string
}

// Freeze fija repos.yaml a los commits actuales de cada repo (o solo de `requested`).
// Con dryRun solo imprime el diff. Guarda bck-repos.yaml si no existe.
func Freeze(p *project.Project, requested []string, dryRun bool, out io.Writer) error {
	all, err := p.DiscoverRepos()
	if err != nil {
		return err
	}
	targets := validateTargets(requested, all)
	if len(targets) == 0 {
		fmt.Fprintln(out, "Nada que congelar.")
		return nil
	}

	data, err := miniyaml.Load(p.ReposYAML)
	if err != nil {
		return err
	}
	lines, err := reposfile.ReadLines(p.ReposYAML)
	if err != nil {
		return err
	}
	original := slices.Clone(lines)
	odooVersion, err := getOdooVersion(p)
	if err != nil {
		return err
	}

	var added, changed []frozenRepo
	var skipped []string

	for _, name := range targets {
		repoPath := filepath.Join(p.Src, name)
		key, exists := reposfile.RepoKey(data, name)

		var entry repoEntry
		if exists {
			entry = entryFromYAML(data[key])
		} else if entry, err = synthesizeEntry(repoPath); err != nil {
			skipped = append(skipped, fmt.Sprintf("%s: no se pudo leer remote.origin.url: %v", name, err))
			continue
		}

		if len(entry.Merges) == 0 {
			skipped = append(skipped, fmt.Sprintf("%s: sin 'merges' definido", name))
			continue
		}

		resolved, warns := resolveMerges(name, repoPath, entry.Merges, odooVersion)
		skipped = append(skipped, warns...)
		if slices.Equal(resolved, entry.Merges) {
			continue // nada resuelto (o ya estaba todo fijado)
		}

		if !exists {
			_, sha, _ := reposfile.SplitMerge(resolved[0])
			lines = reposfile.AppendNewBlock(lines, name, entry.URL, sha)
			added = append(added, frozenRepo{name, resolved})
			continue
		}

		start, end, ok := reposfile.FindBlock(lines, name)
		if !ok {
			skipped = append(skipped, fmt.Sprintf("%s: no se localizo su bloque en repos.yaml", name))
			continue
		}
		// `block` es una VISTA de lines (comparte memoria): modificar block[i] modifica lines.
		// Por eso no hace falta copiar de vuelta, a diferencia de `lines[start:end] = block` en Python.
		block := lines[start:end]
		for i, oldValue := range entry.Merges {
			newValue := resolved[i]
			if oldValue == newValue {
				continue
			}
			idx := reposfile.FindMergeLine(block, oldValue)
			if idx < 0 {
				skipped = append(skipped, fmt.Sprintf("%s: no se localizo la linea de merge '%s', no se toca", name, oldValue))
				continue
			}
			_, oldRef, _ := reposfile.SplitMerge(oldValue)
			block[idx] = strings.Replace(block[idx], oldValue, fmt.Sprintf("%s  #frozen:%s", newValue, oldRef), 1)
		}
		changed = append(changed, frozenRepo{name, resolved})
	}

	if len(added) == 0 && len(changed) == 0 {
		fmt.Fprintln(out, "Nada que congelar (todo ya estaba fijado o no se pudo resolver con seguridad).")
		if len(skipped) > 0 {
			fmt.Fprintln(out, "\nAvisos:")
			for _, msg := range skipped {
				fmt.Fprintf(out, "  - %s\n", msg)
			}
		}
		return nil
	}

	lines = reposfile.SetStamp(lines, time.Now())

	backupCreated := false
	if dryRun {
		fmt.Fprintln(out, "[dry-run] No se escribe nada. Cambios que se aplicarian a repos.yaml:")
		fmt.Fprintln(out)
		diff.Unified(out, p.Rel(p.ReposYAML), p.Rel(p.ReposYAML)+" (congelado)", original, lines, 3)
	} else {
		backupCreated = !reposfile.FileExists(p.BackupYAML)
		if backupCreated {
			if err := reposfile.CopyFile(p.ReposYAML, p.BackupYAML); err != nil {
				return err
			}
		}
		if err := reposfile.WriteLines(p.ReposYAML, lines); err != nil {
			return err
		}
	}

	if len(changed) > 0 {
		fmt.Fprintln(out, "\nActualizados:")
		for _, r := range changed {
			fmt.Fprintf(out, "  %s: %s\n", r.Name, shortRefs(r.Resolved))
		}
	}
	if len(added) > 0 {
		fmt.Fprintln(out, "\nNuevos (antes implicitos via addons.yaml):")
		for _, r := range added {
			fmt.Fprintf(out, "  %s: %s\n", r.Name, shortRefs(r.Resolved))
		}
	}
	if len(skipped) > 0 {
		fmt.Fprintln(out, "\nOmitidos:")
		for _, msg := range skipped {
			fmt.Fprintf(out, "  %s\n", msg)
		}
	}
	switch {
	case dryRun:
		// ya se ha avisado arriba
	case backupCreated:
		fmt.Fprintf(out, "\nrepos.yaml congelado; original guardado en %s\n", p.Rel(p.BackupYAML))
	default:
		fmt.Fprintf(out, "\nrepos.yaml congelado; el original ya estaba en %s (no se toca)\n", p.Rel(p.BackupYAML))
	}
	return nil
}
