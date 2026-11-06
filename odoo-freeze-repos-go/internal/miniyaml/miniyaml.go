// Package miniyaml es un lector YAML mínimo (solo librería estándar): mapas, listas de
// escalares y comentarios.
// Cubre repos.yaml y common.yaml de Doodba; no soporta anclas ni estilo flow.
//
// Valores posibles dentro del árbol (tipo `any`, el "object" de Go):
//   - string          escalar
//   - []string        lista de escalares
//   - map[string]any  mapa anidado
//   - nil             clave sin valor
package miniyaml

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type yamlItem struct {
	indent int
	text   string
}

// stripComment quita el comentario ' #...' final respetando comillas.
func stripComment(text string) string {
	var quote byte // 0 = fuera de comillas
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#' && (i == 0 || text[i-1] == ' ' || text[i-1] == '\t'):
			return text[:i]
		}
	}
	return text
}

// scalar recorta espacios y quita un par de comillas envolventes.
func scalar(text string) string {
	text = strings.TrimSpace(text)
	if len(text) >= 2 && text[0] == text[len(text)-1] && (text[0] == '"' || text[0] == '\'') {
		return text[1 : len(text)-1]
	}
	return text
}

// Cabeceras de bloque literal: |, >, |-, >-, |+, >+
var blockScalarMarkers = map[string]bool{
	"|": true, ">": true, "|-": true, ">-": true, "|+": true, ">+": true,
}

var blockHeaderRE = regexp.MustCompile(`:\s*[|>][+-]?$`)

// parseBlock parsea mapas y listas de escalares por indentación.
// Devuelve el valor y la nueva posición.
func parseBlock(items []yamlItem, pos, indent int) (any, int, error) {
	first := items[pos].text
	if strings.HasPrefix(first, "- ") || first == "-" {
		list := []string{}
		for pos < len(items) && items[pos].indent == indent && strings.HasPrefix(items[pos].text, "-") {
			list = append(list, scalar(items[pos].text[1:]))
			pos++
		}
		return list, pos, nil
	}

	result := map[string]any{}
	for pos < len(items) && items[pos].indent == indent {
		key, rest, found := strings.Cut(items[pos].text, ":")
		if !found {
			return nil, pos, fmt.Errorf("YAML no soportado: '%s'", items[pos].text)
		}
		key, rest = scalar(key), strings.TrimSpace(rest)
		pos++

		switch {
		case blockScalarMarkers[rest]:
			// Texto de bloque: no hace falta su contenido.
			for pos < len(items) && items[pos].indent > indent {
				pos++
			}
			result[key] = ""
		case rest != "":
			result[key] = scalar(rest)
		case pos < len(items) && (items[pos].indent > indent ||
			(items[pos].indent == indent && strings.HasPrefix(items[pos].text, "-"))):
			value, next, err := parseBlock(items, pos, items[pos].indent)
			if err != nil {
				return nil, next, err
			}
			result[key] = value
			pos = next
		default:
			result[key] = nil
		}
	}
	return result, pos, nil
}

// Parse parsea el texto completo. El nivel superior debe ser un mapa.
func Parse(src string) (map[string]any, error) {
	var items []yamlItem
	blockIndent := -1 // indentación del padre de un bloque literal '|' en curso; -1 = ninguno
	for _, raw := range strings.Split(src, "\n") {
		raw = strings.TrimRight(raw, "\r")
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		if blockIndent >= 0 {
			if strings.TrimSpace(raw) == "" || indent > blockIndent {
				continue
			}
			blockIndent = -1
		}
		text := strings.TrimSpace(stripComment(raw))
		if text == "" || text == "---" {
			continue
		}
		items = append(items, yamlItem{indent, text})
		if blockHeaderRE.MatchString(text) {
			blockIndent = indent
		}
	}
	if len(items) == 0 {
		return map[string]any{}, nil
	}
	value, _, err := parseBlock(items, 0, items[0].indent)
	if err != nil {
		return nil, err
	}
	m, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("YAML no soportado: el nivel superior no es un mapa")
	}
	return m, nil
}

// Load lee y parsea un fichero.
func Load(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m, err := Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return m, nil
}

// Conversiones "seguras": si el valor no es del tipo esperado devuelven el valor cero
// (mapa nil, lista nil, "") en lugar de fallar. Es el idioma `v, ok := x.(T)`.

func AsMap(v any) map[string]any { m, _ := v.(map[string]any); return m }
func AsList(v any) []string      { l, _ := v.([]string); return l }
func AsString(v any) string      { s, _ := v.(string); return s }

// Dig baja por mapas anidados siguiendo `keys`; devuelve nil si algún nivel falta.
func Dig(v any, keys ...string) any {
	for _, k := range keys {
		v = AsMap(v)[k] // leer de un mapa nil es válido en Go y da nil
	}
	return v
}
