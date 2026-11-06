//! odoo-freeze-repos: congela/descongela el repos.yaml de un proyecto Doodba a los commits
//! que ya existen en disco, para que el siguiente `invoke git-aggregate` descargue
//! exactamente el mismo código.
//!
//! El crate es una biblioteca (esta raíz) más un binario fino (`main.rs`). Así la lógica se
//! puede probar desde `tests/` igual que lo haría cualquier otro crate.

pub mod cli;
pub mod config;
pub mod freezer;
pub mod gitx;
pub mod project;
pub mod reposfile;
