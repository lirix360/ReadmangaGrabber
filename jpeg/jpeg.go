package jpeg

import (
	"image/jpeg"
	"os"

	"github.com/nfnt/resize"
)

func ResizeImage(path string, width uint, height uint) {
	if width < 1 || height < 1 {
		return
	}

	file, err := os.Open(path)
	if err != nil {
		return
	}

	img, err := jpeg.Decode(file)
	if err != nil {
		return
	}
	file.Close()

	m := resize.Resize(width, height, img, resize.Lanczos3)

	out, err := os.Create(path)
	if err != nil {
		return
	}
	defer out.Close()

	jpeg.Encode(out, m, nil)
}
