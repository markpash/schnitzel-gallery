use std::env;

pub fn assets_path() -> String {
    env::var("SG_ASSETS_PATH").unwrap_or_else(|_| "assets/".to_string())
}

pub fn gallery_path() -> String {
    env::var("SG_GALLERY_PATH").unwrap_or_else(|_| "gallery/".to_string())
}

pub fn thumbnails_path() -> String {
    env::var("SG_THUMBNAILS_PATH").unwrap_or_else(|_| "thumbnails/".to_string())
}
