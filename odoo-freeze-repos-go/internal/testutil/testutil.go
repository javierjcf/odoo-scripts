// Package testutil reúne helpers compartidos por los tests de varios paquetes.
// Es un paquete normal (no _test.go) para poder importarlo desde otros paquetes;
// al estar bajo internal/, solo este módulo puede usarlo.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// CommonYAML y ReposYAML son el contenido inicial del proyecto de prueba.
const CommonYAML = `services:
  odoo:
    build:
      args:
        ODOO_VERSION: "16.0"
`

const ReposYAML = `./odoo:
  defaults:
    depth: $DEPTH_MERGE
  remotes:
    origin: https://example.com/odoo.git
  target: origin $ODOO_VERSION
  merges:
    - origin $ODOO_VERSION
`

// Git ejecuta git en dir (con identidad y firma desactivada) y falla el test si hay error.
func Git(t *testing.T, dir string, args ...string) string {
	t.Helper() // los fallos se reportan en la línea del test que llamó, no aquí
	full := append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// Write crea (con sus carpetas) un fichero con ese contenido.
func Write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Read devuelve el contenido de un fichero.
func Read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// NewRepo crea un repo con un commit y la rama remota simulada origin/16.0.
// Devuelve el SHA del commit.
func NewRepo(t *testing.T, dir, url string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	Git(t, dir, "init", "-q")
	Git(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
	Git(t, dir, "remote", "add", "origin", url)
	Git(t, dir, "update-ref", "refs/remotes/origin/16.0", "HEAD")
	return Git(t, dir, "rev-parse", "HEAD")
}

// NewProject monta en un directorio temporal un proyecto Doodba con 'odoo' (en repos.yaml)
// y 'extra' (implícito). Devuelve la raíz y el SHA de cada repo.
// Se salta el test si git no está instalado.
func NewProject(t *testing.T) (root, odooSHA, extraSHA string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no esta instalado")
	}
	root = t.TempDir() // se borra solo al terminar el test
	Write(t, filepath.Join(root, "common.yaml"), CommonYAML)
	Write(t, filepath.Join(root, "odoo/custom/src/repos.yaml"), ReposYAML)
	odooSHA = NewRepo(t, filepath.Join(root, "odoo/custom/src/odoo"), "https://example.com/odoo.git")
	extraSHA = NewRepo(t, filepath.Join(root, "odoo/custom/src/extra"), "https://example.com/extra.git")
	return root, odooSHA, extraSHA
}
