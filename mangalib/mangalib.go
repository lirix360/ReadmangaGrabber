package mangalib

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cavaliercoder/grab"

	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/pdf"
	"github.com/lirix360/ReadmangaGrabber/tools"
)

type ChaptersRawData struct {
	Chapters struct {
		List []struct {
			ChapterName   string `json:"chapter_name"`
			ChapterNumber string `json:"chapter_number"`
			ChapterVolume int    `json:"chapter_volume"`
		} `json:"list"`
	} `json:"chapters"`
}

type PagesList []struct {
	Page int    `json:"p"`
	URL  string `json:"u"`
}

type Info struct {
	Img struct {
		URL    string `json:"url"`
		Server string `json:"server"`
	} `json:"img"`
	Servers struct {
		Main      string `json:"main"`
		Secondary string `json:"secondary"`
		Compress  string `json:"compress"`
		Fourth    string `json:"fourth"`
	} `json:"servers"`
}

// GetChaptersList - ...
func GetChaptersList(mangaURL string) ([]data.ChaptersList, error) {
	var err error
	var chaptersList []data.ChaptersList
	dataRE := regexp.MustCompile(`(?i)window.__DATA__ = {.+};`)

	body, err := tools.GetPageCF(mangaURL)
	if err != nil {
		return chaptersList, err
	}

	rawData := strings.Trim(dataRE.FindString(string(body)), "window.__DATA__ = ;")

	chaptersRawData := ChaptersRawData{}
	err = json.Unmarshal([]byte(rawData), &chaptersRawData)
	if err != nil {
		return chaptersList, err
	}

	for _, ch := range chaptersRawData.Chapters.List {
		chapter := data.ChaptersList{
			Title: "(" + strconv.Itoa(ch.ChapterVolume) + "-" + ch.ChapterNumber + ") " + ch.ChapterName,
			Path:  "v" + strconv.Itoa(ch.ChapterVolume) + "/c" + ch.ChapterNumber,
		}

		chaptersList = append(chaptersList, chapter)
	}

	return tools.ReverseList(chaptersList), nil
}

// DownloadManga - ...
func DownloadManga(downData data.DownloadOpts) error {
	var err error
	var chaptersList []data.ChaptersList

	switch downData.Type {
	case "all":
		chaptersList, err = GetChaptersList(downData.MangaURL)
		if err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	case "chapters":
		chaptersRaw := strings.Split(strings.Trim(downData.Chapters, "[] \""), "\",\"")
		for _, ch := range chaptersRaw {
			chapter := data.ChaptersList{
				Path: ch,
			}
			chaptersList = append(chaptersList, chapter)
		}
	}

	chaptersTotal := len(chaptersList)
	chaptersCur := 0

	data.WSChan <- data.WSData{
		Cmd: "initProgress",
		Payload: map[string]interface{}{
			"valNow": 0,
			"valMax": chaptersTotal,
			"width":  0,
		},
	}

	for _, chapter := range chaptersList {
		DownloadChapter(downData, chapter)

		chaptersCur++

		time.Sleep(3 * time.Second)

		data.WSChan <- data.WSData{
			Cmd: "updateProgress",
			Payload: map[string]interface{}{
				"valNow": chaptersCur,
				"width":  tools.GetPercent(chaptersCur, chaptersTotal),
			},
		}
	}

	data.WSChan <- data.WSData{
		Cmd: "downloadComplete",
		Payload: map[string]interface{}{
			"text": "Скачивание завершено!",
		},
	}

	return nil
}

// DownloadChapter - ...
func DownloadChapter(downData data.DownloadOpts, curChapter data.ChaptersList) error {
	var err error

	data.WSChan <- data.WSData{
		Cmd: "updateLog",
		Payload: map[string]interface{}{
			"type": "std",
			"text": "Скачиваю главу: " + curChapter.Path,
		},
	}

	infoRE := regexp.MustCompile(`(?i)window.__info = {.+};`)
	pagesRE := regexp.MustCompile(`(?i)window.__pg = \[{.+}\];`)

	chapterURL := strings.TrimRight(downData.MangaURL, "/") + "/" + curChapter.Path

	body, err := tools.GetPageCF(chapterURL)
	if err != nil {
		return err
	}

	rawInfo := strings.Trim(infoRE.FindString(string(body)), "window.__info = ;")
	rawPages := strings.Trim(pagesRE.FindString(string(body)), "window.__pg = ;")

	info := Info{}
	pages := PagesList{}

	err = json.Unmarshal([]byte(rawInfo), &info)
	if err != nil {
		log.Println("UNM1")
		return err
	}

	err = json.Unmarshal([]byte(rawPages), &pages)
	if err != nil {
		log.Println("UNM2")
		return err
	}

	chapterPath := "Manga/" + downData.SavePath + "/" + curChapter.Path

	if _, err := os.Stat(chapterPath); os.IsNotExist(err) {
		os.MkdirAll(chapterPath, 0755)
	}

	var imgServer string

	switch info.Img.Server {
	case "main":
		imgServer = info.Servers.Main
	case "secondary":
		imgServer = info.Servers.Secondary
	case "compress":
		imgServer = info.Servers.Compress
	case "fourth":
		imgServer = info.Servers.Fourth
	}

	var savedFiles []string

	for _, page := range pages {
		imgURL := imgServer + info.Img.URL + page.URL

		client := grab.NewClient()
		client.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0"
		req, err := grab.NewRequest(chapterPath, imgURL)
		// req.HTTPRequest.Header.Set("Referer", chapterURL)
		if err != nil {
			log.Println("Grab req:", err)
		}
		resp := client.Do(req)
		if resp.Err() != nil {
			log.Println("Grab resp:", resp.Err())
		}
		savedFiles = append(savedFiles, resp.Filename)

		time.Sleep(1 * time.Second)
	}

	var wg sync.WaitGroup

	if downData.CBZ == "1" {
		wg.Add(1)

		data.WSChan <- data.WSData{
			Cmd: "updateLog",
			Payload: map[string]interface{}{
				"type": "std",
				"text": "- Создаю CBZ для главы",
			},
		}

		tools.CreateCBZ(chapterPath, &wg)
	}

	if downData.PDF == "1" {
		wg.Add(1)

		data.WSChan <- data.WSData{
			Cmd: "updateLog",
			Payload: map[string]interface{}{
				"type": "std",
				"text": "- Создаю PDF для главы",
			},
		}

		pdf.CreatePDF(chapterPath, savedFiles, &wg)
	}

	wg.Wait()

	if downData.Del == "1" {
		err := os.RemoveAll(chapterPath)
		if err != nil {
			log.Println("-- Ошибка при удалении файлов:", err)
		}
	}

	return nil
}
