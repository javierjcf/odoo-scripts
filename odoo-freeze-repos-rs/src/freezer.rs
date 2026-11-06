//! Comandos freeze, unfreeze y list sobre un proyecto Doodba.

mod freeze;
mod list;
mod unfreeze;

pub use freeze::freeze;
pub use list::list;
pub use unfreeze::unfreeze;

use std::path::Path;

use crate::gitx;
use crate::reposfile::split_merge;

/// Resuelve cada entrada de forma independiente. Las que fallen quedan sin tocar.
/// Devuelve (merges resueltos, avisos).
fn resolve_merges(
    name: &str,
    repo_path: &Path,
    merges: &[String],
    odoo_version: &str,
) -> (Vec<String>, Vec<String>) {
    let mut resolved = merges.to_vec(); // copia explícita: Rust nunca copia en silencio
    let mut warnings = Vec::new();
    for (i, m) in merges.iter().enumerate() {
        let Some((remote, git_ref)) = split_merge(m) else {
            warnings.push(format!(
                "{name}: merges[{i}] ('{m}') no tiene formato 'remote ref'"
            ));
            continue;
        };
        if gitx::is_sha(git_ref) {
            continue; // ya fijado a mano
        }
        match gitx::resolve_ref(repo_path, remote, git_ref, odoo_version) {
            Ok(sha) => resolved[i] = format!("{remote} {sha}"),
            // `{e:#}` imprime toda la cadena de contexto de anyhow ("a: b: c")
            Err(e) => warnings.push(format!(
                "{name}: merges[{i}] ('{m}') no se pudo resolver: {e:#}"
            )),
        }
    }
    (resolved, warnings)
}

/// Filtra los repos pedidos contra los existentes; sin petición, devuelve todos.
fn validate_targets(requested: &[String], all: &[String]) -> Vec<String> {
    if requested.is_empty() {
        return all.to_vec();
    }
    let mut valid = Vec::new();
    for name in requested {
        if all.contains(name) {
            valid.push(name.clone());
        } else {
            eprintln!("[WARN] repo desconocido o sin .git: {name}");
        }
    }
    valid
}

/// Los 10 primeros caracteres de la ref de cada merge.
fn short_refs(merges: &[String]) -> String {
    merges
        .iter()
        .map(|m| {
            split_merge(m)
                .map_or("", |(_, r)| r)
                .chars()
                .take(10)
                .collect::<String>()
        })
        .collect::<Vec<_>>()
        .join(", ")
}

/// Diff unificado (3 líneas de contexto) entre dos versiones de repos.yaml, con el crate `similar`.
fn unified_diff(old: &[String], new: &[String], old_name: &str, new_name: &str) -> String {
    let (old, new) = (old.concat(), new.concat());
    similar::TextDiff::from_lines(&old, &new)
        .unified_diff()
        .context_radius(3)
        .header(old_name, new_name)
        .to_string()
}
