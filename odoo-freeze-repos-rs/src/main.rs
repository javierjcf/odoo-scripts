use std::io;
use std::process::ExitCode;

fn main() -> ExitCode {
    ExitCode::from(odoo_freeze_repos::cli::run(
        std::env::args_os(),
        &mut io::stdout(),
        &mut io::stderr(),
    ))
}
