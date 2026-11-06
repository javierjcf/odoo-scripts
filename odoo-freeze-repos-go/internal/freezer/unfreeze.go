package freezer

import (
	"fmt"
	"io"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/javier/odoo-freeze-repos/internal/diff"
	"github.com/javier/odoo-freeze-repos/internal/gitx"
	"github.com/javier/odoo-freeze-repos/internal/miniyaml"
	"github.com/javier/odoo-freeze-repos/internal/project"
	"github.com/javier/odoo-freeze-repos/internal/reposfile"
)

// Unfreeze devuelve repos.yaml a sus refs flotantes. Con dryRun solo imprime el diff.
// Copia el estado congelado a frozen-repos.yaml antes de tocar nada.
func Unfreeze(p *project.Project, requested []string, dryRun bool, out io.Writer) error {
	data, err := miniyaml.Load(p.ReposYAML)
	if err != nil {
		return err
	}
	lines, err := reposfile.ReadLines(p.ReposYAML)
	if err != nil {
		return err
	}
	original := slices.Clone(lines)

	targets := requested
	if len(targets) == 0 {
		for key := range data { // el orden de iteración de un map en Go es aleatorio
			targets = append(targets, reposfile.NormalizeKey(key))
		}
		sort.Strings(targets)
	}

	markerIdx := reposfile.FindAddedMarker(lines)
	var restored, removed, skipped []string

	for _, name := range targets {
		var start, end int
		key, ok := reposfile.RepoKey(data, name)
		if ok {
			start, end, ok = reposfile.FindBlock(lines, name)
		}
		if !ok {
			skipped = append(skipped, fmt.Sprintf("%s: sin entrada en repos.yaml", name))
			continue
		}

		// Repos añadidos por freeze (tras la marca): se eliminan, no se descongelan.
		if markerIdx >= 0 && start > markerIdx {
			lines = slices.Delete(lines, start, end)
			removed = append(removed, name)
			continue
		}

		block := lines[start:end] // vista, no copia (ver freeze.go)
		entry := entryFromYAML(data[key])
		touched := false

		for i, value := range entry.Merges {
			remote, ref, ok := reposfile.SplitMerge(value)
			if !ok || !gitx.IsSHA(ref) {
				continue
			}
			idx := reposfile.FindMergeLine(block, value)
			if idx < 0 {
				continue
			}
			markerRE := regexp.MustCompile(regexp.QuoteMeta(value) + `\s+#frozen:(\S+)`)
			if m := markerRE.FindStringSubmatch(block[idx]); m != nil {
				block[idx] = strings.Replace(block[idx], m[0], remote+" "+m[1], 1)
				touched = true
			} else if i == 0 && entry.Target != "" {
				// SHA sin marca (fijado a mano): la base es siempre 'target'.
				block[idx] = strings.Replace(block[idx], value, entry.Target, 1)
				touched = true
			} else {
				skipped = append(skipped, fmt.Sprintf("%s: '%s' sin ref original, restauralo a mano", name, value))
			}
		}
		if touched {
			restored = append(restored, name)
		}
	}

	// Si ya no queda ningún repo añadido, se quita también la marca y el hueco que dejó freeze.
	if markerIdx >= 0 {
		tail := lines[markerIdx+1:]
		hasKey := slices.ContainsFunc(tail, func(l string) bool { _, ok := reposfile.MatchTopKey(l); return ok })
		if !hasKey {
			lines = lines[:markerIdx]
			if n := len(lines); n > 0 && strings.TrimSpace(lines[n-1]) == "" {
				lines = lines[:n-1]
			}
		}
	}

	if !slices.ContainsFunc(lines, func(l string) bool { return strings.Contains(l, "#frozen:") }) {
		lines = reposfile.RemoveStamp(lines) // ya no queda nada congelado
	}

	if slices.Equal(lines, original) {
		fmt.Fprintln(out, "Nada que descongelar.")
	} else {
		if dryRun {
			fmt.Fprintln(out, "[dry-run] No se escribe nada. Cambios que se aplicarian a repos.yaml:")
			fmt.Fprintln(out)
			diff.Unified(out, p.Rel(p.ReposYAML)+" (congelado)", p.Rel(p.ReposYAML), original, lines, 3)
			fmt.Fprintln(out)
		} else {
			if err := reposfile.CopyFile(p.ReposYAML, p.FrozenYAML); err != nil {
				return err
			}
			if err := reposfile.WriteLines(p.ReposYAML, lines); err != nil {
				return err
			}
		}
		if len(restored) > 0 {
			fmt.Fprintln(out, "Descongelados: "+strings.Join(restored, ", "))
		}
		if len(removed) > 0 {
			fmt.Fprintln(out, "Eliminados (anadidos por freeze): "+strings.Join(removed, ", "))
		}
		if !dryRun {
			fmt.Fprintf(out, "Estado congelado guardado en %s\n", p.Rel(p.FrozenYAML))
			fmt.Fprintf(out, "Original (antes del primer freeze) en %s\n", p.Rel(p.BackupYAML))
			fmt.Fprintln(out, "Ahora ejecuta 'invoke git-aggregate' para traer lo ultimo antes de volver a congelar.")
		}
	}
	if len(skipped) > 0 {
		fmt.Fprintln(out, "\nAvisos:")
		for _, msg := range skipped {
			fmt.Fprintf(out, "  - %s\n", msg)
		}
	}
	return nil
}
