//! Edición de repos.yaml línea a línea. No se re-serializa el YAML: se modifican solo las
//! líneas necesarias para conservar comentarios, orden y formato del fichero original
//! (ningún crate YAML hace eso: todos pierden los comentarios al escribir).
//!
//! Las funciones que modifican reciben `&mut Vec<String>`: un préstamo mutable EXCLUSIVO.
//! El compilador garantiza que nadie más lee ni escribe esas líneas mientras tanto.

use std::fs;
use std::path::Path;
use std::sync::LazyLock;

use anyhow::{Context, Result};
use regex::Regex;

use crate::config::ReposFile;

/// Devuelve las líneas conservando el salto final de cada una.
pub fn read_lines(path: &Path) -> Result<Vec<String>> {
    let text =
        fs::read_to_string(path).with_context(|| format!("no se pudo leer {}", path.display()))?;
    Ok(text.split_inclusive('\n').map(str::to_string).collect())
}

pub fn write_lines(path: &Path, lines: &[String]) -> Result<()> {
    fs::write(path, lines.concat())
        .with_context(|| format!("no se pudo escribir {}", path.display()))
}

/// Copia src a dst (fs::copy conserva también los permisos).
pub fn copy_file(src: &Path, dst: &Path) -> Result<()> {
    fs::copy(src, dst)
        .with_context(|| format!("no se pudo copiar {} a {}", src.display(), dst.display()))?;
    Ok(())
}

pub fn normalize_key(key: &str) -> &str {
    key.strip_prefix("./").unwrap_or(key)
}

/// Devuelve la clave tal cual aparece en el fichero ("name" o "./name").
pub fn repo_key<'a>(data: &'a ReposFile, name: &str) -> Option<&'a str> {
    data.keys()
        .map(String::as_str)
        .find(|k| normalize_key(k) == name)
}

// `LazyLock` compila la regex la primera vez que se usa (y una sola vez).
// Clave de nivel superior tipo "./odoo:" o "account-reconcile:" (nunca indentada ni comentada).
static TOP_KEY: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"^([^\s#].*):\s*$").unwrap());
// Línea "- <valor>" con comentario opcional detrás; las líneas comentadas no encajan.
static MERGE_LINE: LazyLock<Regex> =
    LazyLock::new(|| Regex::new(r"^\s*-\s+(.+?)\s*(?:#.*)?$").unwrap());

/// Si la línea es una clave de nivel superior, devuelve su texto.
pub fn match_top_key(line: &str) -> Option<&str> {
    let caps = TOP_KEY.captures(line.trim_end_matches(['\r', '\n']))?;
    Some(caps.get(1)?.as_str())
}

/// Devuelve [start, end) de las líneas del bloque de nivel superior de `name`.
pub fn find_block(lines: &[String], name: &str) -> Option<(usize, usize)> {
    let start = lines
        .iter()
        .position(|l| match_top_key(l).is_some_and(|k| normalize_key(k) == name))?;
    let end = lines[start + 1..]
        .iter()
        .position(|l| match_top_key(l).is_some())
        .map_or(lines.len(), |j| start + 1 + j);
    Some((start, end))
}

/// Índice de la línea `- <value>` (con comentario opcional detrás).
pub fn find_merge_line(block: &[String], value: &str) -> Option<usize> {
    block.iter().position(|line| {
        MERGE_LINE
            .captures(line.trim_end_matches(['\r', '\n']))
            .is_some_and(|c| &c[1] == value)
    })
}

/// Separa "remote ref" en sus dos partes.
pub fn split_merge(merge: &str) -> Option<(&str, &str)> {
    let (remote, git_ref) = merge.trim().split_once(char::is_whitespace)?;
    Some((remote, git_ref.trim()))
}

pub const ADDED_MARKER: &str = "# --- A partir de aqui: repos anadidos por odoo-freeze-repos, antes implicitos via addons.yaml ---\n";
/// Marca del nombre anterior del script: se sigue reconociendo en repos.yaml ya congelados.
const LEGACY_ADDED_MARKER: &str = "# --- A partir de aqui: repos anadidos por freeze_repos.py, antes implicitos via addons.yaml ---\n";

/// Índice de la línea marca, si existe.
pub fn find_added_marker(lines: &[String]) -> Option<usize> {
    lines
        .iter()
        .position(|l| l == ADDED_MARKER || l == LEGACY_ADDED_MARKER)
}

