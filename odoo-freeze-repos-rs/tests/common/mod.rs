//! Helpers compartidos por los tests de integración.
//! `tests/common/mod.rs` (en subcarpeta) no se compila como un test más: se incluye con
//! `mod common;` desde cada fichero de `tests/`.
#![allow(dead_code)] // cada test usa solo una parte de los helpers

use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;

use tempfile::TempDir;

pub const COMMON_YAML: &str = "services:
  odoo:
    build:
      args:
        ODOO_VERSION: \"16.0\"
";

pub const REPOS_YAML: &str = "./odoo:
  defaults:
    depth: $DEPTH_MERGE
  remotes:
    origin: https://example.com/odoo.git
  target: origin $ODOO_VERSION
  merges:
    - origin $ODOO_VERSION
";

pub fn git_available() -> bool {
    Command::new("git")
        .arg("--version")
        .output()
        .is_ok_and(|o| o.status.success())
}

/// Ejecuta git en `dir` (con identidad y firma desactivada) y falla el test si hay error.
pub fn git(dir: &Path, args: &[&str]) -> String {
    let out = Command::new("git")
        .args(["-C"])
        .arg(dir)
        .args([
            "-c",
            "user.name=t",
            "-c",
            "user.email=t@t",
            "-c",
            "commit.gpgsign=false",
        ])
        .args(args)
        .output()
        .unwrap();
    assert!(
        out.status.success(),
        "git {args:?}: {}",
        String::from_utf8_lossy(&out.stderr)
    );
    String::from_utf8_lossy(&out.stdout).trim().to_string()
}

pub fn write(path: &Path, content: &str) {
    fs::create_dir_all(path.parent().unwrap()).unwrap();
    fs::write(path, content).unwrap();
}

pub fn read(path: &Path) -> String {
    fs::read_to_string(path).unwrap()
}

/// Crea un repo con un commit y la rama remota simulada origin/16.0. Devuelve el SHA.
pub fn new_repo(dir: &Path, url: &str) -> String {
    fs::create_dir_all(dir).unwrap();
    git(dir, &["init", "-q"]);
    git(dir, &["commit", "-q", "--allow-empty", "-m", "init"]);
    git(dir, &["remote", "add", "origin", url]);
    git(dir, &["update-ref", "refs/remotes/origin/16.0", "HEAD"]);
    git(dir, &["rev-parse", "HEAD"])
}

pub struct Fixture {
    pub dir: TempDir,
    pub odoo_sha: String,
    pub extra_sha: String,
}

impl Fixture {
    pub fn root(&self) -> &str {
        self.dir.path().to_str().unwrap()
    }

    pub fn repos_yaml(&self) -> PathBuf {
        self.dir.path().join("odoo/custom/src/repos.yaml")
    }
}

/// Proyecto con 'odoo' (en repos.yaml) y 'extra' (implícito). `None` si falta git
/// (el test se da por saltado).
pub fn new_project() -> Option<Fixture> {
    if !git_available() {
        eprintln!("git no esta instalado: test saltado");
        return None;
    }
    let dir = tempfile::tempdir().unwrap(); // se borra solo al salir de ámbito
    let root = dir.path();
    write(&root.join("common.yaml"), COMMON_YAML);
    write(&root.join("odoo/custom/src/repos.yaml"), REPOS_YAML);
    let odoo_sha = new_repo(
        &root.join("odoo/custom/src/odoo"),
        "https://example.com/odoo.git",
    );
    let extra_sha = new_repo(
        &root.join("odoo/custom/src/extra"),
        "https://example.com/extra.git",
    );
    Some(Fixture {
        dir,
        odoo_sha,
        extra_sha,
    })
}
