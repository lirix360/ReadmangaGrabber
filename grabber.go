package main

import (
	"compress/flate"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/cavaliercoder/grab"
	"github.com/mholt/archiver"
	"github.com/phpdave11/gofpdf"
)

const (
	dpi       = 96
	mmInInch  = 25.4
	a4Height  = 297
	a4Width   = 210
	maxHeight = 1122
	maxWidth  = 793
)

var mangaChapters []string

func main() {
	fmt.Println()

	flag.Usage = func() {
		fmt.Println("Использование: " + os.Args[0] + " параметры [список глав для скачивания]\n")
		fmt.Println("Параметры:")
		fmt.Println(" -url=адрес_манги\tАдрес страницы описания манги или отдельной главы")
		fmt.Println(" -pdf\t\t\tСоздание PDF файлов для каждой главы после скачивания")
		fmt.Println(" -zip\t\t\tСоздание ZIP архивов для каждой главы после скачивания")
		fmt.Println(" -cbz\t\t\tСоздание CBZ архивов для каждой главы после скачивания")
		fmt.Println(" -delete\t\tУдалить исходные файлы после создания PDF или архивации (используется только вместе с флагами -pdf или -zip)\n")
		fmt.Println("Список глав для скачивания указывается через пробел в формате том/глава (пример: vol1/5 vol10/65)\n")
	}

	urlPtr := flag.String("url", "", "Адрес страницы описания манги или отдельной главы главы")

	zipPtr := flag.Bool("zip", false, "Создать ZIP архивы для каждой главы после скачивания")
	cbzPtr := flag.Bool("cbz", false, "Создать CBZ архивы для каждой главы после скачивания")
	delPtr := flag.Bool("delete", false, "Удалить исходные файлы после архивации")

	pdfPtr := flag.Bool("pdf", false, "Создать PDF файлы для каждой главы после скачивания")

	flag.Parse()

	if *urlPtr == "" {
		flag.Usage()
		os.Exit(0)
	}

	urlParts, err := url.Parse(*urlPtr)
	if err != nil {
		fmt.Println("Произошла ошибка при обработке адреса манги!\n")
		os.Exit(0)
	}

	if urlParts.Host != "readmanga.live" && urlParts.Host != "mintmanga.live" && urlParts.Host != "selfmanga.ru" && urlParts.Host != "allhentai.ru" {
		fmt.Println("Указан некорректный адрес манги! Скачивание доступно только с сайтов readmanga.live, mintmanga.live и selfmanga.ru.\n")
		os.Exit(0)
	}

	pathParts := strings.Split(strings.Trim(urlParts.Path, "/"), "/")

	if len(pathParts) == 1 {
		if len(flag.Args()) > 0 {
			mangaChapters = flag.Args()
		} else {
			getChapters(*urlPtr)
		}
	} else if len(pathParts) == 3 {
		mangaChapters = append(mangaChapters, pathParts[1]+"/"+pathParts[2])
	} else {
		fmt.Println("Указан некорректный адрес манги!\n")
		os.Exit(0)
	}

	if len(mangaChapters) > 0 {
		fmt.Println("Начинаю скачивание.")

		downloadChapters(urlParts.Host, pathParts[0], *pdfPtr, *zipPtr, *cbzPtr, *delPtr)

		fmt.Println("Скачивание завершено.")
	} else {
		fmt.Println("Не найдено глав для скачивания!\n")
		os.Exit(0)
	}

	fmt.Println()
}

func getChapters(mangaURL string) {
	mangaPage, err := goquery.NewDocument(mangaURL)
	if err != nil {
		fmt.Println("Произошла ошибка при поиске глав для скачивания!\n")
		os.Exit(0)
	}

	mangaPage.Find(".chapters-link a").Each(func(i int, s *goquery.Selection) {
		link, err := s.Attr("href")
		if !err {
			fmt.Println("Произошла ошибка при поиске глав для скачивания!\n")
			os.Exit(0)
		}

		linkPaths := strings.Split(strings.Trim(link, "/"), "/")

		mangaChapters = append(mangaChapters, linkPaths[1]+"/"+linkPaths[2])
	})

	for left, right := 0, len(mangaChapters)-1; left < right; left, right = left+1, right-1 {
		mangaChapters[left], mangaChapters[right] = mangaChapters[right], mangaChapters[left]
	}
}

