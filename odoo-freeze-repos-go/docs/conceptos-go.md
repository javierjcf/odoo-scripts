# Conceptos de Go usados en este proyecto

Guía pensada para quien viene de Python. Cada concepto se explica con código real del
proyecto y, cuando aplica, con su equivalente en
[`odoo-freeze-repos.py`](../../odoo-freeze-repos/odoo-freeze-repos.py).

Índice

1. [Herramientas y comandos](#1-herramientas-y-comandos)
2. [Módulos, paquetes e imports](#2-módulos-paquetes-e-imports)
3. [Organización: `cmd/` e `internal/`](#3-organización-cmd-e-internal)
4. [Visibilidad: mayúscula = público](#4-visibilidad-mayúscula--público)
5. [Variables, tipos y valores cero](#5-variables-tipos-y-valores-cero)
6. [Errores como valores](#6-errores-como-valores)
7. [Structs y métodos](#7-structs-y-métodos)
8. [`any` y aserciones de tipo](#8-any-y-aserciones-de-tipo)
9. [Slices: vistas, no copias](#9-slices-vistas-no-copias)
10. [Maps](#10-maps)
11. [Strings, bytes y runes](#11-strings-bytes-y-runes)
12. [Expresiones regulares (RE2)](#12-expresiones-regulares-re2)
13. [`defer`](#13-defer)
14. [Interfaces: `io.Writer`](#14-interfaces-iowriter)
15. [El paquete `flag`](#15-el-paquete-flag)
16. [Fechas](#16-fechas)
17. [Tests](#17-tests)
18. [Biblioteca estándar útil](#18-biblioteca-estándar-útil)
19. [Compilar y distribuir](#19-compilar-y-distribuir)

---

## 1. Herramientas y comandos

Todo viene con el comando `go`; no hacen falta `pip`, `venv`, `black`, `pytest` ni `mypy`.

| Comando | Para qué | Equivalente Python |
|---|---|---|
| `go build ./...` | Compila (comprueba tipos) | — (Python no compila) |
| `go run ./cmd/odoo-freeze-repos list RUTA` | Compila y ejecuta | `python script.py` |
| `go test ./...` | Ejecuta los tests de todos los paquetes | `pytest` |
| `go test -v -run TestFreeze ./internal/freezer` | Un test concreto, con detalle | `pytest -k` |
| `go test -cover ./...` | Cobertura | `pytest --cov` |
| `gofmt -w .` | Formatea el código (estilo único, sin discusión) | `black` |
| `go vet ./...` | Análisis estático de errores típicos | `pyflakes` / `ruff` |
| `go install ./cmd/...` | Instala el binario en `~/go/bin` | `pipx install` |
| `go mod init` / `go mod tidy` | Crea el módulo / ordena dependencias | `pyproject.toml` / `pip freeze` |
| `go doc strings.Cut` | Documentación en terminal | `pydoc` |

`./...` significa "este directorio y todos los inferiores".

## 2. Módulos, paquetes e imports

- Un **módulo** es el proyecto entero. Se define en [`go.mod`](../go.mod) con su nombre
  (`github.com/javier/odoo-freeze-repos`) y la versión de Go. Es lo más parecido a un
  `pyproject.toml`. No hay dependencias externas, así que no existe `go.sum`.
- Un **paquete** es una carpeta: todos los `.go` de una carpeta comparten la primera línea
  `package nombre` y se compilan juntos. No hace falta importarse entre ellos (a diferencia de
  los módulos `.py`, que cada uno es un espacio de nombres).
- Un import usa la ruta del módulo más la carpeta:

```go
import "github.com/javier/odoo-freeze-repos/internal/gitx"

gitx.RunGit(...)   // se usa con el nombre del paquete
```

- Go **no permite imports ni variables sin usar**: es un error de compilación, no un aviso.
- El paquete `main` con una función `main()` es el punto de entrada
  ([cmd/odoo-freeze-repos/main.go](../cmd/odoo-freeze-repos/main.go)). El `if __name__ ==
  "__main__"` de Python no existe.
- No hay imports circulares: si A importa B, B no puede importar A. Por eso existe
  [`internal/testutil`](../internal/testutil/testutil.go), separado de los paquetes que lo usan.

## 3. Organización: `cmd/` e `internal/`

Convención muy extendida (no obligatoria):

```
cmd/<nombre>/main.go    un directorio por ejecutable; main.go lo mínimo posible
internal/<paquete>/     código del proyecto
```

- `internal/` es especial: **el compilador impide** que otros módulos lo importen. Es API
  privada garantizada.
- El `main` solo interpreta argumentos y delega. La lógica vive en paquetes pequeños y con una
  responsabilidad, que se prueban por separado:

| Paquete | Responsabilidad | Depende de |
|---|---|---|
| `miniyaml` | Parsear YAML | — |
| `diff` | Diff unificado | — |
| `gitx` | Ejecutar git y resolver refs | — |
| `reposfile` | Editar `repos.yaml` por líneas | — |
| `project` | Rutas del proyecto Doodba | — |
| `freezer` | Los tres comandos | todos los anteriores |

Las flechas solo apuntan hacia abajo; eso mantiene el código fácil de seguir.

## 4. Visibilidad: mayúscula = público

No hay `public`/`private` ni prefijo `_`. La **primera letra** decide:

```go
func RunGit(...)   // exportada: visible desde otros paquetes (gitx.RunGit)
func isSHA(...)    // privada: solo dentro de su paquete
```

Misma regla para campos de structs (`Project.Root` vs un hipotético `root`), constantes y
tipos. Por eso al mover código a paquetes hubo que renombrar `runGit` a `RunGit`: antes
estaba todo en un único paquete.

## 5. Variables, tipos y valores cero

```go
var dryRun bool        // declaración explícita; vale false
name := "web"          // := declara e infiere el tipo (solo dentro de funciones)
name = "account"       // = asigna a una variable ya declarada
```

- Tipado estático: no puedes asignar un `int` a una variable `string`.
- **Valor cero**: toda variable tiene un valor por defecto útil: `0`, `""`, `false`, `nil`
  (punteros, slices, maps, interfaces). Se aprovecha mucho; por ejemplo `entryFromYAML` en
  [freezer.go](../internal/freezer/freezer.go) devuelve un `repoEntry` con campos vacíos si el
  YAML no tenía nada.
- `const` para constantes (`addedMarker`, `stampPrefix`); `var` si hace falta calcularlas
  (`legacyAddedMarker` usa `strings.ReplaceAll`, que no puede ser `const`).
- `_` descarta un valor: `_, ref, _ := reposfile.SplitMerge(m)`.
- Funciones con **varios retornos**: `func SplitMerge(s string) (remote, ref string, ok bool)`.
  Es la alternativa a devolver tuplas.

## 6. Errores como valores

Python:

```python
try:
    sha = resolve_ref(...)
except subprocess.CalledProcessError as exc:
    ...
```

Go:

```go
sha, err := gitx.ResolveRef(...)
if err != nil {
    // manejar o propagar
}
```

- `error` es un tipo (una interfaz) y `nil` significa "sin error".
- Se **propaga** hacia arriba añadiendo contexto con `fmt.Errorf("...: %w", err)`. `%w`
  envuelve el error original, de modo que luego se puede inspeccionar con `errors.Is` /
  `errors.As`.
- No hay `sys.exit()` repartido por el código: las funciones devuelven `error` y solo `main`
  decide el código de salida (ver `run` en [main.go](../cmd/odoo-freeze-repos/main.go)).
- Es verboso a propósito: cada punto de fallo queda visible en la lectura.

En [gitx.go](../internal/gitx/gitx.go) también aparece la **aserción de tipo** para sacar el
stderr de git: `if ee, ok := err.(*exec.ExitError); ok { ... }`.

## 7. Structs y métodos

No hay clases. Un `struct` agrupa datos y los métodos se declaran aparte con un *receptor*:

```go
type Project struct { Root, Src, ReposYAML string }

func (p *Project) Validate(needCommon bool) error { ... }
```

- `*Project` es un **puntero**: el método recibe una referencia, no una copia. Se usa cuando
  el método modifica el struct o el struct es grande. Se llama igual: `p.Validate(true)`.
- `New(root)` en [project.go](../internal/project/project.go) es la convención de
  *constructor*: una función normal que devuelve el struct. No existe `__init__`.
- No hay herencia; se compone con campos y con interfaces.

## 8. `any` y aserciones de tipo

El YAML parseado tiene forma libre. En Python bastan `dict`/`list`/`str`. En Go el tipo
`any` (alias de `interface{}`) admite cualquier valor, y para usarlo hay que comprobar qué
es:

```go
m, ok := v.(map[string]any)   // "comma ok": ok es false si v no era un mapa
```

[miniyaml.go](../internal/miniyaml/miniyaml.go) encapsula eso en `AsMap`, `AsList` y
`AsString`, que devuelven el valor cero si el tipo no coincide. Combinado con que **leer de
un map nil es válido**, permite `AsMap(v)["target"]` sin comprobaciones intermedias.

## 9. Slices: vistas, no copias

Un slice (`[]string`) es una ventana sobre un array subyacente. Esto es lo que más sorprende
a quien viene de Python.

**`a[i:j]` NO copia**, comparte memoria:

```go
block := lines[start:end]
block[2] = "otra cosa"       // ¡también cambia lines[start+2]!
```

En [freeze.go](../internal/freezer/freeze.go) se aprovecha: modificar `block[idx]` ya
modifica `lines`. En Python hacía falta `lines[start:end] = block_lines` porque
`lines[start:end]` crea una lista nueva.

**`append` puede reservar memoria nueva**, así que se reasigna siempre:

```go
lines = append(lines, "nueva\n")      // correcto
append(lines, "nueva\n")              // error de compilación: resultado sin usar
```

Por eso `AppendNewBlock`, `SetStamp` y `RemoveStamp` **devuelven** el slice.

**Copiar de verdad** (para conservar el original y compararlo):

```go
original := slices.Clone(lines)
resolved := append([]string(nil), merges...)   // idioma clásico de copia
```

Otras operaciones del paquete `slices`: `slices.Delete`, `slices.Equal`,
`slices.ContainsFunc`. Los slices **no se pueden comparar con `==`**; hay que usar
`slices.Equal`.

## 10. Maps

```go
known := make(map[string]bool, len(all))
known["web"] = true
if known[name] { ... }          // clave inexistente => valor cero (false)
v, ok := m["clave"]             // para distinguir "no existe" de "vale cero"
```

- **El orden de iteración es aleatorio a propósito.** Python 3.7+ conserva el orden de
  inserción; Go no. Cuando importa el orden hay que ordenar las claves
  (`sort.Strings`, ver `Unfreeze`).
- Un map `nil` se puede leer pero **no escribir** (panic).

## 11. Strings, bytes y runes

- Un `string` es una secuencia inmutable de **bytes** (UTF-8). `len(s)` son bytes, no
  caracteres. `s[i]` es un byte.
- Para tratar caracteres Unicode: `[]rune(s)` (un `rune` es un punto de código). Se usa en
  `shortRefs` para no cortar un carácter por la mitad.
- Los parsers del proyecto trabajan con bytes (`text[i] == '#'`) porque los caracteres
  que buscan (`#`, comillas, `:`) son ASCII y UTF-8 no los confunde con otros.
- El paquete `strings` cubre lo habitual:

| Python | Go |
|---|---|
| `s.strip()` | `strings.TrimSpace(s)` |
| `s.lstrip(" ")` | `strings.TrimLeft(s, " ")` |
| `s.startswith(p)` | `strings.HasPrefix(s, p)` |
| `a in s` | `strings.Contains(s, a)` |
| `s.split(",")` | `strings.Split(s, ",")` |
| `s.partition(":")` | `strings.Cut(s, ":")` |
| `s.replace(a, b, 1)` | `strings.Replace(s, a, b, 1)` |
| `s.splitlines(True)` | `strings.SplitAfter(s, "\n")` (+ quitar el elemento vacío final) |
| `", ".join(xs)` | `strings.Join(xs, ", ")` |
| `f"{x:<{w}}"` | `fmt.Sprintf("%-*s", w, x)` |

## 12. Expresiones regulares (RE2)

El paquete `regexp` usa RE2: tiempo lineal garantizado, pero **sin lookahead/lookbehind ni
backreferences**. La regex original `^(?!\s)(?!#)(\S.*):\s*$` se reescribió como
`^([^\s#].*):\s*$` (ver [reposfile.go](../internal/reposfile/reposfile.go)).

```go
var topKeyRE = regexp.MustCompile(`^([^\s#].*):\s*$`)   // MustCompile hace panic si es inválida
m := topKeyRE.FindStringSubmatch(line)                  // nil si no hay coincidencia
// m[0] = coincidencia completa, m[1] = primer grupo
```

- Se compilan una vez a nivel de paquete (`var`), no dentro de bucles.
- Los *raw strings* con comillas invertidas (`` `...` ``) evitan escapar las barras.
- `regexp.QuoteMeta(s)` equivale a `re.escape(s)`.
- `$` solo casa al final del texto (Python también lo hace antes de un `\n` final), por eso
  se hace `strings.TrimRight(line, "\r\n")` antes de aplicar la regex.

## 13. `defer`

`defer` programa una llamada para cuando la función termine, por cualquier camino
(retorno normal, error o panic). Es el equivalente de `with` / `try/finally`:

```go
in, err := os.Open(src)
if err != nil { return err }
defer in.Close()
// ... usar in; se cierra solo al salir
```

Ver `CopyFile` en [io.go](../internal/reposfile/io.go).

## 14. Interfaces: `io.Writer`

Una interfaz es un conjunto de métodos; cualquier tipo que los tenga la cumple, **sin
declararlo** (tipado estructural, como el *duck typing* de Python pero comprobado al compilar).

```go
type Writer interface { Write(p []byte) (n int, err error) }
```

`os.Stdout`, `*bytes.Buffer` y `*strings.Builder` son `io.Writer`. Por eso:

```go
func Freeze(p *project.Project, requested []string, dryRun bool, out io.Writer) error
fmt.Fprintf(out, "Actualizados:\n")   // escribe en lo que le pasen
```

En producción `out` es `os.Stdout`; en los tests es un `bytes.Buffer` y se comprueba lo
escrito. Python usaba `print()`, que obliga a capturar stdout para probarlo.

## 15. El paquete `flag`

```go
fs := flag.NewFlagSet("freeze", flag.ContinueOnError)
fs.BoolVar(&dryRun, "dry-run", false, "descripción")
fs.Parse(args)
```

- Acepta `-n` y `--dry-run` indistintamente.
- **Se detiene en el primer argumento que no es un flag.** Con `argparse` puedes escribir
  `freeze RUTA -n`; con `flag`, `-n` se tomaría como un argumento más. Se resuelve en
  `parseInterspersed` ([main.go](../cmd/odoo-freeze-repos/main.go)), que vuelve a llamar a
  `Parse` tras cada posicional.
- Un `FlagSet` por subcomando es el patrón estándar sin librerías externas (alternativa:
  `cobra`).

## 16. Fechas

Go no usa `%Y-%m-%d`. Se escribe cómo se vería **la fecha de referencia**
`Mon Jan 2 15:04:05 MST 2006` (los números 1 2 3 4 5 6 7):

```go
now.Format("2006-01-02 15:04")   // 2026-11-05 23:35
```

## 17. Tests

- Un test vive en `*_test.go`, en el mismo paquete, y es una función `TestXxx(t *testing.T)`.
  `go test` los encuentra solo; no hay fixtures mágicos ni decoradores.
- **Tests de tabla**: lista de casos + `t.Run` (equivale a `pytest.mark.parametrize`).
  Ver `TestParseYAML` en [miniyaml_test.go](../internal/miniyaml/miniyaml_test.go).
- Fallos: `t.Errorf` (sigue) o `t.Fatalf` (para el test).
- Helpers de `testing.T` usados aquí:
  - `t.Helper()`: los fallos se atribuyen a la línea del test que llamó al helper.
  - `t.TempDir()`: directorio temporal que se borra solo (como `tmp_path`).
  - `t.Skip(...)`: se salta si falta algo (aquí, git).
- Los tests de [freezer_test.go](../internal/freezer/freezer_test.go) crean **repos git
  reales** en directorios temporales; el más valioso comprueba que `freeze` + `unfreeze`
  devuelve `repos.yaml` exactamente igual que el original.
- Los helpers compartidos entre paquetes están en un paquete normal,
  [`internal/testutil`](../internal/testutil/testutil.go), porque el código de un `_test.go`
  no se puede importar desde otro paquete.

## 18. Biblioteca estándar útil

| Necesidad | Paquete |
|---|---|
| Ejecutar procesos | `os/exec` (`exec.Command(...).Output()` ≈ `subprocess.run`) |
| Ficheros | `os` (`ReadFile`, `WriteFile`, `Stat`, `MkdirAll`) |
| Rutas | `path/filepath` (≈ `pathlib`) |
| Formato / impresión | `fmt` (`Sprintf`, `Fprintf`, `Errorf`) |
| Orden | `sort`, `slices` |
| Mínimo / máximo | `min()` y `max()` (incluidos en el lenguaje desde Go 1.21) |
| Errores | `errors` (`Is`, `As`) |

## 19. Compilar y distribuir

- `go build` produce **un único ejecutable estático**: sin intérprete, sin venv y sin
  instalar dependencias en la máquina destino.
- Compilación cruzada con variables de entorno:

```bash
GOOS=linux  GOARCH=arm64 go build -o bin/odoo-freeze-repos-arm64 ./cmd/odoo-freeze-repos
GOOS=darwin GOARCH=arm64 go build -o bin/odoo-freeze-repos-mac   ./cmd/odoo-freeze-repos
```

- `go install ./cmd/odoo-freeze-repos` deja el binario en `~/go/bin` (ver el `README` para
  añadirlo al `PATH` sin romper otras máquinas).
- Reducir el tamaño: `go build -ldflags="-s -w" ...` quita símbolos de depuración.
