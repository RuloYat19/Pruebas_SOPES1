// SPDX-License-Identifier: GPL-2.0

//! Módulo de Rust para la actividad extra del Proyecto 2.

#![no_std]

use kernel::prelude::*;

module! {
    type: ModuloRust,
    name: b"rust_modulo",
    author: b"Raul Emanuel Yat Cancinos - 202300722",
    description: b"Modulo Rust que imprime mi nombre y carnet",
    license: b"GPL",
}

struct ModuloRust;

impl kernel::Module for ModuloRust {
    fn init(_name: &'static CStr, _module: &'static ThisModule) -> Result<Self> {
        pr_info!("Hola wenas a todos, soy Raul Emanuel Yat Cancinos de carnet 202300722, ando haciendo esto desde Rust\n");
        Ok(ModuloRust)
    }
}

impl Drop for ModuloRust {
    fn drop(&mut self) {
        pr_info!("Modulo de Rust descargado. Adios!\n");
    }
}