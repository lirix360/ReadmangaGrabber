package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cavaliercoder/grab"
	"github.com/jhoonb/archivex"
)

var mangaChapters []string
var reqURL string

func main() {
	fmt.Println()

	flag.Usage = func() {
		fmt.Println("Использование: " + os.Args[0] + " параметры [список глав для скачивания]\n")
		fmt.Println("Параметры:")
		fmt.Println(" -url=адрес_манги\tАдрес страницы описания манги или отдельной главы")
		fmt.Println(" -zip\t\t\tСоздание ZIP архивов для каждой главы после скачивания")
		fmt.Println(" -delete\t\tУдалить исходные файлы после архивации (используется только вместе с флагом -zip)\n")
		fmt.Println("Список глав для скачивания указывается через пробел в формате том/глава (пример: vol1/5 vol10/65)\n")
	}

	urlPtr := flag.String("url", "", "Адрес страницы описания манги или отдельной главы главы")

	zipPtr := flag.Bool("zip", false, "Создать ZIP архивы для каждой главы после скачивания")
	delPtr := flag.Bool("delete", false, "Удалить исходные файлы после архивации")

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

	if urlParts.Host != "readmanga.me" && urlParts.Host != "mintmanga.com" && urlParts.Host != "selfmanga.ru" {
		fmt.Println("Указан некорректный адрес манги! Скачивание доступно только с сайтов readmanga.me, mintmanga.com и selfmanga.ru.\n")
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

		downloadChapters(urlParts.Host, pathParts[0], *zipPtr, *delPtr)

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

func downloadChapters(mangaHost string, mangaName string, createZip bool, deleteSource bool) {
	url := "http://" + mangaHost + "/" + mangaName + "/"

	for i := 0; i < len(mangaChapters); i++ {
		imageLinks := getImageLinks(url + mangaChapters[i])

		reqURL = url + mangaChapters[i]

		if len(imageLinks) > 0 {
			if _, err := os.Stat("Downloads/" + mangaName + "/" + mangaChapters[i]); os.IsNotExist(err) {
				os.MkdirAll("Downloads/"+mangaName+"/"+mangaChapters[i], 0755)
			}

			fmt.Println("Скачиваю главу " + mangaChapters[i] + ".")

			for x := 0; x < len(imageLinks); x++ {
				downloadFile(imageLinks[x], mangaName, mangaChapters[i])

				time.Sleep(200 * time.Millisecond)
			}

			if createZip {
				fmt.Println("- Архивирую главу " + mangaChapters[i] + ".")

				zip := new(archivex.ZipFile)
				zip.Create("Downloads/" + mangaName + "/" + mangaChapters[i] + ".zip")
				zip.AddAll("Downloads/"+mangaName+"/"+mangaChapters[i], true)
				zip.Close()

				if deleteSource {
					os.RemoveAll("Downloads/" + mangaName + "/" + mangaChapters[i])
				}
			}
		} else {
			fmt.Println("В главе " + mangaChapters[i] + " не найдено страниц для скачивания!")
		}
	}
}

func getImageLinks(chapterURL string) []string {
	var imageLinks []string

	resp, err := http.Get(chapterURL + "?mature=1")
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

			imageLinks = append(imageLinks, strings.Trim(tmpParts[1], "\"'")+strings.Trim(tmpParts[0], "\"'")+strings.Trim(tmpParts[2], "\"'"))
		}
	}

	return imageLinks
}

func downloadFile(fileURL string, mangaName string, mangaChapter string) bool {
	client := grab.NewClient()
	req, _ := grab.NewRequest("Downloads/"+mangaName+"/"+mangaChapter, fileURL)

	client.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36"

	req.HTTPRequest.Header.Set("Referer", reqURL)

	resp := client.Do(req)
	if err := resp.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		return false
	}

	return true
}
