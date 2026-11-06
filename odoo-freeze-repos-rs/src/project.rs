//! Rutas de un proyecto Doodba a partir de su carpeta raíz.

use std::fs;
use std::path::{Path, PathBuf};

use anyhow::{Context, Result, bail};

#[derive(Debug, Clone)]
pub struct Project {
    pub root: PathBuf,
    pub src: PathBuf,
    pub repos_yaml: PathBuf,
    pub backup_yaml: PathBuf,
    pub frozen_yaml: PathBuf,
    pub common_yaml: PathBuf,
}

impl Project {
    /// Calcula las rutas a partir de la raíz. Solo toca el disco para hacerla absoluta.
    pub fn new(root: &Path) -> Result<Project> {
        let root = std::path::absolute(root)
            .with_context(|| format!("ruta no valida: {}", root.display()))?;
        let src = root.join("odoo/custom/src");
        Ok(Project {
            repos_yaml: src.join("repos.yaml"),
            backup_yaml: src.join("bck-repos.yaml"),
            frozen_yaml: src.join("frozen-repos.yaml"),
            common_yaml: root.join("common.yaml"),
            src,
            root,
        })
    }

    /// Ruta relativa a la raíz del proyecto (para mensajes).
    pub fn rel(&self, path: &Path) -> String {
        path.strip_prefix(&self.root)
            .unwrap_or(path)
            .display()
            .to_string()
    }

    /// Comprueba que existan los ficheros imprescindibles.
    pub fn validate(&self, need_common: bool) -> Result<()> {
        let mut required = vec![&self.src, &self.repos_yaml];
        if need_common {
            required.push(&self.common_yaml);
        }
        let missing: Vec<String> = required
            .into_iter()
            .filter(|p| !p.exists())
            .map(|p| p.display().to_string())
            .collect();
        if !missing.is_empty() {
            bail!(
                "{} no parece un proyecto Doodba; falta: {}",
                self.root.display(),
                missing.join(", ")
            );
        }
        Ok(())
    }

    /// Repos con .git bajo odoo/custom/src (sin private/), ordenados.
    pub fn discover_repos(&self) -> Result<Vec<String>> {
        let mut repos = Vec::new();
        for entry in fs::read_dir(&self.src)
            .with_context(|| format!("no se pudo leer {}", self.src.display()))?
        {
            let path = entry?.path();
            let name = path
                .file_name()
                .unwrap_or_default()
                .to_string_lossy()
                .into_owned();
            if name != "private" && path.join(".git").is_dir() {
                repos.push(name);
            }
        }
        repos.sort();
        Ok(repos)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn discover_repos_ignores_private_and_non_git() {
        let dir = tempfile::tempdir().unwrap();
        let src = dir.path().join("odoo/custom/src");
        for d in ["web/.git", "account/.git", "private/.git", "sin-git"] {
            fs::create_dir_all(src.join(d)).unwrap();
        }
        let p = Project::new(dir.path()).unwrap();
        assert_eq!(p.discover_repos().unwrap(), ["account", "web"]);
    }

    #[test]
    fn validate_fails_on_empty_dir() {
        let dir = tempfile::tempdir().unwrap();
        let err = Project::new(dir.path())
            .unwrap()
            .validate(false)
            .unwrap_err();
        assert!(
            err.to_string().contains("no parece un proyecto Doodba"),
            "{err}"
        );
    }

    #[test]
    fn rel_strips_root() {
        let p = Project::new(Path::new("/tmp/proj")).unwrap();
        assert_eq!(p.rel(&p.repos_yaml), "odoo/custom/src/repos.yaml");
    }
}
