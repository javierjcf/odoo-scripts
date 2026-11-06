//! Interfaz de línea de comandos, definida con `clap` (derive): la propia estructura de datos
//! describe los argumentos, y clap genera el parseo, `--help`, `--version` y los errores de uso.

use std::ffi::OsString;
use std::io::Write;
use std::path::PathBuf;

use clap::{Args, Parser, Subcommand};

use crate::freezer;
use crate::project::Project;

/// Congela y descongela el repos.yaml de un proyecto Doodba a los commits que ya hay en disco.
#[derive(Parser)]
#[command(name = "odoo-freeze-repos", version)]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

#[derive(Subcommand)]
enum Command {
    /// Sustituye cada ref de 'merges' por su SHA (la ref original queda en '#frozen:<ref>')
    Freeze(EditArgs),
    /// Devuelve repos.yaml a sus refs flotantes
    Unfreeze(EditArgs),
    /// Muestra fecha, hash y mensaje del commit actual de cada repo
    List(TargetArgs),
}

#[derive(Args)]
struct TargetArgs {
    /// Raíz del proyecto Doodba (donde están common.yaml y odoo/)
    root: PathBuf,
    /// Carpetas de odoo/custom/src a tratar, separadas por comas o espacios (por defecto: todas)
    #[arg(value_delimiter = ',')]
    repos: Vec<String>,
}

#[derive(Args)]
struct EditArgs {
    #[command(flatten)]
    target: TargetArgs,
    /// Muestra el diff de repos.yaml sin escribir nada
    #[arg(short = 'n', long)]
    dry_run: bool,
}

/// Lógica completa de la CLI. Devuelve el código de salida.
/// Recibe los writers para poder probarla sin tocar stdout/stderr reales.
pub fn run<I, T>(args: I, out: &mut dyn Write, err: &mut dyn Write) -> u8
where
    I: IntoIterator<Item = T>,
    T: Into<OsString> + Clone,
{
    let cli = match Cli::try_parse_from(args) {
        Ok(cli) => cli,
        Err(e) => {
            // --help y --version salen con 0 por stdout; los errores de uso, con 2 por stderr.
            let text = e.render().to_string();
            let _ = if e.use_stderr() {
                write!(err, "{text}")
            } else {
                write!(out, "{text}")
            };
            return e.exit_code() as u8;
        }
    };
    match execute(cli, out) {
        Ok(()) => 0,
        Err(e) => {
            let _ = writeln!(err, "error: {e:#}");
            1
        }
    }
}

fn execute(cli: Cli, out: &mut dyn Write) -> anyhow::Result<()> {
    let open = |root: &PathBuf, need_common: bool| -> anyhow::Result<Project> {
        let project = Project::new(root)?;
        project.validate(need_common)?;
        Ok(project)
    };
    match cli.command {
        Command::Freeze(a) => freezer::freeze(
            &open(&a.target.root, true)?,
            &a.target.repos,
            a.dry_run,
            out,
        ),
        Command::Unfreeze(a) => freezer::unfreeze(
            &open(&a.target.root, false)?,
            &a.target.repos,
            a.dry_run,
            out,
        ),
        Command::List(a) => freezer::list(&open(&a.root, false)?, &a.repos, out),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn run_args(args: &[&str]) -> (u8, String, String) {
        let (mut out, mut err) = (Vec::new(), Vec::new());
        let code = run(args.iter().copied(), &mut out, &mut err);
        (
            code,
            String::from_utf8(out).unwrap(),
            String::from_utf8(err).unwrap(),
        )
    }

    #[test]
    fn usage_errors_exit_with_2() {
        assert_eq!(run_args(&["odoo-freeze-repos"]).0, 2);
        assert_eq!(run_args(&["odoo-freeze-repos", "nope", "x"]).0, 2);
        assert_eq!(run_args(&["odoo-freeze-repos", "freeze"]).0, 2);
        assert_eq!(
            run_args(&["odoo-freeze-repos", "freeze", "--raro", "x"]).0,
            2
        );
        assert_eq!(run_args(&["odoo-freeze-repos", "list", "-n", "x"]).0, 2); // list no tiene --dry-run
    }

    #[test]
    fn help_and_version_go_to_stdout() {
        let (code, out, _) = run_args(&["odoo-freeze-repos", "--help"]);
        assert_eq!(code, 0);
        assert!(
            out.contains("freeze") && out.contains("unfreeze") && out.contains("list"),
            "{out}"
        );
        let (code, out, _) = run_args(&["odoo-freeze-repos", "--version"]);
        assert_eq!(code, 0);
        assert!(out.starts_with("odoo-freeze-repos "), "{out}");
    }

    #[test]
    fn repos_accept_commas_and_spaces() {
        let cli =
            Cli::try_parse_from(["odoo-freeze-repos", "freeze", "/p", "a,b", "c", "-n"]).unwrap();
        let Command::Freeze(args) = cli.command else {
            panic!("se esperaba freeze")
        };
        assert_eq!(args.target.repos, ["a", "b", "c"]);
        assert!(args.dry_run);
    }
}
