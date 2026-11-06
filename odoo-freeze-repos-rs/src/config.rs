//! Lectura de repos.yaml y common.yaml con `serde`.
//!
//! `#[derive(Deserialize)]` genera el código que convierte el YAML en estas estructuras: los
//! campos que no declaramos (`defaults`, `remotes`...) se ignoran, y el YAML real (anclas,
//! estilo flow, comillas) lo resuelve `serde_yaml_ng`. Solo se LEE aquí; la edición de
//! repos.yaml se hace línea a línea (ver `reposfile`) para conservar comentarios y formato.

use std::collections::BTreeMap;
use std::fs;
use std::path::Path;

use anyhow::{Context, Result, bail};
use serde::Deserialize;
use serde_yaml_ng::Value;

/// Una entrada de repos.yaml. `Option` porque la clave puede faltar o estar vacía.
#[derive(Debug, Default, Deserialize)]
pub struct RepoEntry {
    target: Option<String>,
    merges: Option<Vec<String>>,
    /// Solo para entradas sintetizadas (repos que no estaban en repos.yaml).
    #[serde(skip)]
    pub url: String,
}

impl RepoEntry {
    /// Entrada mínima para un repo que hoy depende del auto-generado via addons.yaml.
    pub fn synthesized(url: String) -> Self {
        RepoEntry {
            target: Some("origin $ODOO_VERSION".to_string()),
            merges: Some(vec!["origin $ODOO_VERSION".to_string()]),
            url,
        }
    }

    pub fn target(&self) -> &str {
        self.target.as_deref().unwrap_or("")
    }

    pub fn merges(&self) -> &[String] {
        self.merges.as_deref().unwrap_or(&[])
    }
}

/// Nombre del repo (con o sin "./") -> su entrada. BTreeMap las mantiene ordenadas.
pub type ReposFile = BTreeMap<String, Option<RepoEntry>>;

pub fn parse_repos(text: &str) -> Result<ReposFile> {
    // Un documento vacío (o solo comentarios) se lee como `None`.
    Ok(serde_yaml_ng::from_str::<Option<ReposFile>>(text)?.unwrap_or_default())
}

pub fn load_repos(path: &Path) -> Result<ReposFile> {
    let text =
        fs::read_to_string(path).with_context(|| format!("no se pudo leer {}", path.display()))?;
    parse_repos(&text).with_context(|| format!("YAML no valido en {}", path.display()))
}

/// `services.odoo.build.args.ODOO_VERSION` de common.yaml.
pub fn odoo_version(path: &Path) -> Result<String> {
    let text =
        fs::read_to_string(path).with_context(|| format!("no se pudo leer {}", path.display()))?;
    let doc: Value = serde_yaml_ng::from_str(&text)
        .with_context(|| format!("YAML no valido en {}", path.display()))?;

    let version = ["services", "odoo", "build", "args", "ODOO_VERSION"]
        .iter()
        .try_fold(&doc, |value, key| value.get(*key))
        .and_then(|value| match value {
            Value::String(s) => Some(s.clone()),
            Value::Number(n) => Some(n.to_string()), // ODOO_VERSION: 16.0 sin comillas
            _ => None,
        })
        .filter(|v| !v.is_empty());
    match version {
        Some(v) => Ok(v),
        None => bail!(
            "{} no define services.odoo.build.args.ODOO_VERSION",
            path.display()
        ),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn typical_repos_file() {
        let src = "./odoo:
  defaults:
    depth: $DEPTH_MERGE
  remotes:
    origin: https://github.com/odoo/odoo.git  # comentario
  target: origin $ODOO_VERSION
  merges:
    - origin $ODOO_VERSION
    - origin abc  #frozen:x
web:
  merges:
  - origin 16.0
vacio:
";
        let repos = parse_repos(src).unwrap();
        let odoo = repos["./odoo"].as_ref().unwrap();
        assert_eq!(odoo.target(), "origin $ODOO_VERSION");
        assert_eq!(odoo.merges(), ["origin $ODOO_VERSION", "origin abc"]);
        assert_eq!(repos["web"].as_ref().unwrap().merges(), ["origin 16.0"]);
        assert!(repos["vacio"].is_none());
    }

    #[test]
    fn flow_style_and_anchors_work() {
        // Cosas que el parser casero no soportaba.
        let src = "base: &b {target: origin 16.0, merges: [origin 16.0]}\nother: *b\n";
        let repos = parse_repos(src).unwrap();
        assert_eq!(repos["other"].as_ref().unwrap().merges(), ["origin 16.0"]);
    }

    #[test]
    fn empty_documents_are_empty() {
        assert!(parse_repos("").unwrap().is_empty());
        assert!(parse_repos("# solo comentarios\n").unwrap().is_empty());
    }

    #[test]
    fn invalid_yaml_is_an_error() {
        assert!(parse_repos("a: [sin cerrar\n").is_err());
    }

    #[test]
    fn odoo_version_quoted_and_bare() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("common.yaml");
        for (yaml_value, want) in [("\"16.0\"", "16.0"), ("17.0", "17.0")] {
            let doc = format!(
                "services:\n  odoo:\n    build:\n      args:\n        ODOO_VERSION: {yaml_value}\n"
            );
            fs::write(&path, doc).unwrap();
            assert_eq!(odoo_version(&path).unwrap(), want);
        }
        fs::write(&path, "services: {}\n").unwrap();
        assert!(
            odoo_version(&path)
                .unwrap_err()
                .to_string()
                .contains("ODOO_VERSION")
        );
    }
}
