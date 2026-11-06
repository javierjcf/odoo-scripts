package freezer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/javier/odoo-freeze-repos/internal/project"
	"github.com/javier/odoo-freeze-repos/internal/reposfile"
	"github.com/javier/odoo-freeze-repos/internal/testutil"
)

// newProject monta el proyecto de prueba y lo abre como *project.Project.
func newProject(t *testing.T) (p *project.Project, odooSHA, extraSHA string) {
	t.Helper()
	root, odooSHA, extraSHA := testutil.NewProject(t)
	p, err := project.New(root)
	if err != nil {
		t.Fatal(err)
	}
	return p, odooSHA, extraSHA
}

func TestFreezeUnfreezeRoundTrip(t *testing.T) {
	p, odooSHA, extraSHA := newProject(t)
	var out bytes.Buffer

	if err := Freeze(p, nil, false, &out); err != nil {
		t.Fatal(err)
	}
	frozen := testutil.Read(t, p.ReposYAML)
	for _, want := range []string{
		"# Congelado con odoo-freeze-repos el ",
		"    - origin " + odooSHA + "  #frozen:$ODOO_VERSION\n",
		reposfile.AddedMarker,
		"extra:\n",
		"    origin: https://example.com/extra.git\n",
		"    - origin " + extraSHA + "  #frozen:$ODOO_VERSION\n",
	} {
		if !strings.Contains(frozen, want) {
			t.Errorf("repos.yaml congelado no contiene %q:\n%s", want, frozen)
		}
	}
	if testutil.Read(t, p.BackupYAML) != testutil.ReposYAML {
		t.Error("bck-repos.yaml deberia ser el original")
	}

	// Un segundo freeze no debe sobrescribir el backup ni cambiar nada.
	out.Reset()
	if err := Freeze(p, nil, false, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Nada que congelar") {
		t.Errorf("segundo freeze: %s", out.String())
	}

	// unfreeze debe devolver el fichero EXACTAMENTE al original.
	out.Reset()
	if err := Unfreeze(p, nil, false, &out); err != nil {
		t.Fatal(err)
	}
	if got := testutil.Read(t, p.ReposYAML); got != testutil.ReposYAML {
		t.Errorf("unfreeze no restauro el original:\n%s", got)
	}
	if testutil.Read(t, p.FrozenYAML) != frozen {
		t.Error("frozen-repos.yaml deberia contener el estado congelado")
	}
}

func TestFreezeDryRunDoesNotWrite(t *testing.T) {
	p, odooSHA, _ := newProject(t)
	var out bytes.Buffer
	if err := Freeze(p, nil, true, &out); err != nil {
		t.Fatal(err)
	}
	if testutil.Read(t, p.ReposYAML) != testutil.ReposYAML {
		t.Error("--dry-run modifico repos.yaml")
	}
	if reposfile.FileExists(p.BackupYAML) {
		t.Error("--dry-run creo el backup")
	}
	if !strings.Contains(out.String(), "+    - origin "+odooSHA+"  #frozen:$ODOO_VERSION") {
		t.Errorf("el diff no muestra el cambio:\n%s", out.String())
	}
}

func TestFreezeSpecificRepoAndUnresolvable(t *testing.T) {
	p, odooSHA, _ := newProject(t)
	testutil.Write(t, p.ReposYAML, strings.Replace(testutil.ReposYAML,
		"    - origin $ODOO_VERSION\n",
		"    - origin $ODOO_VERSION\n    - origin refs/pull/9/head\n", 1))
	var out bytes.Buffer
	if err := Freeze(p, []string{"odoo"}, false, &out); err != nil {
		t.Fatal(err)
	}
	got := testutil.Read(t, p.ReposYAML)
	if !strings.Contains(got, "origin "+odooSHA+"  #frozen:$ODOO_VERSION") {
		t.Errorf("no se congelo la rama:\n%s", got)
	}
	if !strings.Contains(got, "    - origin refs/pull/9/head\n") {
		t.Errorf("la ref sin resolver deberia quedar intacta:\n%s", got)
	}
	if strings.Contains(got, "extra:") {
		t.Error("extra no se pidio y no deberia anadirse")
	}
	if !strings.Contains(out.String(), "Omitidos:") {
		t.Errorf("deberia avisar de la ref no resuelta:\n%s", out.String())
	}
}
