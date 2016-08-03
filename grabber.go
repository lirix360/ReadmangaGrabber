package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jhoonb/archivex"
	"github.com/mgutz/ansi"
)

var mangaChapters []string

func main() {
	fmt.Println()

	flag.Usage = func() {
		fmt.Println(ansi.Blue, "Использование: "+os.Args[0]+" параметры [список глав для скачивания]\n", ansi.Reset)
		fmt.Println(ansi.Blue, "Параметры:", ansi.Reset)
		fmt.Println(ansi.Blue, " -url=адрес_манги\tАдрес страницы описания манги или отдельной главы", ansi.Reset)
		fmt.Println(ansi.Blue, " -zip\t\t\tСоздание ZIP архивов для каждой главы после скачивания", ansi.Reset)
		fmt.Println(ansi.Blue, " -delete\t\tУдалить исходные файлы после архивации (используется только вместе с флагом -zip)\n", ansi.Reset)
		fmt.Println(ansi.Blue, "Список глав для скачивания указывается через пробел в формате том/глава (пример: vol1/5 vol10/65)\n", ansi.Reset)
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
		fmt.Println(ansi.Red, "Произошла ошибка при обработке адреса манги!\n", ansi.Reset)
		os.Exit(0)
	}

	if urlParts.Host != "readmanga.me" && urlParts.Host != "mintmanga.com" && urlParts.Host != "selfmanga.ru" {
		fmt.Println(ansi.Red, "Указан некорректный адрес манги! Скачивание доступно только с сайтов readmanga.me, mintmanga.com и selfmanga.ru.\n", ansi.Reset)
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
		fmt.Println(ansi.Red, "Указан некорректный адрес манги!\n", ansi.Reset)
		os.Exit(0)
	}

	if len(mangaChapters) > 0 {
		fmt.Println(ansi.Green, "Начинаю скачивание.", ansi.Reset)

		downloadChapters(urlParts.Host, pathParts[0], *zipPtr, *delPtr)

		fmt.Println(ansi.Green, "Скачивание завершено.", ansi.Reset)
	} else {
		fmt.Println(ansi.Red, "Не найдено глав для скачивания!\n", ansi.Reset)
		os.Exit(0)
	}

	fmt.Println()
}

func getChapters(mangaURL string) {
	mangaPage, err := goquery.NewDocument(mangaURL)
	if err != nil {
		fmt.Println(ansi.Red, "Произошла ошибка при поиске глав для скачивания!\n", ansi.Reset)
		os.Exit(0)
	}

	mangaPage.Find(".chapters-link a").Each(func(i int, s *goquery.Selection) {
		link, err := s.Attr("href")
		if err {
			fmt.Println(ansi.Red, "Произошла ошибка при поиске глав для скачивания!\n", ansi.Reset)
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

		if len(imageLinks) > 0 {
			fmt.Println(ansi.Green, "Скачиваю главу "+mangaChapters[i]+".", ansi.Reset)

			for x := 0; x < len(imageLinks); x++ {
				downloadFile(imageLinks[x], mangaName, mangaChapters[i])

				time.Sleep(200 * time.Millisecond)
			}

			if createZip {
				fmt.Println(ansi.Green, "- Архивирую главу "+mangaChapters[i]+".", ansi.Reset)

				zip := new(archivex.ZipFile)
				zip.Create("Downloaded manga/" + mangaName + "/" + mangaChapters[i] + ".zip")
				zip.AddAll("Downloaded manga/"+mangaName+"/"+mangaChapters[i], true)
				zip.Close()

				if deleteSource {
					os.RemoveAll("Downloaded manga/" + mangaName + "/" + mangaChapters[i])
				}
			}
		} else {
			fmt.Println(ansi.Red, "В главе "+mangaChapters[i]+" не найдено страниц для скачивания!", ansi.Reset)
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
	urlParts := strings.Split(fileURL, "/")

	if _, err := os.Stat("Downloaded manga/" + mangaName + "/" + mangaChapter); os.IsNotExist(err) {
		os.MkdirAll("Downloaded manga/"+mangaName+"/"+mangaChapter, 0755)
	}

	fp, err := os.OpenFile("Downloaded manga/"+mangaName+"/"+mangaChapter+"/"+urlParts[len(urlParts)-1], os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Println(ansi.Red, "Произошла ошибка при скачивании файла ("+mangaChapter+"/"+urlParts[len(urlParts)-1]+")!\n", ansi.Reset)
		return false
	}
	defer fp.Close()

	resp, err := http.Get(fileURL)
	if err != nil {
		fmt.Println(ansi.Red, "Произошла ошибка при скачивании файла ("+mangaChapter+"/"+urlParts[len(urlParts)-1]+")!\n", ansi.Reset)
		return false
	}
	defer resp.Body.Close()

	_, err2 := io.Copy(fp, resp.Body)
	if err2 != nil {
		fmt.Println(ansi.Red, "Произошла ошибка при скачивании файла ("+mangaChapter+"/"+urlParts[len(urlParts)-1]+")!\n", ansi.Reset)
		return false
	}

	return true
}
