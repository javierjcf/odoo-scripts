# odoo-freeze-repos (Rust)

Congela y descongela el `repos.yaml` de un proyecto [Doodba](https://github.com/Tecnativa/doodba)
a los commits que ya hay en disco, para que el siguiente `invoke git-aggregate` descargue
exactamente el mismo código.

Es el tercer port de la misma herramienta, con idéntico comportamiento y ficheros de respaldo:

| Versión | Carpeta |
|---|---|
| Python (original) | [`../odoo-freeze-repos`](../odoo-freeze-repos) |
| Go | [`../odoo-freeze-repos-go`](../odoo-freeze-repos-go) |
| **Rust** (esta) | `odoo-freeze-repos-rs` |

Usa siete crates bien conocidos (ver [Dependencias](#dependencias)) y genera un único binario.
La versión Go y la original en Python usan solo su biblioteca estándar.

## Requisitos

- Rust 1.88 o superior, edición 2024 (`rustup`; se probó con 1.99) y acceso a crates.io para la primera compilación.
- `git` en el `PATH` (la herramienta lo ejecuta para leer los commits).

## Compilar e instalar

```bash
cargo build --release     # binario en target/release/odoo-freeze-repos
cargo install --path .    # instala en ~/.cargo/bin (ya está en tu PATH si usas rustup)
cargo test                # tests unitarios + de integración
cargo clippy --all-targets && cargo fmt --check    # análisis y formato
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

`-n` / `--dry-run` (solo `freeze` y `unfreeze`) muestra el diff de `repos.yaml` sin escribir nada,
ni `repos.yaml` ni los respaldos.

## Ficheros de respaldo

Se crean en `odoo/custom/src/`:

| Fichero | Contenido | Cuándo se escribe |
|---|---|---|
| `bck-repos.yaml` | El **original**, antes del primer `freeze` | Solo si no existe. Nunca se sobrescribe; para empezar de cero, bórralo |
| `frozen-repos.yaml` | El **último estado congelado** | En cada `unfreeze`, antes de descongelar |

## Estructura del código

```
Cargo.toml
src/
  main.rs           binario fino: llama a cli::run y devuelve el código de salida
  lib.rs            raíz de la biblioteca: declara los módulos
  cli.rs            argumentos y subcomandos (clap derive)
  config.rs         lectura de repos.yaml y common.yaml (serde)
  freezer.rs        utilidades comunes de los comandos (+ diff con similar)
  freezer/          freeze.rs, unfreeze.rs, list.rs
  reposfile.rs      edición línea a línea de repos.yaml (bloques, marcas, sello)
  gitx.rs           llamadas a git y resolución de refs
  project.rs        rutas de un proyecto Doodba
tests/
  common/mod.rs     helpers (repos git de prueba)
  freeze.rs         ciclo freeze/unfreeze con repos git reales
  cli.rs            la CLI de punta a punta
docs/conceptos-rust.md
```

Los tests unitarios viven dentro de cada módulo (`#[cfg(test)] mod tests`), junto al código.

## Dependencias

| Crate | Para qué |
|---|---|
| `clap` | Argumentos, subcomandos, `--help` y `--version` |
| `serde` + `serde_yaml_ng` | Leer `repos.yaml` y `common.yaml` |
| `regex` | Reconocer claves y líneas de `merges` al editar `repos.yaml` |
| `similar` | Diff unificado de `--dry-run` |
| `chrono` | Fecha y hora del sello |
| `anyhow` | Errores con contexto |
| `tempfile` (solo tests) | Directorios temporales |

`repos.yaml` se **lee** con serde, pero se **edita** línea a línea: ningún crate YAML conserva
los comentarios y el formato al reescribir. `git` se ejecuta como proceso, sin `git2`/`gix`.

Con ellos el código tiene ~980 líneas (sin tests) frente a ~1.440 de la versión solo con `std`;
a cambio, el binario pasa de 458 KB a 2,7 MB y la compilación release de ~3 s a ~25 s.

## Documentación

- [docs/conceptos-rust.md](docs/conceptos-rust.md): conceptos de Rust usados en este proyecto
  y diferencias con las versiones Python y Go.