/// Inserta el comentario separador una única vez.
fn ensure_added_marker(lines: &mut Vec<String>) {
    if find_added_marker(lines).is_some() {
        return;
    }
    if let Some(last) = lines.last_mut()
        && !last.ends_with('\n')
    {
        last.push('\n');
    }
    lines.push("\n".to_string());
    lines.push(ADDED_MARKER.to_string());
}

pub fn append_new_block(lines: &mut Vec<String>, name: &str, url: &str, sha: &str) {
    ensure_added_marker(lines);
    lines.extend([
        "\n".to_string(),
        format!("{name}:\n"),
        "  defaults:\n".to_string(),
        "    depth: $DEPTH_MERGE\n".to_string(),
        "  remotes:\n".to_string(),
        format!("    origin: {url}\n"),
        "  target: origin $ODOO_VERSION\n".to_string(),
        "  merges:\n".to_string(),
        format!("    - origin {sha}  #frozen:$ODOO_VERSION\n"),
    ]);
}

pub const STAMP_PREFIX: &str = "# Congelado con odoo-freeze-repos el ";
/// Nombre anterior del script.
const LEGACY_STAMP_PREFIX: &str = "# Congelado con freeze_repos.py el ";

fn has_stamp(lines: &[String]) -> bool {
    lines
        .first()
        .is_some_and(|l| l.starts_with(STAMP_PREFIX) || l.starts_with(LEGACY_STAMP_PREFIX))
}

/// Pone (o actualiza) en la primera línea la fecha y hora del freeze.
pub fn set_stamp(lines: &mut Vec<String>, when: &str) {
    let stamp = format!("{STAMP_PREFIX}{when}\n");
    if has_stamp(lines) {
        lines[0] = stamp;
    } else {
        lines.insert(0, stamp);
    }
}

pub fn remove_stamp(lines: &mut Vec<String>) {
    if has_stamp(lines) {
        lines.remove(0);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn lines(items: &[&str]) -> Vec<String> {
        items.iter().map(|s| s.to_string()).collect()
    }

    #[test]
    fn find_block_ranges() {
        let l = lines(&[
            "# cabecera\n",
            "./odoo:\n",
            "  target: x\n",
            "\n",
            "account:\n",
            "  target: y\n",
        ]);
        assert_eq!(find_block(&l, "odoo"), Some((1, 4)));
        assert_eq!(find_block(&l, "account"), Some((4, 6)));
        assert_eq!(find_block(&l, "nope"), None);
    }

    #[test]
    fn top_key_rules() {
        assert_eq!(match_top_key("./odoo:\n"), Some("./odoo"));
        assert_eq!(match_top_key("web:   \n"), Some("web"));
        assert_eq!(match_top_key("  indented:\n"), None);
        assert_eq!(match_top_key("# comment:\n"), None);
        assert_eq!(match_top_key("key: value\n"), None);
        assert_eq!(match_top_key(":\n"), None);
    }

    #[test]
    fn find_merge_line_cases() {
        let block = lines(&[
            "repo:\n",
            "  merges:\n",
            "    # - origin comentado\n",
            "    - origin $ODOO_VERSION  #frozen:x\n",
            "    - origin 123\n",
        ]);
        assert_eq!(find_merge_line(&block, "origin $ODOO_VERSION"), Some(3));
        assert_eq!(find_merge_line(&block, "origin 123"), Some(4));
        assert_eq!(find_merge_line(&block, "origin comentado"), None);
    }

    #[test]
    fn stamp_round_trip() {
        let mut l = lines(&["a:\n"]);
        set_stamp(&mut l, "2026-03-06 09:05");
        assert_eq!(
            l[0],
            "# Congelado con odoo-freeze-repos el 2026-03-06 09:05\n"
        );
        set_stamp(&mut l, "2026-03-07 10:00"); // actualiza, no duplica
        assert_eq!(l.len(), 2);
        remove_stamp(&mut l);
        assert_eq!(l, lines(&["a:\n"]));
    }

    #[test]
    fn split_merge_cases() {
        assert_eq!(
            split_merge("origin  refs/pull/1/head "),
            Some(("origin", "refs/pull/1/head"))
        );
        assert_eq!(split_merge("solo"), None);
    }

    #[test]
    fn new_block_adds_marker_once() {
        let mut l = lines(&["a:\n"]);
        append_new_block(&mut l, "x", "url", "sha");
        append_new_block(&mut l, "y", "url", "sha");
        assert_eq!(l.iter().filter(|s| *s == ADDED_MARKER).count(), 1);
    }
}
