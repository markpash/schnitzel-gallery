package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/nfnt/resize"
	"golang.org/x/sync/semaphore"
)

//go:embed static
var static embed.FS

//go:embed notfound.jpg
var notfound []byte

var (
	GALLERY_PATH    = ""
	THUMBNAILS_PATH = ""
)

const defaultThumbnailName = "thumbnail.jpg"

var thumbNameMatch = regexp.MustCompile(`thumbnail\.(jpg|jpeg|png)$`)

var sem *semaphore.Weighted

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
		if thumbNameMatch.MatchString(strings.ToLower(dirEntry.Name())) {
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
			item.Thumbnail = path.Join(item.Thumbnail, defaultThumbnailName)
		}
		items = append(items, item)
	}

	return c.JSON(items)
}

func makeThumb(dest io.Writer, origFile io.Reader, thumbPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// From here onwards, things get a little intensive, so limit the
	// number of concurrent executions of this code-path.
	if err := sem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer sem.Release(1)

	if err := os.MkdirAll(path.Dir(thumbPath), 0o755); err != nil {
		return err
	}

	// If the original file can't be decoded as an image, then generate
	// an all-black default image and return that.
	inputImage, _, err := image.Decode(origFile)
	if err != nil {
		// TODO: This could be done once beforehand.
		m := image.NewRGBA(image.Rect(0, 0, 400, 266))
		draw.Draw(m, m.Bounds(), image.Transparent, image.Point{}, draw.Src)

		// Encode directly to the response
		if err := jpeg.Encode(dest, m, nil); err != nil {
			return err
		}
		return nil
	}

	// If the thumbnail file didn't exist at all, then attempt to create
	// all the parent directories and create the file. We create the
	// file here, despite not writing to it with this handle, because we
	// want to return an error within this request context.
	thumbFile, err := os.Create(thumbPath)
	if err != nil {
		return err
	}
	defer thumbFile.Close()

	// The original file is a valid image, generate and store a
	// thumbnail.
	generatedThumb := resize.Thumbnail(400, 400, inputImage, resize.Lanczos3)

	// Encode the thumbnail to an in-memory buffer
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, generatedThumb, nil); err != nil {
		return err
	}

	// Asynchronously write the encoded thumbnail to disk. By first
	// making a copy of the buffer contents, then passing that to the
	// goroutine.
	bufCopy := make([]byte, buf.Len())
	copy(bufCopy, buf.Bytes())

	go func(buf []byte) {
		// The handle for the file may not last long enough to be
		// accessed here, so open the file again.
		f, err := os.OpenFile(thumbPath, os.O_WRONLY, os.ModePerm)
		if err != nil {
			log.Printf("failed to write thumbnail to disk: %s", err.Error())
			return
		}
		defer f.Close()

		if _, err := f.Write(buf); err != nil {
			log.Printf("failed to write thumbnail to disk: %s", err.Error())
		}
	}(bufCopy)

	// Write out the in-memory thumbnail to the response.
	if _, err := buf.WriteTo(dest); err != nil {
		return err
	}

	return nil
}

func findThumbSource(dir string) (*os.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !thumbNameMatch.MatchString(strings.ToLower(entry.Name())) {
			continue
		}

		f, err := os.Open(path.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}

		return f, nil
	}

	return nil, errors.New("source image for thumbnail not found")
}

func thumbnailHandler(c *fiber.Ctx) error {
	urlPath, err := url.QueryUnescape(c.Params("*"))
	if err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}

	galleryPath := path.Join(GALLERY_PATH, urlPath)
	thumbPath := path.Join(THUMBNAILS_PATH, urlPath)

	thumbPresent := true
	// Check to see if the thumbnail exists on disk
	thumbFile, err := os.Open(thumbPath)
	if err != nil {
		// The thumbnail doesn't exist
		thumbPresent = false
	} else {
		defer thumbFile.Close()

		// The thumbnail file exits, but the file may be empty.
		stat, err := thumbFile.Stat()
		if err != nil || stat.Size() < 1 {
			thumbPresent = false
		}
	}

	// If the thumbnail exists, send it.
	if thumbPresent {
		if _, err := io.Copy(c, thumbFile); err != nil {
			return c.SendStatus(http.StatusInternalServerError)
		}
		return nil
	}

	// The thumbnail isn't present, generate it if necessary.

	origNotFound := false
	// Don't continue if the original file isn't present.
	origFile, err := os.Open(galleryPath)
	if err != nil {
		// There's an edge-case where the request is for a thumbnail
		// called thumbnail.jpg, but the original image for it would
		// have uppercases, or a different extension/format. Handle this
		// special case.
		if path.Base(thumbPath) == defaultThumbnailName {
			// Iterate over the gallery directory and try to find a
			// match.
			origFile, err = findThumbSource(path.Dir(galleryPath))
			if err != nil {
				origNotFound = true
			}
			defer origFile.Close()
		} else {
			origNotFound = true
		}
	}
	defer origFile.Close()

	// The original file doesn't exist, return 404 and a notfound
	// image.
	if origNotFound {
		if _, err := c.Write(notfound); err != nil {
			return c.SendStatus(http.StatusInternalServerError)
		}
		return c.SendStatus(http.StatusNotFound)
	}

	if err := makeThumb(c, origFile, thumbPath); err != nil {
		return c.SendStatus(http.StatusInternalServerError)
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

	CONCURRENT_THUMBS, ok := os.LookupEnv("SG_CONCURRENT_THUMBS")
	if !ok {
		CONCURRENT_THUMBS = "16"
	}

	c, err := strconv.Atoi(CONCURRENT_THUMBS)
	if err != nil {
		log.Fatal(err)
	}

	if c < 1 {
		log.Fatal("SG_CONCURRENT_THUMBS must be greater than 0")
	}

	sem = semaphore.NewWeighted(int64(c))

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
