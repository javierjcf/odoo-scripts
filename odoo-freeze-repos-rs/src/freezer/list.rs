use std::io::Write;

use anyhow::Result;

use super::validate_targets;
use crate::gitx::run_git;
use crate::project::Project;

/// Muestra fecha, hash y mensaje del commit actual de cada repo.
pub fn list(p: &Project, requested: &[String], out: &mut dyn Write) -> Result<()> {
    let all = p.discover_repos()?;
    let targets = validate_targets(requested, &all);
    if targets.is_empty() {
        writeln!(out, "Nada que listar.")?;
        return Ok(());
    }

    let width = targets.iter().map(|n| n.len()).max().unwrap_or(0);
    for name in &targets {
        match run_git(
            &p.src.join(name),
            &["log", "-1", "--date=short", "--format=%cd\t%h\t%s"],
        ) {
            Ok(info) => {
                // splitn(3): con mensaje vacío el trim quita el tab final y salen 2 trozos.
                let mut parts = info.splitn(3, '\t');
                let date = parts.next().unwrap_or("");
                let short = parts.next().unwrap_or("");
                let subject = parts.next().unwrap_or("");
                writeln!(out, "{name:<width$}  {date}  {short}  {subject}")?;
            }
            Err(e) => writeln!(out, "{name:<width$}  ?  ?  [error: {e:#}]")?,
        }
    }
    Ok(())
}
