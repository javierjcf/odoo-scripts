use std::io::Write;

use anyhow::Result;
use regex::Regex;

use super::unified_diff;
use crate::config::{self, RepoEntry};
use crate::gitx::is_sha;
use crate::project::Project;
use crate::reposfile as rf;

/// Devuelve repos.yaml a sus refs flotantes. Con `dry_run` solo imprime el diff.
/// Copia el estado congelado a frozen-repos.yaml antes de tocar nada.
pub fn unfreeze(
    p: &Project,
    requested: &[String],
    dry_run: bool,
    out: &mut dyn Write,
) -> Result<()> {
    let data = config::load_repos(&p.repos_yaml)?;
    let mut lines = rf::read_lines(&p.repos_yaml)?;
    let original = lines.clone();

    let targets: Vec<String> = if requested.is_empty() {
        let mut all: Vec<String> = data
            .keys()
            .map(|k| rf::normalize_key(k).to_string())
            .collect();
        all.sort();
        all
    } else {
        requested.to_vec()
    };

    let marker_idx = rf::find_added_marker(&lines);
    let mut restored: Vec<String> = Vec::new();
    let mut removed: Vec<String> = Vec::new();
    let mut skipped: Vec<String> = Vec::new();

    for name in &targets {
        let range =
            rf::repo_key(&data, name).and_then(|k| rf::find_block(&lines, name).map(|r| (k, r)));
        let Some((key, (start, end))) = range else {
            skipped.push(format!("{name}: sin entrada en repos.yaml"));
            continue;
        };

        // Repos añadidos por freeze (tras la marca): se eliminan, no se descongelan.
        if marker_idx.is_some_and(|m| start > m) {
            lines.drain(start..end);
            removed.push(name.clone());
            continue;
        }

        let block = &mut lines[start..end];
        let empty = RepoEntry::default();
        let entry = data.get(key).and_then(Option::as_ref).unwrap_or(&empty);
        let mut touched = false;

        for (i, value) in entry.merges().iter().enumerate() {
            let Some((remote, git_ref)) = rf::split_merge(value) else {
                continue;
            };
            if !is_sha(git_ref) {
                continue;
            }
            let Some(idx) = rf::find_merge_line(block, value) else {
                continue;
            };

            // `<valor> #frozen:<ref original>`
            let marker = Regex::new(&format!(r"{}\s+#frozen:(\S+)", regex::escape(value)))?;
            if let Some(caps) = marker.captures(&block[idx]) {
                let span = caps.get(0).expect("el grupo 0 siempre existe");
                let new_line = format!(
                    "{}{remote} {}{}",
                    &block[idx][..span.start()],
                    &caps[1],
                    &block[idx][span.end()..]
                );
                block[idx] = new_line;
                touched = true;
            } else if i == 0 && !entry.target().is_empty() {
                // SHA sin marca (fijado a mano): la base es siempre 'target'.
                block[idx] = block[idx].replacen(value.as_str(), entry.target(), 1);
                touched = true;
            } else {
                skipped.push(format!(
                    "{name}: '{value}' sin ref original, restauralo a mano"
                ));
            }
        }
        if touched {
            restored.push(name.clone());
        }
    }

    // Si ya no queda ningún repo añadido, se quita también la marca y el hueco que dejó freeze.
    if let Some(m) = marker_idx {
        let has_key = lines[m + 1..]
            .iter()
            .any(|l| rf::match_top_key(l).is_some());
        if !has_key {
            lines.truncate(m);
            if lines.last().is_some_and(|l| l.trim().is_empty()) {
                lines.pop();
            }
        }
    }

    if !lines.iter().any(|l| l.contains("#frozen:")) {
        rf::remove_stamp(&mut lines); // ya no queda nada congelado
    }

    if lines == original {
        writeln!(out, "Nada que descongelar.")?;
    } else {
        if dry_run {
            let rel = p.rel(&p.repos_yaml);
            writeln!(
                out,
                "[dry-run] No se escribe nada. Cambios que se aplicarian a repos.yaml:\n"
            )?;
            write!(
                out,
                "{}",
                unified_diff(&original, &lines, &format!("{rel} (congelado)"), &rel)
            )?;
            writeln!(out)?;
        } else {
            rf::copy_file(&p.repos_yaml, &p.frozen_yaml)?;
            rf::write_lines(&p.repos_yaml, &lines)?;
        }
        if !restored.is_empty() {
            writeln!(out, "Descongelados: {}", restored.join(", "))?;
        }
        if !removed.is_empty() {
            writeln!(
                out,
                "Eliminados (anadidos por freeze): {}",
                removed.join(", ")
            )?;
        }
        if !dry_run {
            writeln!(
                out,
                "Estado congelado guardado en {}",
                p.rel(&p.frozen_yaml)
            )?;
            writeln!(
                out,
                "Original (antes del primer freeze) en {}",
                p.rel(&p.backup_yaml)
            )?;
            writeln!(
                out,
                "Ahora ejecuta 'invoke git-aggregate' para traer lo ultimo antes de volver a congelar."
            )?;
        }
    }
    if !skipped.is_empty() {
        writeln!(out, "\nAvisos:")?;
        for msg in &skipped {
            writeln!(out, "  - {msg}")?;
        }
    }
    Ok(())
}
