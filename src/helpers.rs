use crate::paths::{gallery_path, thumbnails_path};
use image::FilterType::Lanczos3;
use std::fs::{create_dir_all, read_dir};
use std::io;
use std::path::{Path, PathBuf};

#[derive(Serialize, Eq, Ord, PartialOrd, PartialEq)]
pub struct DirItem {
    name: String,
    path: String,
    thumbnail: String,
    dir: bool,
}

pub fn dirvec(path: PathBuf) -> Result<Vec<DirItem>, io::Error> {
    let dirlist = read_dir(Path::new(&gallery_path()).join(&path))?;
    let thumbnames = vec!["thumbnail.jpg", "thumbnail.png"];
    let mut diritems: Vec<DirItem> = Vec::new();

    for item in dirlist {
        let item = item?;
        let filename = item.file_name().into_string().unwrap();
        if !thumbnames.contains(&filename.to_lowercase().as_str()) {
            let isdir = item.file_type()?.is_dir();
            let mut item_path = Path::new("/gallery/").join(&path).join(&filename);
            let mut thumb_path = Path::new("/thumbnails/").join(&path).join(&filename);
            if isdir {
                item_path = path.join(&filename);
                thumb_path = thumb_path.join("thumbnail.jpg");
            }
            diritems.push(DirItem {
                name: filename,
                path: item_path.to_str().unwrap().to_string(),
                thumbnail: thumb_path.to_str().unwrap().to_string(),
                dir: isdir,
            });
        }
    }
    diritems.sort();
    Ok(diritems)
}

pub fn genthumb(path: PathBuf) {
    let thumb = Path::new(&thumbnails_path()).join(&path);
    let image = Path::new(&gallery_path()).join(&path);
    create_dir_all(thumb.parent().unwrap()).unwrap();
    let original_image = image::open(image).unwrap();
    let resized = original_image.resize_to_fill(400, 400, Lanczos3);
    resized.save(thumb).unwrap();
}
