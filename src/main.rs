#![feature(proc_macro_hygiene, decl_macro)]

#[macro_use]
extern crate rocket;
#[macro_use]
extern crate serde_derive;
#[macro_use]
extern crate serde_json;

mod helpers;
mod paths;
use dotenv;
use helpers::{dirvec, genthumb};
use paths::{assets_path, gallery_path, thumbnails_path};
use rocket::response::NamedFile;
use rocket_contrib::json::Json;
use rocket_contrib::serve::StaticFiles;
use rocket_contrib::templates::Template;
use serde_json::Value;
use std::collections::HashMap;
use std::io;
use std::path::{Path, PathBuf};
use std::str::FromStr;

#[get("/")]
fn index() -> Template {
    let context = HashMap::<String, String>::new();
    Template::render("index", &context)
}

#[get("/thumbnails/<path..>")]
fn thumbnails(path: PathBuf) -> Result<NamedFile, io::Error> {
    let local_path = Path::new(&gallery_path()).join(&path);
    let local_thumb = Path::new(&thumbnails_path()).join(&path);
    if local_thumb.is_file() {
        NamedFile::open(local_thumb)
    } else if local_path.is_file() && !local_thumb.is_file() {
        genthumb(path);
        NamedFile::open(local_thumb)
    } else {
        NamedFile::open(Path::new(&assets_path()).join("notfound.jpg"))
    }
}

#[get("/gallery/<path..>")]
fn gallery(path: PathBuf) -> Result<NamedFile, io::Error> {
    let local_path = Path::new(&gallery_path()).join(path);
    if local_path.is_file() {
        NamedFile::open(local_path)
    } else {
        NamedFile::open(Path::new(&assets_path()).join("notfound.jpg"))
    }
}

#[get("/api")]
fn apihome() -> Json<Value> {
    match dirvec(PathBuf::from_str("").unwrap()) {
        Ok(diritems) => Json(json!({"status": "success", "content": diritems})),
        Err(_) => Json(json!({"status": "fail", "content": "this path does not exist"})),
    }
}

#[get("/api/<path..>")]
fn api(path: PathBuf) -> Json<Value> {
    match dirvec(path) {
        Ok(diritems) => Json(json!({"status": "success", "content": diritems})),
        Err(_) => Json(json!({"status": "fail", "content": "this path does not exist"})),
    }
}

fn main() {
    dotenv::dotenv().unwrap();
    rocket::ignite()
        .mount("/", routes![index, api, apihome, gallery, thumbnails])
        .mount("/assets", StaticFiles::from(assets_path()))
        .attach(Template::fairing())
        .launch();
}
