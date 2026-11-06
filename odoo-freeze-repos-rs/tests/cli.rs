mod common;

use common::{REPOS_YAML, new_project, read};
use odoo_freeze_repos::cli::run;

fn run_args(args: &[&str]) -> (u8, String, String) {
    let args = std::iter::once("odoo-freeze-repos").chain(args.iter().copied());
    let (mut out, mut err) = (Vec::new(), Vec::new());
    let code = run(args, &mut out, &mut err);
    (
        code,
        String::from_utf8(out).unwrap(),
        String::from_utf8(err).unwrap(),
    )
}

#[test]
fn flags_can_follow_the_path() {
    let Some(fx) = new_project() else { return };
    let (code, _, err) = run_args(&["freeze", fx.root(), "-n"]);
    assert_eq!(code, 0, "{err}");
    assert_eq!(read(&fx.repos_yaml()), REPOS_YAML, "-n modifico repos.yaml");
}

#[test]
fn list_shows_commit_subject() {
    let Some(fx) = new_project() else { return };
    let (code, out, err) = run_args(&["list", fx.root()]);
    assert_eq!(code, 0, "{err}");
    assert!(out.contains("init"), "{out}");
}

#[test]
fn invalid_project_exits_with_1() {
    let dir = tempfile::tempdir().unwrap();
    let (code, _, err) = run_args(&["list", dir.path().to_str().unwrap()]);
    assert_eq!(code, 1);
    assert!(err.contains("no parece un proyecto Doodba"), "{err}");
}
