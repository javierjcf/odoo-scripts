# odoo-freeze-repos (Go)

Congela y descongela el `repos.yaml` de un proyecto [Doodba](https://github.com/Tecnativa/doodba)
a los commits que ya hay en disco, para que el siguiente `invoke git-aggregate` descargue
exactamente el mismo código.

Es el port a Go de [`../odoo-freeze-repos/odoo-freeze-repos.py`](../odoo-freeze-repos/odoo-freeze-repos.py),
con el mismo comportamiento y ficheros de respaldo, más `--dry-run`. Solo usa la librería
estándar de Go (sin dependencias externas) y genera un único binario.

## Requisitos

- Go 1.27 o superior (`sudo pacman -S go` en Arch/CachyOS).
- `git` en el `PATH` (la herramienta lo ejecuta para leer los commits).

## Compilar e instalar

```bash
make build      # binario en bin/odoo-freeze-repos
make install    # instala en ~/go/bin (añádelo al PATH)
make test       # tests
make check      # gofmt + go vet + tests
```

Sin `make`: `go build -o bin/odoo-freeze-repos ./cmd/odoo-freeze-repos`.

Para tener `odoo-freeze-repos` en cualquier terminal, añade `~/go/bin` al `PATH` en tu
`.zshrc` solo si existe (así no falla en máquinas sin Go):

```zsh
[ -d "$HOME/go/bin" ] && export PATH="$PATH:$HOME/go/bin"
```

## Uso

```bash
odoo-freeze-repos freeze   [-n] RUTA [REPO,REPO ...]
odoo-freeze-repos unfreeze [-n] RUTA [REPO,REPO ...]
odoo-freeze-repos list          RUTA [REPO,REPO ...]
```

`RUTA` es la raíz del proyecto Doodba (donde están `common.yaml` y `odoo/`). Sin `REPO` actúa
sobre todos los repos con `.git` bajo `odoo/custom/src/` (excepto `private/`). `REPO` admite
nombres separados por comas o espacios. Los flags pueden ir antes o después de `RUTA`.

| Comando | Qué hace |
|---|---|
| `freeze` | Sustituye cada ref de `merges` por su SHA y deja la ref original en `#frozen:<ref>`. Los repos que no estaban en `repos.yaml` se añaden al final, tras una línea de marca. Escribe la fecha y hora en la primera línea |
| `unfreeze` | Devuelve cada SHA con `#frozen:` a su ref; un SHA sin marca (fijado a mano) pasa a `target` si es el primer merge; elimina los repos añadidos tras la marca |
| `list` | Muestra fecha, hash y mensaje del commit actual de cada repo |

### `--dry-run` (`-n`)

En `freeze` y `unfreeze` muestra el diff de `repos.yaml` sin escribir nada (ni `repos.yaml` ni
los respaldos):

```console
$ odoo-freeze-repos freeze -n ~/proyectos/mi-doodba web
[dry-run] No se escribe nada. Cambios que se aplicarian a repos.yaml:

--- odoo/custom/src/repos.yaml
+++ odoo/custom/src/repos.yaml (congelado)
@@ -14,4 +15,4 @@
   merges:
-  - origin $ODOO_VERSION
+  - origin 03ca7f978f38...  #frozen:$ODOO_VERSION
```

## Ficheros de respaldo

Se crean en `odoo/custom/src/`:

| Fichero | Contenido | Cuándo se escribe |
|---|---|---|
| `bck-repos.yaml` | El **original**, antes del primer `freeze` | Solo si no existe. Nunca se sobrescribe; para empezar de cero, bórralo |
| `frozen-repos.yaml` | El **último estado congelado** | En cada `unfreeze`, antes de descongelar |

Para volver atrás se copia a mano el que corresponda sobre `repos.yaml`:

```bash
cp odoo/custom/src/frozen-repos.yaml odoo/custom/src/repos.yaml   # retomar el último congelado
cp odoo/custom/src/bck-repos.yaml    odoo/custom/src/repos.yaml   # volver al original
```

## Cómo resuelve las refs

Sin red, solo con lo que hay en disco:

1. Una rama normal (`origin $ODOO_VERSION`) se resuelve con `<remote>/<rama>` ya trackeada.
2. Una ref puntual (PR, etc.) se busca en el historial: el merge commit que la menciona y su
   segundo padre.

Lo que no se pueda resolver con seguridad se deja intacto y se avisa en "Omitidos".

## Estructura del código

```
cmd/odoo-freeze-repos/   main.go: CLI (flags, subcomandos, códigos de salida)
internal/freezer/        freeze, unfreeze y list
internal/reposfile/      edición línea a línea de repos.yaml (bloques, marcas, sello)
internal/miniyaml/       parser YAML mínimo (mapas, listas, comentarios)
internal/gitx/           llamadas a git y resolución de refs
internal/project/        rutas de un proyecto Doodba
internal/diff/           diff unificado para --dry-run
internal/testutil/       helpers compartidos por los tests
docs/conceptos-go.md     guía de los conceptos de Go usados, comparados con Python
```

## Documentación

- [docs/conceptos-go.md](docs/conceptos-go.md): conceptos de Go explicados con el código de
  este proyecto y comparados con la versión Python.
