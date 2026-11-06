mod common;

use common::{REPOS_YAML, new_project, read, write};
use odoo_freeze_repos::freezer::{freeze, unfreeze};
use odoo_freeze_repos::project::Project;
use odoo_freeze_repos::reposfile::ADDED_MARKER;

fn output(buf: Vec<u8>) -> String {
    String::from_utf8(buf).unwrap()
}

#[test]
fn freeze_unfreeze_round_trip() {
    let Some(fx) = new_project() else { return };
    let p = Project::new(fx.dir.path()).unwrap();
    let mut out = Vec::new();

    freeze(&p, &[], false, &mut out).unwrap();
    let frozen = read(&p.repos_yaml);
    for want in [
        "# Congelado con odoo-freeze-repos el ".to_string(),
        format!("    - origin {}  #frozen:$ODOO_VERSION\n", fx.odoo_sha),
        ADDED_MARKER.to_string(),
        "extra:\n".to_string(),
        "    origin: https://example.com/extra.git\n".to_string(),
        format!("    - origin {}  #frozen:$ODOO_VERSION\n", fx.extra_sha),
    ] {
        assert!(frozen.contains(&want), "falta {want:?} en:\n{frozen}");
    }
    assert_eq!(
        read(&p.backup_yaml),
        REPOS_YAML,
        "bck-repos.yaml deberia ser el original"
    );

    // Un segundo freeze no debe sobrescribir el backup ni cambiar nada.
    let mut out = Vec::new();
    freeze(&p, &[], false, &mut out).unwrap();
    assert!(output(out).contains("Nada que congelar"));

    // unfreeze debe devolver el fichero EXACTAMENTE al original.
    let mut out = Vec::new();
    unfreeze(&p, &[], false, &mut out).unwrap();
    assert_eq!(read(&p.repos_yaml), REPOS_YAML);
    assert_eq!(
        read(&p.frozen_yaml),
        frozen,
        "frozen-repos.yaml deberia contener el estado congelado"
    );
}

#[test]
fn dry_run_does_not_write() {
    let Some(fx) = new_project() else { return };
    let p = Project::new(fx.dir.path()).unwrap();
    let mut out = Vec::new();

    freeze(&p, &[], true, &mut out).unwrap();
    assert_eq!(
        read(&p.repos_yaml),
        REPOS_YAML,
        "--dry-run modifico repos.yaml"
    );
    assert!(!p.backup_yaml.exists(), "--dry-run creo el backup");
    let text = output(out);
    assert!(
        text.contains(&format!(
            "+    - origin {}  #frozen:$ODOO_VERSION",
            fx.odoo_sha
        )),
        "el diff no muestra el cambio:\n{text}"
    );
}

#[test]
fn freeze_specific_repo_and_unresolvable_ref() {
    let Some(fx) = new_project() else { return };
    let p = Project::new(fx.dir.path()).unwrap();
    write(
        &p.repos_yaml,
        &REPOS_YAML.replace(
            "    - origin $ODOO_VERSION\n",
            "    - origin $ODOO_VERSION\n    - origin refs/pull/9/head\n",
        ),
    );
    let mut out = Vec::new();
    freeze(&p, &["odoo".to_string()], false, &mut out).unwrap();

    let got = read(&p.repos_yaml);
    assert!(
        got.contains(&format!("origin {}  #frozen:$ODOO_VERSION", fx.odoo_sha)),
        "{got}"
    );
    assert!(
        got.contains("    - origin refs/pull/9/head\n"),
        "la ref sin resolver deberia quedar intacta"
    );
    assert!(!got.contains("extra:"), "extra no se pidio");
    assert!(output(out).contains("Omitidos:"));
}
