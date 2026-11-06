// odoo-freeze-repos: congela/descongela el repos.yaml de un proyecto Doodba a los commits
// que ya existen en disco, para que el siguiente `invoke git-aggregate` descargue
// exactamente el mismo código.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/javier/odoo-freeze-repos/internal/freezer"
	"github.com/javier/odoo-freeze-repos/internal/project"
)

const usage = `Uso:
  odoo-freeze-repos freeze   [-n] RUTA [REPO,REPO ...]
  odoo-freeze-repos unfreeze [-n] RUTA [REPO,REPO ...]
  odoo-freeze-repos list          RUTA [REPO,REPO ...]

RUTA es la raiz del proyecto Doodba (donde estan common.yaml y odoo/). Sin REPO actua
sobre todos los repos con .git bajo odoo/custom/src/ (excluyendo siempre private/).
REPO admite nombres separados por comas o espacios.

  freeze    sustituye cada ref de 'merges' por su SHA (la ref original queda en '#frozen:<ref>')
  unfreeze  devuelve repos.yaml a sus refs flotantes
  list      muestra fecha, hash y mensaje del commit actual de cada repo

Opciones de freeze/unfreeze:
  -n, --dry-run   muestra el diff de repos.yaml sin escribir nada
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run contiene toda la lógica de la CLI y devuelve el código de salida.
// Recibe los writers para poder probarla sin tocar os.Stdout.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprint(stderr, usage)
		return 2
	}
	command := args[0]
	if command != "freeze" && command != "unfreeze" && command != "list" {
		fmt.Fprintf(stderr, "error: comando desconocido: %q\n\n%s", command, usage)
		return 2
	}

	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	var dryRun bool
	if command != "list" {
		fs.BoolVar(&dryRun, "dry-run", false, "muestra el diff sin escribir nada")
		fs.BoolVar(&dryRun, "n", false, "alias de --dry-run")
	}
	positional, err := parseInterspersed(fs, args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if len(positional) < 1 {
		fmt.Fprintf(stderr, "error: falta RUTA\n\n%s", usage)
		return 2
	}

	p, err := project.New(positional[0])
	if err == nil {
		err = p.Validate(command == "freeze")
	}
	if err == nil {
		repos := splitRepos(positional[1:])
		switch command {
		case "freeze":
			err = freezer.Freeze(p, repos, dryRun, stdout)
		case "unfreeze":
			err = freezer.Unfreeze(p, repos, dryRun, stdout)
		case "list":
			err = freezer.List(p, repos, stdout)
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

// parseInterspersed permite poner los flags en cualquier posición (`freeze RUTA -n`).
// El paquete flag de Go se detiene en el primer argumento que no es un flag.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return positional, nil
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
}

// splitRepos admite "a,b c" -> [a b c] (comas y/o espacios).
func splitRepos(args []string) []string {
	var out []string
	for _, a := range args {
		for _, r := range strings.Split(a, ",") {
			if r != "" {
				out = append(out, r)
			}
		}
	}
	return out
}
