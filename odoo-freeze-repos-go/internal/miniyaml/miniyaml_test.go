package miniyaml

import (
	"reflect"
	"strings"
	"testing"
)

// Tests de tabla: la forma idiomática en Go de probar muchos casos con una sola función.
func TestParseYAML(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want map[string]any
	}{
		{
			name: "entrada tipica de repos.yaml",
			src: `./odoo:
  defaults:
    depth: $DEPTH_MERGE
  remotes:
    origin: https://github.com/odoo/odoo.git  # comentario
  target: origin $ODOO_VERSION
  merges:
    - origin $ODOO_VERSION
    - origin abc  #frozen:x
`,
			want: map[string]any{"./odoo": map[string]any{
				"defaults": map[string]any{"depth": "$DEPTH_MERGE"},
				"remotes":  map[string]any{"origin": "https://github.com/odoo/odoo.git"},
				"target":   "origin $ODOO_VERSION",
				"merges":   []string{"origin $ODOO_VERSION", "origin abc"},
			}},
		},
		{
			name: "lista al mismo nivel que la clave",
			src:  "repo:\n  merges:\n  - origin a\n  - origin b\n",
			want: map[string]any{"repo": map[string]any{"merges": []string{"origin a", "origin b"}}},
		},
		{
			name: "comillas y almohadilla dentro de comillas",
			src:  "a: \"16.0\"\nb: 'x # no es comentario'\nc: v # si lo es\n",
			want: map[string]any{"a": "16.0", "b": "x # no es comentario", "c": "v"},
		},
		{
			name: "bloque literal se ignora",
			src:  "a: |\n  texto: raro\n  mas\nb: 1\n",
			want: map[string]any{"a": "", "b": "1"},
		},
		{
			name: "separador --- y clave sin valor",
			src:  "---\nvacio:\notro: x\n",
			want: map[string]any{"vacio": nil, "otro": "x"},
		},
		{
			name: "fichero vacio",
			src:  "# solo comentarios\n",
			want: map[string]any{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.src)
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v\nwant %#v", got, tc.want)
			}
		})
	}
}

func TestParseYAMLError(t *testing.T) {
	if _, err := Parse("sin dos puntos\n"); err == nil || !strings.Contains(err.Error(), "no soportado") {
		t.Errorf("se esperaba error 'no soportado', got %v", err)
	}
}

func TestDigAndHelpers(t *testing.T) {
	data, _ := Parse("services:\n  odoo:\n    build:\n      args:\n        ODOO_VERSION: \"16.0\"\n")
	if got := AsString(Dig(data, "services", "odoo", "build", "args", "ODOO_VERSION")); got != "16.0" {
		t.Errorf("got %q", got)
	}
	if got := Dig(data, "services", "nope", "x"); got != nil {
		t.Errorf("se esperaba nil, got %v", got)
	}
}
