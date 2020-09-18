use std::env;

pub fn gallery_path() -> String {
    env::var("SG_GALLERY_PATH").unwrap_or_else(|_| "gallery/".to_string())
}

pub fn thumbnails_path() -> String {
    env::var("SG_THUMBNAILS_PATH").unwrap_or_else(|_| "thumbnails/".to_string())
}

pub fn notfound_path() -> String {
    env::var("SG_NOTFOUND_PATH").unwrap_or_else(|_| "notfound.jpg".to_string())
}
