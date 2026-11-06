use std::io::Write;

use anyhow::Result;

use super::{resolve_merges, short_refs, validate_targets};
use crate::config::{self, RepoEntry};
use crate::gitx;
use crate::project::Project;
use crate::reposfile as rf;

/// Resultado de congelar un repo (para el resumen final).
struct Frozen {
    name: String,
    resolved: Vec<String>,
}

/// Fija repos.yaml a los commits actuales de cada repo (o solo de `requested`).
/// Con `dry_run` solo imprime el diff. Guarda bck-repos.yaml si no existe.
pub fn freeze(p: &Project, requested: &[String], dry_run: bool, out: &mut dyn Write) -> Result<()> {
    let all = p.discover_repos()?;
    let targets = validate_targets(requested, &all);
    if targets.is_empty() {
        writeln!(out, "Nada que congelar.")?;
        return Ok(());
    }

    let data = config::load_repos(&p.repos_yaml)?;
    let mut lines = rf::read_lines(&p.repos_yaml)?;
    let original = lines.clone();
    let odoo_version = config::odoo_version(&p.common_yaml)?;

    let mut added: Vec<Frozen> = Vec::new();
    let mut changed: Vec<Frozen> = Vec::new();
    let mut skipped: Vec<String> = Vec::new();

    for name in &targets {
        let repo_path = p.src.join(name);
        let key = rf::repo_key(&data, name);

        // `fallback` es el dueño de la entrada cuando no viene de `data` (se inicializa, como
        // mucho, una vez); `entry` solo la presta.
        let fallback: RepoEntry;
        let entry: &RepoEntry = match key {
            Some(k) => match data.get(k).and_then(Option::as_ref) {
                Some(entry) => entry,
                None => {
                    fallback = RepoEntry::default(); // clave sin contenido
                    &fallback
                }
            },
            None => match gitx::run_git(&repo_path, &["config", "--get", "remote.origin.url"]) {
                Ok(url) => {
                    fallback = RepoEntry::synthesized(url);
                    &fallback
                }
                Err(e) => {
                    skipped.push(format!("{name}: no se pudo leer remote.origin.url: {e:#}"));
                    continue;
                }
            },
        };

        if entry.merges().is_empty() {
            skipped.push(format!("{name}: sin 'merges' definido"));
            continue;
        }

        let (resolved, warnings) = resolve_merges(name, &repo_path, entry.merges(), &odoo_version);
        skipped.extend(warnings);
        if resolved == entry.merges() {
            continue; // nada resuelto (o ya estaba todo fijado)
        }

        if key.is_none() {
            let sha = rf::split_merge(&resolved[0]).map_or("", |(_, r)| r);
            rf::append_new_block(&mut lines, name, &entry.url, sha);
            added.push(Frozen {
                name: name.clone(),
                resolved,
            });
            continue;
        }

        let Some((start, end)) = rf::find_block(&lines, name) else {
            skipped.push(format!("{name}: no se localizo su bloque en repos.yaml"));
            continue;
        };
        // `block` es un préstamo mutable de una parte de `lines`. Mientras exista, el
        // compilador prohíbe tocar `lines` por otro camino.
        let block = &mut lines[start..end];
        for (old_value, new_value) in entry.merges().iter().zip(&resolved) {
            if old_value == new_value {
                continue;
            }
            let Some(idx) = rf::find_merge_line(block, old_value) else {
                skipped.push(format!(
                    "{name}: no se localizo la linea de merge '{old_value}', no se toca"
                ));
                continue;
            };
            let old_ref = rf::split_merge(old_value).map_or("", |(_, r)| r);
            block[idx] = block[idx].replacen(
                old_value.as_str(),
                &format!("{new_value}  #frozen:{old_ref}"),
                1,
            );
        }
        changed.push(Frozen {
            name: name.clone(),
            resolved,
        });
    }

    if added.is_empty() && changed.is_empty() {
        writeln!(
            out,
            "Nada que congelar (todo ya estaba fijado o no se pudo resolver con seguridad)."
        )?;
        if !skipped.is_empty() {
            writeln!(out, "\nAvisos:")?;
            for msg in &skipped {
                writeln!(out, "  - {msg}")?;
            }
        }
        return Ok(());
    }

    rf::set_stamp(
        &mut lines,
        &chrono::Local::now().format("%Y-%m-%d %H:%M").to_string(),
    );

    let mut backup_created = false;
    if dry_run {
        let rel = p.rel(&p.repos_yaml);
        writeln!(
            out,
            "[dry-run] No se escribe nada. Cambios que se aplicarian a repos.yaml:\n"
        )?;
        write!(
            out,
            "{}",
            super::unified_diff(&original, &lines, &rel, &format!("{rel} (congelado)"))
        )?;
    } else {
        backup_created = !p.backup_yaml.exists();
        if backup_created {
            rf::copy_file(&p.repos_yaml, &p.backup_yaml)?;
        }
        rf::write_lines(&p.repos_yaml, &lines)?;
    }

    if !changed.is_empty() {
        writeln!(out, "\nActualizados:")?;
        for r in &changed {
            writeln!(out, "  {}: {}", r.name, short_refs(&r.resolved))?;
        }
    }
    if !added.is_empty() {
        writeln!(out, "\nNuevos (antes implicitos via addons.yaml):")?;
        for r in &added {
            writeln!(out, "  {}: {}", r.name, short_refs(&r.resolved))?;
        }
    }
    if !skipped.is_empty() {
        writeln!(out, "\nOmitidos:")?;
        for msg in &skipped {
            writeln!(out, "  {msg}")?;
        }
    }
    let backup = p.rel(&p.backup_yaml);
    if dry_run {
        // ya se ha avisado arriba
    } else if backup_created {
        writeln!(out, "\nrepos.yaml congelado; original guardado en {backup}")?;
    } else {
        writeln!(
            out,
            "\nrepos.yaml congelado; el original ya estaba en {backup} (no se toca)"
        )?;
    }
    Ok(())
}
