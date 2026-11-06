package freezer

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/javier/odoo-freeze-repos/internal/gitx"
	"github.com/javier/odoo-freeze-repos/internal/project"
)

// List muestra fecha, hash y mensaje del commit actual de cada repo.
func List(p *project.Project, requested []string, out io.Writer) error {
	all, err := p.DiscoverRepos()
	if err != nil {
		return err
	}
	targets := validateTargets(requested, all)
	if len(targets) == 0 {
		fmt.Fprintln(out, "Nada que listar.")
		return nil
	}

	width := 0
	for _, name := range targets {
		width = max(width, len(name))
	}
	for _, name := range targets {
		info, err := gitx.RunGit(filepath.Join(p.Src, name), "log", "-1", "--date=short", "--format=%cd\t%h\t%s")
		if err != nil {
			fmt.Fprintf(out, "%-*s  ?  ?  [error: %v]\n", width, name, err)
			continue
		}
		// SplitN(..., 3) con un mensaje vacío da solo 2 trozos (TrimSpace quita el tab final).
		parts := strings.SplitN(info, "\t", 3)
		for len(parts) < 3 {
			parts = append(parts, "")
		}
		fmt.Fprintf(out, "%-*s  %s  %s  %s\n", width, name, parts[0], parts[1], parts[2])
	}
	return nil
}
