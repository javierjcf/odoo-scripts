# Conceptos de Rust usados en este proyecto

Guía para quien conoce Python y ya vio la versión Go
([`conceptos-go.md`](../../odoo-freeze-repos-go/docs/conceptos-go.md)). Cada concepto se
explica con código real de este crate y se compara con las otras dos versiones.

> **Esta versión usa crates.** Una primera versión se escribió solo con la biblioteca
> estándar (como las de Python y Go) y necesitó ~1.440 líneas; con siete crates bien elegidos
> el mismo programa tiene ~980. La sección [16](#16-crates-usados-y-por-qué-no-otros) explica
> qué sustituye cada uno y qué se pierde.

Índice

1. [Cargo y herramientas](#1-cargo-y-herramientas)
2. [Crates, módulos y visibilidad](#2-crates-módulos-y-visibilidad)
3. [Propiedad (ownership)](#3-propiedad-ownership)
4. [Préstamos: `&` y `&mut`](#4-préstamos--y-mut)
5. [`String` y `&str`](#5-string-y-str)
6. [Vec, slices y arrays](#6-vec-slices-y-arrays)
7. [Enums y `match`](#7-enums-y-match)
8. [`Option` y `Result`; el operador `?`](#8-option-y-result-el-operador-)
9. [Structs, `impl` y traits](#9-structs-impl-y-traits)
10. [Lifetimes](#10-lifetimes)
11. [Iteradores y closures](#11-iteradores-y-closures)
12. [`let else`, `if let` y let chains](#12-let-else-if-let-y-let-chains)
13. [Formato y macros](#13-formato-y-macros)
14. [Entrada/salida: `dyn Write`](#14-entradasalida-dyn-write)
15. [RAII y `Drop`](#15-raii-y-drop)
16. [Crates usados (y por qué no otros)](#16-crates-usados-y-por-qué-no-otros)
17. [Tests](#17-tests)
18. [Compilar y distribuir](#18-compilar-y-distribuir)
19. [Python vs Go vs Rust: tabla resumen](#19-python-vs-go-vs-rust-tabla-resumen)

---

## 1. Cargo y herramientas

`cargo` es a la vez gestor de paquetes, sistema de compilación, runner de tests y más.

| Comando | Para qué | Python | Go |
|---|---|---|---|
| `cargo build` / `--release` | Compila (debug / optimizado) | — | `go build` |
| `cargo run -- list RUTA` | Compila y ejecuta | `python x.py` | `go run` |
| `cargo test` | Tests unitarios, de integración y doctests | `pytest` | `go test` |
| `cargo fmt` | Formatea (rustfmt) | `black` | `gofmt` |
| `cargo clippy` | Linter con sugerencias de estilo | `ruff` | `go vet` (más limitado) |
| `cargo install --path .` | Instala el binario en `~/.cargo/bin` | `pipx install` | `go install` |
| `cargo doc --open` | Genera la documentación desde los `///` | `sphinx` | `go doc` |
| `cargo add <crate>` | Añade una dependencia | `pip install` | `go get` |

- [`Cargo.toml`](../Cargo.toml) es el `pyproject.toml`/`go.mod`. `cargo add clap --features
  derive` lo edita por ti. Aquí hay 7 dependencias directas y 1 solo para tests
  (`[dev-dependencies]`).
- Cargo genera `Cargo.lock` (versiones exactas) y la carpeta `target/` (ignorada en git).
- `edition = "2024"` fija el dialecto del lenguaje. Las ediciones permiten evolucionar la
  sintaxis sin romper código antiguo; cada crate elige la suya y todas interoperan.

## 2. Crates, módulos y visibilidad

- Un **crate** es la unidad de compilación: una biblioteca o un binario. Este paquete tiene
  los dos: [`lib.rs`](../src/lib.rs) (la lógica) y [`main.rs`](../src/main.rs) (un `main` de
  cuatro líneas). Así `tests/` puede usar la lógica como cualquier usuario externo.
- Los **módulos** forman un árbol que se declara a mano:

```rust
// lib.rs
pub mod gitx;      // busca src/gitx.rs o src/gitx/mod.rs
pub mod freezer;   // src/freezer.rs  (+ src/freezer/freeze.rs declarado dentro con `mod freeze;`)
```

  A diferencia de Go (cada carpeta es un paquete y se incluye sola) o de Python (importas lo
  que quieres), en Rust un fichero **no existe para el compilador hasta que lo declaras** con
  `mod`.
- **Visibilidad: todo es privado por defecto**, y `pub` lo abre. Es el inverso de Python
  (todo público) y más explícito que la mayúscula de Go:

```rust
pub fn freeze(...)        // visible desde fuera del crate
fn resolve_merges(...)    // privada al módulo `freezer`
pub(crate) fn x()         // visible solo dentro del crate
```

- Los hijos ven lo privado del padre: `freezer/freeze.rs` hace `use super::{entry_from_yaml, ...}`
  de funciones privadas de `freezer.rs`.
- `use` trae nombres al ámbito: `use crate::error::Result;` (`crate::` = raíz de este crate,
  `super::` = módulo padre).

## 3. Propiedad (ownership)

**Es la idea central de Rust.** Cada valor tiene exactamente un dueño; cuando el dueño sale de
ámbito, el valor se libera (sin recolector de basura, sin `free` manual). Asignar o pasar un
valor lo **mueve**:

```rust
let a = String::from("hola");
let b = a;            // a se MUEVE a b
// println!("{a}");   // error de compilación: a ya no es válido
```

- Python: variables son referencias compartidas y un GC libera. Go: igual, con GC.
- Los tipos baratos (`i32`, `bool`, `char`...) implementan `Copy` y se copian en vez de moverse.
- Copiar algo caro es **explícito**: `.clone()` o `.to_vec()`. En
  [freezer.rs](../src/freezer.rs): `let mut resolved = merges.to_vec();`. En Go
  `append([]string(nil), merges...)` o `slices.Clone`; en Python `list(merges)`.
- Consecuencia práctica: casi nunca hay copias accidentales, ni datos compartidos por error.

## 4. Préstamos: `&` y `&mut`

Para usar un valor sin quedártelo se **presta**:

- `&T`: referencia compartida, solo lectura. Puede haber **muchas** a la vez.
- `&mut T`: referencia mutable y **exclusiva**. Solo **una**, y ninguna `&T` a la vez.

El compilador (el *borrow checker*) lo comprueba en tiempo de compilación. Por eso las
funciones de [reposfile.rs](../src/reposfile.rs) que modifican reciben `&mut Vec<String>`.

**Comparación con el problema de los slices de Go** (ver `conceptos-go.md` §9). En Go,
`block := lines[start:end]` comparte memoria y un `append` posterior puede desacoplarlos o no,
según la capacidad. En Rust:

```rust
let block = &mut lines[start..end];   // préstamo mutable de una parte
// ... usar block ...
// mientras `block` viva, `lines` NO se puede tocar por otro camino: lo prohíbe el compilador
```

El mismo código que en Go "funciona por cómo se comporta la memoria", en Rust funciona
**porque está demostrado correcto**. Es lo que más se nota al pasar de uno a otro.

Los errores típicos al empezar ("cannot borrow as mutable because it is also borrowed as
immutable") son este sistema avisando de un bug real o de una estructura mejorable.

## 5. `String` y `&str`

Dos tipos de texto, ambos UTF-8 garantizado:

| Tipo | Qué es | Análogo |
|---|---|---|
| `String` | Texto **propio**, en el heap, crecible | `str` mutable de Python |
| `&str` | **Vista** de texto prestado (no copia) | un slice de Go |

Regla práctica: las funciones **reciben `&str`** (acepta literales y `&String`) y **devuelven
`String`** si crean texto nuevo.

```rust
pub fn normalize_key(key: &str) -> &str { key.strip_prefix("./").unwrap_or(key) }
```

Esa función devuelve una porción del mismo texto, sin copiar nada. Más ejemplos:

- `s[a..b]` corta por **bytes** y hace `panic` si no cae en frontera de carácter. No se
  puede indexar por posición de carácter (`s[3]` no compila). En el parser se corta solo en
  posiciones de caracteres ASCII (`#`, `:`), que siempre son frontera válida.
- `s.chars()` recorre caracteres Unicode; `s.len()` son bytes (como en Go).
- `ref` es **palabra reservada**, así que el parámetro se llama `git_ref`. (En Go y Python es
  un nombre libre.) Si de verdad lo necesitas: `r#ref`.

## 6. Vec, slices y arrays

- `Vec<T>`: array dinámico propio (la `list` de Python / el `[]T` de Go).
- `&[T]` y `&mut [T]`: **slices** prestados, una vista de parte de un `Vec` o array.
- `[T; N]`: array de tamaño fijo conocido en compilación.
- Los vectores se comparan con `==` directamente (`lines == original`); en Go hay que usar
  `slices.Equal`.
- `lines.drain(start..end)` borra un rango; `lines.truncate(n)`, `lines.insert(0, x)`.
- `text.split_inclusive('\n')` conserva el salto de línea en cada trozo; en Go hay que
  recortar un elemento vacío final a mano.
- El **mapa**: `HashMap` (orden arbitrario) o `BTreeMap` (ordenado por clave). Aquí se usa
  `BTreeMap`: `unfreeze` sin repos iterará en orden sin trabajo extra, a diferencia de Go
  (aleatorio a propósito) y de Python (orden de inserción).

## 7. Enums y `match`

Los enums de Rust son **tipos suma**: cada variante puede llevar datos.

```rust
// serde_yaml_ng::Value, el árbol de un YAML cualquiera
pub enum Value { Null, Bool(bool), Number(Number), String(String), Sequence(Vec<Value>), Mapping(Mapping), .. }
```

Esto sustituye al `any` de Go (que obliga a `v.(T)`, comprobado en ejecución) y al tipado
dinámico de Python. El compilador obliga a que un `match` cubra **todas** las variantes
(ver `odoo_version` en [config.rs](../src/config.rs)):

```rust
match value {
    Value::String(s) => Some(s.clone()),
    Value::Number(n) => Some(n.to_string()),   // ODOO_VERSION: 16.0 sin comillas
    _ => None,                                 // `_` = todo lo demás
}
```

Si añades una variante y olvidas tratarla en algún `match`, no compila. Mismo patrón en
[`cli.rs`](../src/cli.rs) con `enum Command { Freeze, Unfreeze, List }`: un comando
inexistente es **imposible de representar** una vez parseado. En Go y Python es una cadena
que hay que seguir comparando.

`match` también hace patrones con guardas:

```rust
"-n" | "--dry-run" if command != Command::List => dry_run = true,
```

## 8. `Option` y `Result`; el operador `?`

No existen `null` ni excepciones. Se usan dos enums de la biblioteca estándar:

```rust
enum Option<T> { Some(T), None }      // "puede no haber valor"
enum Result<T, E> { Ok(T), Err(E) }   // "puede fallar"
```

- Python devuelve `None` o lanza excepción; Go devuelve `(valor, error)` y se puede ignorar
  el error por descuido. En Rust **no puedes usar el valor sin tratar el caso de fallo**, y
  el compilador avisa si descartas un `Result`.
- El operador **`?`** propaga el error al llamador (equivalente a los `if err != nil { return
  err }` de Go, en un carácter):

```rust
let data = config::load_repos(&p.repos_yaml)?;
```

  Sobre un `Option` devuelve `None`. Convierte el tipo de error con `From`: con `anyhow::Error`
  cualquier error estándar (`io::Error`, el de serde...) se convierte solo, por eso
  `writeln!(out, ...)?` funciona en funciones que devuelven `anyhow::Result`.
- **`anyhow`** es el estándar de facto para aplicaciones: un único tipo de error con
  contexto. `.with_context(|| format!("no se pudo leer {}", path.display()))?` añade una capa
  y `{e:#}` imprime la cadena entera ("no se pudo leer x: No such file or directory"). Es
  lo más parecido a `fmt.Errorf("...: %w", err)` de Go. Para **bibliotecas** se usa
  `thiserror` (enums de error propios); para una CLI, `anyhow`.
- Se encadenan transformaciones sin ifs anidados:

```rust
data.get(key)                       // Option<&Option<RepoEntry>>
    .and_then(Option::as_ref)       // Option<&RepoEntry>
    .map(|entry| entry.target())    // Option<&str>
    .filter(|t| !t.is_empty())      // descarta el valor vacío
    .map(str::to_string)            // Option<String>
    .ok_or_else(|| anyhow!("falta target"))   // Option -> Result
```

- `unwrap()`/`expect()` hacen `panic` si hay `None`/`Err`. Aceptables en tests; en código
  real se evitan.
- `main` no hace `exit` por todos lados: `run()` devuelve un `u8` con el código y `main` lo
  convierte en `ExitCode`.

## 9. Structs, `impl` y traits

- `struct` agrupa datos; los métodos van en un bloque `impl` (como en Go, no dentro del tipo).
  `&self` = préstamo de solo lectura, `&mut self` = mutable, `self` = consume el valor.
- No hay constructores especiales: por convención `Project::new(...)` es una función asociada.
- Un **trait** es un conjunto de comportamientos (equivale a las interfaces de Go, pero se
  implementan **explícitamente**: `impl Trait for Tipo`, mientras que en Go basta tener los métodos).
  Ejemplos en el crate:

```rust
impl fmt::Display for MiTipo { ... }         // permite {} en format!
impl From<io::Error> for MiError { ... }     // conversión usada por `?`
impl Drop for MiRecurso { ... }              // código al salir de ámbito
```

- `#[derive(...)]` genera implementaciones automáticamente (algo así como `@dataclass`, pero
  extensible por cualquier crate). Este proyecto usa tres:
  - `#[derive(Debug, Clone)]` en [project.rs](../src/project.rs) (de la biblioteca estándar).
  - `#[derive(Deserialize)]` en [config.rs](../src/config.rs): **serde** genera el código que
    convierte un YAML en la estructura; los campos desconocidos se ignoran y los tipos
    se comprueban. Sustituye al parser casero de ~230 líneas de la primera versión.
  - `#[derive(Parser, Subcommand, Args)]` en [cli.rs](../src/cli.rs): **clap** genera el
    parseo de argumentos a partir de la estructura.
- Un trait puede aceptarse de dos formas: genérico (`fn f(w: impl Write)`, se especializa en
  compilación) o dinámico (`fn f(w: &mut dyn Write)`, vía puntero, como una interfaz de Go).
  Este crate usa `dyn Write` por simplicidad.

## 10. Lifetimes

Cuando una referencia sale de una función, el compilador necesita saber **de qué dato
depende**. Casi siempre lo deduce solo ("elisión"); a veces hay que anotarlo con `'a`:

```rust
pub fn repo_key<'a>(data: &'a ReposFile, name: &str) -> Option<&'a str>   // reposfile.rs
```

Se lee: "el `&str` devuelto es una clave de `data` y no puede vivir más que `data`" (`name` no
tiene `'a`: no se devuelve nada que dependa de él). Es una promesa comprobada: si intentas devolver una referencia a una variable local, no compila. No
existe en Python ni en Go (el GC mantiene vivo lo referenciado), y evita para siempre los
punteros colgantes y los *use-after-free*.

También en `reposfile::match_top_key`, que devuelve un trozo de la línea recibida. En esta
versión, al usar crates que ya devuelven los tipos adecuados, casi no hay que escribir
`'a` a mano.

## 11. Iteradores y closures

Los iteradores son perezosos y componibles; el compilador los optimiza hasta igualar un bucle
manual. Equivalen a las comprensiones de Python y a los `for` de Go:

```rust
let missing: Vec<String> = required
    .into_iter()
    .filter(|p| !p.exists())            // |p| ... es una closure (lambda)
    .map(|p| p.display().to_string())
    .collect();                         // consume el iterador y construye el Vec
```

- `collect()` decide el tipo de resultado por la anotación (`Vec<String>`, `String`...).
- `enumerate()`, `zip()`, `skip()`, `take_while()`, `any()`, `position()`, `max()`... evitan la
  mayoría de bucles con índice. Clippy avisa cuando un `for i in 0..n` debería ser un
  iterador (y lo hizo en la primera versión de este proyecto).
- Una closure puede capturar variables del entorno: `let get = |key: &str| map.and_then(|m| m.get(key));`.

## 12. `let else`, `if let` y let chains

Formas para desestructurar sin `match` completo:

```rust
let Some((key, rest)) = text.split_once(':') else {   // let-else: sale si no encaja
    return Err(...);
};

if let Ok(sha) = run_git(...) { return Ok(sha); }     // if let

if let Some(last) = lines.last_mut()                  // let chains (edición 2024)
    && !last.ends_with('\n')
{
    last.push('\n');
}
```

`let else` es el equivalente de Go de `x, ok := ...; if !ok { return }`, pero el valor solo
existe donde es válido.

## 13. Formato y macros

`println!`, `format!`, `writeln!`, `assert!`, `vec!` terminan en `!`: son **macros** (código
que se expande en compilación), no funciones. `format!` se comprueba en compilación:

```rust
format!("{name:<width$}  {date}")     // variables inline; `<` alinea; width$ = ancho dinámico
```

Python usa f-strings; Go `Sprintf` con verbos en tiempo de ejecución (un `%d` con un string
solo falla al ejecutar). En Rust, un tipo que no implemente `Display` en un `{}` no compila.

## 14. Entrada/salida: `dyn Write`

Igual que en Go con `io.Writer`, las funciones de salida reciben dónde escribir:

```rust
pub fn freeze(p: &Project, requested: &[String], dry_run: bool, out: &mut dyn Write) -> Result<()>
writeln!(out, "Actualizados:")?;
```

En producción `out` es `io::stdout()`; en los tests, un `Vec<u8>` (que implementa `Write`).
Los tests de la CLI capturan stdout y stderr sin tocar el proceso.

Una diferencia: la salida de procesos y ficheros son **bytes**; convertir a texto es explícito
(`String::from_utf8_lossy`, `fs::read_to_string` falla si no es UTF-8 válido).

## 15. RAII y `Drop`

Cuando un valor sale de ámbito, Rust ejecuta su `Drop` automáticamente. Sirve para cerrar
ficheros (no hay `with` ni `defer`: un `File` se cierra solo) y para limpiar recursos:

```rust
impl Drop for MiRecurso {
    fn drop(&mut self) { /* liberar */ }
}
```

El crate `tempfile` lo usa para los directorios temporales de los tests:

```rust
let dir = tempfile::tempdir().unwrap();   // se borra al salir de ámbito...
```

...aunque el test falle (un `panic` también ejecuta los `Drop`). El equivalente en Go es
`t.TempDir()` y en Python el fixture `tmp_path`. La primera versión de este proyecto
implementaba su propio `TempDir` con `Drop` en 20 líneas; con `tempfile` son cero.

## 16. Crates usados (y por qué no otros)

Rust tiene una biblioteca estándar deliberadamente pequeña; lo demás son **crates**
(<https://crates.io>). Antes de añadir uno conviene preguntarse si compensa: cada dependencia
es código ajeno que compilar, actualizar y auditar. Aquí compensan siete:

| Crate | Sustituye | Equivale en Python / Go | Notas |
|---|---|---|---|
| `clap` (derive) | Bucle manual de argumentos (~70 líneas) | `argparse` / `cobra` | `--help`, `--version`, errores de uso y códigos de salida gratis |
| `serde` + `serde_yaml_ng` | Parser YAML casero (~230 líneas) | `PyYAML` + dataclasses | Soporta anclas y estilo flow; deserializa a estructuras tipadas. `serde_yaml` está abandonado, `serde_yaml_ng` es su fork mantenido |
| `regex` | 3 funciones que imitaban regex | `re` / `regexp` | Misma sintaxis que RE2 de Go (sin lookahead) |
| `similar` | LCS y formato unificado propios (~140 líneas) | `difflib` | Un diff unificado en 5 líneas |
| `chrono` | Llamar a `date` y un algoritmo de calendario | `datetime` / `time` | `default-features = false, features = ["clock"]` evita arrastrar dependencias de más |
| `anyhow` | Tipo `Error` propio y `map_err` | excepciones / `fmt.Errorf("%w")` | Contexto en cada capa |
| `tempfile` (solo tests) | `TempDir` propio con `Drop` | `tmp_path` / `t.TempDir()` | |

**Lo que NO se sustituyó, a propósito:**

- **La edición de `repos.yaml`.** Ningún crate YAML conserva comentarios y formato al
  reescribir (`serde_yaml_ng` los pierde). Por eso serde solo **lee**, y los cambios se hacen
  línea a línea como en las otras versiones.
- **Llamar al binario `git`** en vez de usar `git2` (libgit2) o `gix`. Se necesitan tres
  comandos; con `git2` habría que compilar C, y el comportamiento (config, `safe.directory`,
  formato de merges) dejaría de ser el de tu git.

**Lo que se paga:**

| | Solo `std` | Con crates |
|---|---|---|
| Código (sin tests) | ~1.440 líneas | ~980 líneas |
| Binario release | 458 KB | 2,7 MB |
| Compilación release limpia | ~3 s | ~25 s |
| Crates en el árbol | 0 | 53 (7 directos) |

`regex`, `chrono`, `clap` y `serde` son grandes. Se puede reducir con `opt-level = "z"` o
quitando features, pero para una herramienta personal 2,7 MB da igual. La diferencia real
está en la **superficie a mantener**: antes eran 1.440 líneas propias; ahora son 980 más
53 crates que actualizar (`cargo update`) y auditar (`cargo audit`).

## 17. Tests

- `#[test]` marca una función de test; `#[cfg(test)] mod tests { ... }` en el **mismo fichero**
  que el código se compila solo con `cargo test`. Así los tests unitarios acceden a lo privado.
- Los **tests de integración** viven en `tests/`; cada fichero es un crate aparte que usa la
  API pública (como un usuario). Ver [tests/freeze.rs](../tests/freeze.rs).
- `tests/common/mod.rs` (subcarpeta) es el sitio de los helpers compartidos: no se ejecuta como
  test.
- `assert!`, `assert_eq!(izquierda, derecha, "mensaje")` hacen `panic` si fallan, y el
  ejecutor marca el test como fallido. No hay `t.Errorf` que siga: el primer fallo para el test.
- No hay *tests de tabla* integrados ni `parametrize`: se hace con un array de casos y un bucle
  (ver `odoo_version_quoted_and_bare` en [config.rs](../src/config.rs)); con un `assert_eq!`
  que incluya el nombre del caso en el mensaje.
- Saltar un test si falta `git`: devolver temprano con `let Some(fx) = new_project() else { return };`
  (no hay `t.Skip`; el test cuenta como pasado).
- Los comentarios `///` generan documentación y los ejemplos de código en ellos se ejecutan
  como **doctests**.

## 18. Compilar y distribuir

- `cargo build --release` produce un ejecutable nativo: aquí 2,7 MB con `strip = true` y
  `lto = true` (458 KB solo con la biblioteca estándar; la versión Go pesa 3,5 MB porque
  incluye su runtime y recolector). Es estático salvo la libc del sistema.
- Compilar es **notablemente más lento** que en Go: la primera compilación en release tarda
  ~25 s porque compila las 53 dependencias; luego, solo se recompila tu código. En
  desarrollo (`cargo build`) es bastante más rápido.
- Compilación cruzada: `rustup target add aarch64-unknown-linux-gnu` y
  `cargo build --release --target aarch64-unknown-linux-gnu` (hace falta el linker de destino;
  con `musl` se obtiene un binario totalmente estático).
- En `Cargo.toml`, `[profile.release]` con `lto = true` y `strip = true` reduce el tamaño.

## 19. Python vs Go vs Rust: tabla resumen

| Aspecto | Python | Go | Rust |
|---|---|---|---|
| Tipado | Dinámico | Estático, simple | Estático, expresivo (enums, traits, genéricos) |
| Memoria | GC | GC | Propiedad + préstamos, sin GC |
| Errores | Excepciones | `(valor, error)` | `Result` / `Option` + `?` |
| Valores nulos | `None` en todas partes | `nil` | No existe; `Option<T>` |
| Visibilidad | Todo público (`_` por convención) | Mayúscula = público | Privado salvo `pub` |
| Compartir datos mutables | Libre | Slices/punteros compartidos | Prohibido por el compilador (`&mut` exclusivo) |
| Texto | `str` (Unicode) | `string` de bytes | `String`/`&str`, UTF-8 validado |
| Módulos | Cada fichero | Cada carpeta = paquete | Árbol declarado con `mod` |
| Biblioteca estándar | Enorme | Grande (regex, flag, http...) | Pequeña (crates para casi todo; `cargo add`) |
| Concurrencia | GIL | goroutines | hilos/async con garantía de no data races |
| Compilación | — | Muy rápida | Lenta |
| Tamaño del binario (esta herramienta) | n/a | 3,5 MB | 458 KB (solo `std`) / 2,7 MB (con crates) |
| Arranque (esta herramienta, `list`) | ~55 ms | ~5,6 ms | ~5 ms (domina el `git` que lanza) |
| Líneas (sin tests, aprox.) | 572 (solo `std`) | ~1.200 (solo `std`) | ~1.440 solo `std` / ~980 con crates |
