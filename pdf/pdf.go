package pdf

import (
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"

	_ "image/gif" // GIF images
	_ "image/png" // PNG images

	"golang.org/x/image/webp"

	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/logger"

	"github.com/phpdave11/gofpdf"
)

// CreatePDF - ...
func CreatePDF(chapterPath string, savedFiles []string) error {
	var opt gofpdf.ImageOptions

	pdf := gofpdf.New("P", "mm", "A4", "")

	for _, file := range savedFiles {
		imageFile, err := convertImg(file)
		if err != nil {
			data.WSChan <- data.WSData{
				Cmd: "updateLog",
				Payload: map[string]interface{}{
					"type": "err",
					"text": "-- Ошибка при создании PDF файла (" + chapterPath + ".pdf):" + err.Error(),
				},
			}
			return err
		}

		width, height, err := resizeToFit(imageFile)
		if err != nil {
			data.WSChan <- data.WSData{
				Cmd: "updateLog",
				Payload: map[string]interface{}{
					"type": "err",
					"text": "-- Ошибка при создании PDF файла (" + chapterPath + ".pdf):" + err.Error(),
				},
			}
			return err
		}

		if width < height {
			pdf.AddPage()
			pdf.ImageOptions(imageFile, (data.PDFOpts.A4Width-width)/2, (data.PDFOpts.A4Height-height)/2, width, height, false, opt, 0, "")
		} else {
			pdf.AddPageFormat("L", pdf.GetPageSizeStr("A4"))
			pdf.ImageOptions(imageFile, (data.PDFOpts.A4Height-width)/2, (data.PDFOpts.A4Width-height)/2, width, height, false, opt, 0, "")
		}
	}

	err := pdf.OutputFileAndClose(chapterPath + ".pdf")
	if err != nil {
		logger.Log.Error("Ошибка при создании PDF файла ("+chapterPath+".pdf):", err)
		data.WSChan <- data.WSData{
			Cmd: "updateLog",
			Payload: map[string]interface{}{
				"type": "err",
				"text": "-- Ошибка при создании PDF файла (" + chapterPath + ".pdf):" + err.Error(),
			},
		}
		return err
	}

	err = os.RemoveAll(chapterPath + "/pdf")
	if err != nil {
		logger.Log.Error("Ошибка при удалении временных файлов PDF:", err)
	}

	return nil
}

func resizeToFit(imgFilename string) (float64, float64, error) {
	var widthScale, heightScale float64

	width, height, err := getImageDimension(imgFilename)
	if err != nil {
		return 0, 0, err
	}

	if width < height {
		widthScale = data.PDFOpts.MaxWidth / width
		heightScale = data.PDFOpts.MaxHeight / height
	} else {
		widthScale = data.PDFOpts.MaxHeight / width
		heightScale = data.PDFOpts.MaxWidth / height
	}

	scale := math.Min(widthScale, heightScale)

	return math.Round(pixelsToMM(scale * width)), math.Round(pixelsToMM(scale * height)), nil
}

func convertImg(srcImg string) (string, error) {
	var imgSrc image.Image
	var err error

	srcPath := filepath.Dir(srcImg)
	dstPath := filepath.Join(srcPath, "pdf")
	srcFile := filepath.Base(srcImg)
	dstFile := filepath.Join(dstPath, srcFile+".jpg")

	fileExt := filepath.Ext(srcImg)

	imgFile, _ := os.Open(srcImg)

	if fileExt == ".webp" {
		imgSrc, err = webp.Decode(imgFile)
		if err != nil {
			logger.Log.Error("Ошибка при декодировании файла ("+srcImg+"):", err)
			imgFile.Close()
			return "", err
		}
	} else {
		imgSrc, _, err = image.Decode(imgFile)
		if err != nil {
			logger.Log.Error("Ошибка при декодировании файла ("+srcImg+"):", err)
			imgFile.Close()
			return "", err
		}
	}

	if _, err = os.Stat(dstPath); os.IsNotExist(err) {
		err = os.MkdirAll(dstPath, 0755)
		if err != nil {
			logger.Log.Error("Ошибка при создании временной папки PDF:", err)
			imgFile.Close()
			return "", err
		}
	}

	newImg := image.NewRGBA(imgSrc.Bounds())

	draw.Draw(newImg, newImg.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(newImg, newImg.Bounds(), imgSrc, imgSrc.Bounds().Min, draw.Over)

	jpgFile, err := os.Create(dstFile)
	if err != nil {
		logger.Log.Error("Ошибка при создании временного файла ("+dstFile+"):", err)
		imgFile.Close()
		return "", err
	}

	var opt jpeg.Options
	opt.Quality = 90

	err = jpeg.Encode(jpgFile, newImg, &opt)
	if err != nil {
		logger.Log.Error("Ошибка при записи временного файла ("+dstFile+"):", err)
		imgFile.Close()
		jpgFile.Close()
		return "", err
	}

	imgFile.Close()
	jpgFile.Close()

	return dstFile, nil
}

func getImageDimension(imagePath string) (float64, float64, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		logger.Log.Error("Ошибка при открытии файла:", err)
		return 0, 0, err
	}
	defer file.Close()

	image, _, err := image.DecodeConfig(file)
	if err != nil {
		logger.Log.Error("Ошибка при обработке файла:", err)
		return 0, 0, err
	}

	return float64(image.Width), float64(image.Height), nil
}

func pixelsToMM(val float64) float64 {
	return float64(val * data.PDFOpts.MmInInch / data.PDFOpts.DPI)
}