func downloadChapters(mangaHost, mangaName string, createPdf, createZip, createCbz, deleteSource bool) {
	url := "http://" + mangaHost + "/" + mangaName + "/"

	for i := 0; i < len(mangaChapters); i++ {
		fmt.Println("Скачиваю главу " + mangaChapters[i] + ".")

		imageLinks := getImageLinks(url + mangaChapters[i])

		if len(imageLinks) > 0 {
			if _, err := os.Stat("Downloads/" + mangaName + "/" + mangaChapters[i]); os.IsNotExist(err) {
				os.MkdirAll("Downloads/"+mangaName+"/"+mangaChapters[i], 0755)
			}

			imagesReqs := make([]*grab.Request, 0)

			for x := 0; x < len(imageLinks); x++ {
				imageReq, _ := grab.NewRequest("Downloads/"+mangaName+"/"+mangaChapters[i], imageLinks[x])
				imageReq.HTTPRequest.Header.Set("Referer", url+mangaChapters[i])

				imagesReqs = append(imagesReqs, imageReq)
			}

			client := grab.NewClient()
			client.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.106 Safari/537.36"

			respch := client.DoBatch(2, imagesReqs...)

			for resp := range respch {
				if err := resp.Err(); err != nil {
					fmt.Printf("- Ошибка скачивания файла: %s (%s)", resp.Filename, resp.Request.URL())
				}
			}

			var wg sync.WaitGroup

			if createZip || createCbz {
				wg.Add(1)

				go func(mangaName, mangaChapter string, wg *sync.WaitGroup) {
					defer wg.Done()

					fmt.Println("- Архивирую главу " + mangaChapter + ".")

					z := archiver.Zip{
						CompressionLevel:       flate.NoCompression,
						MkdirAll:               true,
						SelectiveCompression:   true,
						ContinueOnError:        false,
						OverwriteExisting:      false,
						ImplicitTopLevelFolder: false,
					}

					err := z.Archive([]string{"Downloads/" + mangaName + "/" + mangaChapter}, "Downloads/"+mangaName+"/"+mangaChapter+".zip")
					if err != nil {
						fmt.Println("-- Ошибка при создании архива:", err.Error())
					}
					z.Close()
				}(mangaName, mangaChapters[i], &wg)
			}

			if createPdf {
				wg.Add(1)

				fmt.Println("- Создаю PDF для главы " + mangaChapters[i] + ".")

				go createPDF("Downloads/"+mangaName+"/"+mangaChapters[i], imageLinks, &wg)
			}

			wg.Wait()

			if createCbz {
				err := os.Rename("Downloads/"+mangaName+"/"+mangaChapters[i]+".zip", "Downloads/"+mangaName+"/"+mangaChapters[i]+".cbz")
				if err != nil {
					fmt.Println("-- Ошибка при переименовании архива:", err.Error())
				}
			}

			if deleteSource {
				err := os.RemoveAll("Downloads/" + mangaName + "/" + mangaChapters[i])
				if err != nil {
					fmt.Println("-- Ошибка при удалении файлов:", err.Error())
				}
			}
		} else {
			fmt.Println("- В главе " + mangaChapters[i] + " не найдено страниц для скачивания!")
		}
	}
}

func getImageLinks(chapterURL string) []string {
	var imageLinks []string

	resp, err := http.Get(chapterURL + "?mtr=1")
	if err != nil {
		return imageLinks
	}

	pageBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return imageLinks
	}

	resp.Body.Close()

	r := regexp.MustCompile(`rm_h\.init(.+);`)
	r2 := regexp.MustCompile(`\[.+\]`)

	imagePartsString := strings.Trim(r2.FindString(r.FindString(string(pageBody))), "[]")

	if imagePartsString != "" {
		imageParts := strings.Split(imagePartsString, "],[")

		for i := 0; i < len(imageParts); i++ {
			tmpParts := strings.Split(imageParts[i], ",")

			imageLinks = append(imageLinks, strings.Trim(tmpParts[0], "\"'")+strings.Trim(tmpParts[2], "\"'"))
		}
	}

	return imageLinks
}

func createPDF(path string, imageLinks []string, wg *sync.WaitGroup) {
	defer wg.Done()

	var images []string

	for x := 0; x < len(imageLinks); x++ {
		fileURL, _ := url.Parse(imageLinks[x])
		images = append(images, filepath.Base(fileURL.Path))
	}

	var opt gofpdf.ImageOptions

	pdf := gofpdf.New("P", "mm", "A4", "")

	for _, i := range images {
		width, height := resizeToFit(path + "/" + i)

		imageFile := checkImg(path + "/" + i)
		if imageFile == "SKIP" {
			continue
		}

		if width < height {
			pdf.AddPage()
			pdf.ImageOptions(imageFile, (a4Width-width)/2, (a4Height-height)/2, width, height, false, opt, 0, "")
		} else {
			pdf.AddPageFormat("L", pdf.GetPageSizeStr("A4"))
			pdf.ImageOptions(imageFile, (a4Height-width)/2, (a4Width-height)/2, width, height, false, opt, 0, "")
		}
	}

	err := pdf.OutputFileAndClose(path + ".pdf")
	if err != nil {
		fmt.Println("- Ошибка создания PDF файла: ", err.Error())
	}
}

func pixelsToMM(val float64) float64 {
	return float64(val * mmInInch / dpi)
}

func resizeToFit(imgFilename string) (float64, float64) {
	var widthScale, heightScale float64

	width, height := getImageDimension(imgFilename)

	if width < height {
		widthScale = maxWidth / width
		heightScale = maxHeight / height
	} else {
		widthScale = maxHeight / width
		heightScale = maxWidth / height
	}

	scale := math.Min(widthScale, heightScale)

	return math.Round(pixelsToMM(scale * width)), math.Round(pixelsToMM(scale * height))
}

func getImageDimension(imagePath string) (float64, float64) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	image, _, err := image.DecodeConfig(file)
	if err != nil {
		log.Fatal(err)
	}

	return float64(image.Width), float64(image.Height)
}

func checkImg(srcImg string) (completeImg string) {
	srcExt := filepath.Ext(srcImg)

	if srcExt == ".png" {
		completeImg = convertPng(srcImg)
	} else {
		completeImg = srcImg
	}

	return completeImg
}

func convertPng(pngImg string) string {
	pngImgFile, _ := os.Open(pngImg)

	imgSrc, err := png.Decode(pngImgFile)
	if err != nil {
		fmt.Println("-- Skipping file: " + pngImg)
		fmt.Println("-- " + err.Error())

		pngImgFile.Close()

		return "SKIP"
	}

	newImg := image.NewRGBA(imgSrc.Bounds())

	draw.Draw(newImg, newImg.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	draw.Draw(newImg, newImg.Bounds(), imgSrc, imgSrc.Bounds().Min, draw.Over)

	jpgImgFile, _ := os.Create(pngImg + ".jpg")

	var opt jpeg.Options
	opt.Quality = 80

	err = jpeg.Encode(jpgImgFile, newImg, &opt)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	pngImgFile.Close()
	jpgImgFile.Close()

	err = os.Remove(pngImg)

	return pngImg + ".jpg"
}
