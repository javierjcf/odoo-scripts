package reposfile

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// NormalizeKey quita el prefijo "./" de una clave de repos.yaml.
func NormalizeKey(key string) string { return strings.TrimPrefix(key, "./") }

// RepoKey devuelve la clave tal cual aparece en el mapa ("name" o "./name").
func RepoKey(data map[string]any, name string) (string, bool) {
	for key := range data {
		if NormalizeKey(key) == name {
			return key, true
		}
	}
	return "", false
}

// Clave de nivel superior tipo "./odoo:" o "account-reconcile:" (nunca indentada ni comentada).
// RE2 (el motor de regex de Go) no tiene lookahead: `(?!\s)(?!#)(\S.*)` pasa a `[^\s#].*`.
var topKeyRE = regexp.MustCompile(`^([^\s#].*):\s*$`)

// MatchTopKey indica si la línea es una clave de nivel superior y devuelve su texto.
func MatchTopKey(line string) (string, bool) {
	m := topKeyRE.FindStringSubmatch(strings.TrimRight(line, "\r\n"))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// FindBlock devuelve [start, end) de las líneas del bloque de nivel superior de `name`.
func FindBlock(lines []string, name string) (start, end int, ok bool) {
	start = -1
	for i, line := range lines {
		if key, isKey := MatchTopKey(line); isKey && NormalizeKey(key) == name {
			start = i
			break
		}
	}
	if start < 0 {
		return 0, 0, false
	}
	end = len(lines)
	for j := start + 1; j < len(lines); j++ {
		if _, isKey := MatchTopKey(lines[j]); isKey {
			end = j
			break
		}
	}
	return start, end, true
}

var mergeLineRE = regexp.MustCompile(`^\s*-\s+(.+?)\s*(?:#.*)?$`)

// FindMergeLine devuelve el índice de la línea '- <value>' (con comentario opcional detrás);
// ignora líneas comentadas. -1 si no hay ninguna.
func FindMergeLine(block []string, value string) int {
	for i, line := range block {
		if m := mergeLineRE.FindStringSubmatch(strings.TrimRight(line, "\r\n")); m != nil && m[1] == value {
			return i
		}
	}
	return -1
}

// SplitMerge separa "remote ref" en sus dos partes.
func SplitMerge(merge string) (remote, ref string, ok bool) {
	merge = strings.TrimSpace(merge)
	i := strings.IndexAny(merge, " \t")
	if i < 0 {
		return "", "", false
	}
	return merge[:i], strings.TrimSpace(merge[i:]), true
}

// AddedMarker separa los repos añadidos por freeze del resto de repos.yaml.
const AddedMarker = "# --- A partir de aqui: repos anadidos por odoo-freeze-repos, antes implicitos via addons.yaml ---\n"

// Marca del nombre anterior del script: se sigue reconociendo en repos.yaml ya congelados.
var legacyAddedMarker = strings.ReplaceAll(AddedMarker, "odoo-freeze-repos", "freeze_repos.py")

// FindAddedMarker devuelve el índice de la línea marca, o -1.
func FindAddedMarker(lines []string) int {
	for i, line := range lines {
		if line == AddedMarker || line == legacyAddedMarker {
			return i
		}
	}
	return -1
}

// ensureAddedMarker inserta el comentario separador una única vez.
// Devuelve el slice: `append` puede reservar memoria nueva, así que SIEMPRE hay que reasignar.
func ensureAddedMarker(lines []string) []string {
	if FindAddedMarker(lines) >= 0 {
		return lines
	}
	if n := len(lines); n > 0 && !strings.HasSuffix(lines[n-1], "\n") {
		lines[n-1] += "\n"
	}
	return append(lines, "\n", AddedMarker)
}

func AppendNewBlock(lines []string, name, url, sha string) []string {
	lines = ensureAddedMarker(lines)
	return append(lines,
		"\n",
		name+":\n",
		"  defaults:\n",
		"    depth: $DEPTH_MERGE\n",
		"  remotes:\n",
		"    origin: "+url+"\n",
		"  target: origin $ODOO_VERSION\n",
		"  merges:\n",
		"    - origin "+sha+"  #frozen:$ODOO_VERSION\n",
	)
}

// Prefijos de la primera línea de repos.yaml con la fecha del último freeze.
const (
	StampPrefix       = "# Congelado con odoo-freeze-repos el "
	legacyStampPrefix = "# Congelado con freeze_repos.py el " // nombre anterior del script
)

func hasStamp(lines []string) bool {
	return len(lines) > 0 && (strings.HasPrefix(lines[0], StampPrefix) || strings.HasPrefix(lines[0], legacyStampPrefix))
}

// SetStamp pone (o actualiza) en la primera línea la fecha y hora del freeze.
func SetStamp(lines []string, now time.Time) []string {
	// Go formatea fechas con una fecha de referencia ("2006-01-02 15:04"), no con %Y-%m-%d.
	stamp := fmt.Sprintf("%s%s\n", StampPrefix, now.Format("2006-01-02 15:04"))
	if hasStamp(lines) {
		lines[0] = stamp
		return lines
	}
	return append([]string{stamp}, lines...)
}

func RemoveStamp(lines []string) []string {
	if hasStamp(lines) {
		return lines[1:]
	}
	return lines
}
