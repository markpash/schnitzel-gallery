package main

import (
	"embed"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/nfnt/resize"
)

//go:embed static
var static embed.FS

var (
	GALLERY_PATH    = ""
	THUMBNAILS_PATH = ""
)

var thumbnames = map[string]struct{}{"thumbnail.jpg": {}, "thumbnail.png": {}}

type dirItem struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Thumbnail string `json:"thumbnail"`
	Dir       bool   `json:"dir"`
}

func handleApi(c *fiber.Ctx) error {
	apiPath, err := url.QueryUnescape(c.Params("*"))
	if err != nil {
		return err
	}

	dirs, err := os.ReadDir(path.Join(GALLERY_PATH, apiPath))
	if err != nil {
		return err
	}

	items := []dirItem{}
	for _, dirEntry := range dirs {
		if _, ok := thumbnames[dirEntry.Name()]; ok {
			continue
		}

		item := dirItem{
			Name:      dirEntry.Name(),
			Path:      path.Join("/gallery", apiPath, dirEntry.Name()),
			Thumbnail: path.Join("/thumbnails", apiPath, dirEntry.Name()),
			Dir:       dirEntry.IsDir(),
		}

		if item.Dir {
			item.Path = path.Join(apiPath, dirEntry.Name())
			item.Thumbnail = path.Join(item.Thumbnail, "thumbnail.jpg")
		}
		items = append(items, item)
	}

	return c.JSON(items)
}

// genThumb never errors, it simply returns a solid colour image
func genThumb(input *os.File) image.Image {
	// Create an image from the input file
	inputImage, _, err := image.Decode(input)
	if err == nil {
		// If the input file was an image and it was successfully parsed,
		// then produce a thumbnail image and return it.
		return resize.Thumbnail(400, 400, inputImage, resize.Lanczos3)
	}

	// The input wasn't a valid image, so produce an all black image and
	// return that.
	m := image.NewRGBA(image.Rect(0, 0, 400, 266))
	draw.Draw(m, m.Bounds(), image.Transparent, image.Point{}, draw.Src)

	return m
}

func createThumb(imagePath string) error {
	galleryPath := path.Join(GALLERY_PATH, imagePath)
	thumbPath := path.Join(THUMBNAILS_PATH, imagePath)

	// Open the original image
	file, err := os.Open(galleryPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create the directory structure for the thumbnail we want to store.
	if err := os.MkdirAll(path.Dir(thumbPath), 0o755); err != nil {
		return err
	}

	// Create the thumbnail file.
	thumbFile, err := os.Create(thumbPath)
	if err != nil {
		return err
	}
	defer thumbFile.Close()

	// Generate a thumbnail and encode into a JPEG, then store.
	if err := jpeg.Encode(thumbFile, genThumb(file), nil); err != nil {
		return err
	}

	return nil
}

func thumbnailHandler(c *fiber.Ctx) error {
	imgPath, err := url.QueryUnescape(c.Params("*"))
	if err != nil {
		return err
	}

	filePath := path.Join(THUMBNAILS_PATH, imgPath)
	if _, err = os.Open(filePath); err != nil {
		if err := createThumb(imgPath); err != nil {
			return err
		}
	}

	if err := c.SendFile(path.Join(THUMBNAILS_PATH, imgPath), false); err != nil {
		return err
	}

	return nil
}

func galleryHandler(c *fiber.Ctx) error {
	imgPath, err := url.QueryUnescape(c.Params("*"))
	if err != nil {
		return err
	}

	if err := c.SendFile(path.Join(GALLERY_PATH, imgPath), false); err != nil {
		return err
	}

	return nil
}

func main() {
	var ok bool
	GALLERY_PATH, ok = os.LookupEnv("SG_GALLERY_PATH")
	if !ok {
		GALLERY_PATH = "./gallery"
	}

	THUMBNAILS_PATH, ok = os.LookupEnv("SG_THUMBNAILS_PATH")
	if !ok {
		THUMBNAILS_PATH = "./thumbnails"
	}

	LISTEN_ADDR, ok := os.LookupEnv("SG_LISTEN_ADDR")
	if !ok {
		LISTEN_ADDR = "127.0.0.1:3000"
	}

	app := fiber.New()
	app.Get("/api/*", handleApi)
	app.Get("/thumbnails/*", thumbnailHandler)
	app.Get("/gallery/*", galleryHandler)
	app.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(static),
		PathPrefix: "static",
		Browse:     false,
	}))

	log.Fatal(app.Listen(LISTEN_ADDR))
}
