//! Llamadas a la CLI de git y resolución de refs a commits.
//!
//! Se ejecuta el binario `git` en vez de usar `git2`/`gix`: la herramienta solo necesita
//! tres comandos, y así el comportamiento es exactamente el de tu git (config, safe.directory...)
//! sin compilar libgit2 ni arrastrar cientos de dependencias.

use std::path::Path;
use std::process::Command;

use anyhow::{Context, Result, anyhow, bail};

/// Ejecuta `git -C repo args...` y devuelve la salida sin espacios finales.
pub fn run_git(repo: &Path, args: &[&str]) -> Result<String> {
    let output = Command::new("git")
        .arg("-C")
        .arg(repo)
        .args(args)
        .output()
        .context("no se pudo ejecutar git")?;
    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        bail!("git {}: {}", args.join(" "), stderr.trim());
    }
    Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

/// Indica si `s` es un hash completo de 40 caracteres hexadecimales.
pub fn is_sha(s: &str) -> bool {
    s.len() == 40 && s.chars().all(|c| c.is_ascii_hexdigit())
}

/// Resuelve (remote, ref) al commit que representa hoy, sin red.
///
/// Dos estrategias, en orden:
/// 1. Rama normal ya trackeada localmente (`<remote>/<rama>`): exacta y sin ambigüedad.
/// 2. Ref puntual (PR, etc.) que git-aggregator fusionó en su día: se busca el merge
///    commit cuyo mensaje menciona esa ref exacta ("Merge commit '<ref>' of ... into
///    ...") y se toma su segundo padre. Se usa el más reciente si hay varios.
///
/// (`ref` es palabra reservada en Rust, de ahí el nombre `git_ref`.)
pub fn resolve_ref(repo: &Path, remote: &str, git_ref: &str, odoo_version: &str) -> Result<String> {
    let branch = git_ref.replace("$ODOO_VERSION", odoo_version);
    let spec = format!("{remote}/{branch}");
    if let Ok(sha) = run_git(repo, &["rev-parse", "--verify", "--quiet", &spec]) {
        return Ok(sha);
    }

    let log = run_git(
        repo,
        &["log", "--first-parent", "--merges", "--format=%H\t%s"],
    )
    .context("error ejecutando git log")?;
    let needle = format!("'{git_ref}'");
    for line in log.lines() {
        let (sha, subject) = line.split_once('\t').unwrap_or((line, ""));
        if subject.contains(&needle) {
            return run_git(repo, &["rev-parse", &format!("{sha}^2")]).with_context(|| {
                format!(
                    "no se pudo resolver el segundo padre de {}",
                    &sha[..10.min(sha.len())]
                )
            });
        }
    }
    Err(anyhow!(
        "no se encontro ningun merge commit que mencione '{git_ref}'"
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn is_sha_cases() {
        assert!(is_sha(&"a".repeat(40)));
        assert!(is_sha(&"A".repeat(40)));
        assert!(!is_sha(&"g".repeat(40)));
        assert!(!is_sha("abc"));
        assert!(!is_sha("$ODOO_VERSION"));
    }
}
