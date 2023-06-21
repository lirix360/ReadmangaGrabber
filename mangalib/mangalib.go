package mangalib

import (
	"encoding/json"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cavaliergopher/grab/v3"

	"github.com/lirix360/ReadmangaGrabber/config"
	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/history"
	"github.com/lirix360/ReadmangaGrabber/logger"
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

func GetMangaInfo(mangaURL string) (data.MangaInfo, error) {
	var err error
	var mangaInfo data.MangaInfo

	body, err := tools.GetPageCF(mangaURL)
	if err != nil {
		return mangaInfo, err
	}

	chaptersPage, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return mangaInfo, err
	}

	origTitle := chaptersPage.Find(".media-name__alt").Text()

	if origTitle == "" {
		origTitle = chaptersPage.Find(".media-name__main").Text()
	}

	mangaInfo.TitleOrig = origTitle
	mangaInfo.TitleRu = chaptersPage.Find(".media-name__main").Text()

	return mangaInfo, nil
}

func GetChaptersList(mangaURL string) ([]data.ChaptersList, error) {
	var err error
	var chaptersList []data.ChaptersList
	dataRE := regexp.MustCompile(`(?i)window.__DATA__ = {.+};`)

	body, err := tools.GetPageCF(mangaURL)
	if err != nil {
		return chaptersList, err
	}

	buf := new(strings.Builder)
	_, err = io.Copy(buf, body)
	if err != nil {
		return chaptersList, err
	}

	rawData := strings.Trim(dataRE.FindString(buf.String()), "window.__DATA__ = ;")

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

func DownloadManga(downData data.DownloadOpts) error {
	var err error
	var chaptersList []data.ChaptersList
	var saveChapters []string
	savedFilesByVol := make(map[string][]string)

	switch downData.Type {
	case "all":
		chaptersList, err = GetChaptersList(downData.MangaURL)
		if err != nil {
			logger.Log.Error("Ошибка при получении списка глав:", err)
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
		volume := strings.Split(chapter.Path, "/")[0]

		chSavedFiles, err := DownloadChapter(downData, chapter)
		if err != nil {
			data.WSChan <- data.WSData{
				Cmd: "updateLog",
				Payload: map[string]interface{}{
					"type": "err",
					"text": "-- Ошибка при скачивании главы:" + err.Error(),
				},
			}
		}

		savedFilesByVol[volume] = append(savedFilesByVol[volume], chSavedFiles...)

		chaptersCur++

		saveChapters = append(saveChapters, chapter.Path)

		time.Sleep(time.Duration(config.Cfg.Mangalib.TimeoutChapter) * time.Millisecond)

		data.WSChan <- data.WSData{
			Cmd: "updateProgress",
			Payload: map[string]interface{}{
				"valNow": chaptersCur,
				"width":  tools.GetPercent(chaptersCur, chaptersTotal),
			},
		}
	}

	if downData.PDFvol == "1" {
		data.WSChan <- data.WSData{
			Cmd: "updateLog",
			Payload: map[string]interface{}{
				"type": "std",
				"text": "Создаю PDF для томов",
			},
		}

		chapterPath := path.Join(config.Cfg.Savepath, downData.SavePath)

		pdf.CreateVolPDF(chapterPath, savedFilesByVol, downData.Del)
	}

	mangaID := tools.GetMD5(downData.MangaURL)
	history.SaveHistory(mangaID, saveChapters)

	data.WSChan <- data.WSData{
		Cmd: "downloadComplete",
		Payload: map[string]interface{}{
			"text": "Скачивание завершено!",
		},
	}

	return nil
}

func DownloadChapter(downData data.DownloadOpts, curChapter data.ChaptersList) ([]string, error) {
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
		return nil, err
	}

	buf := new(strings.Builder)
	_, err = io.Copy(buf, body)
	if err != nil {
		return nil, err
	}

	rawInfo := strings.Trim(infoRE.FindString(buf.String()), "window.__info = ;")
	rawPages := strings.Trim(pagesRE.FindString(buf.String()), "window.__pg = ;")

	info := Info{}
	pages := PagesList{}

	err = json.Unmarshal([]byte(rawInfo), &info)
	if err != nil {
		logger.Log.Error("Ошибка при распаковке данных (Info):", err)
		return nil, err
	}

	err = json.Unmarshal([]byte(rawPages), &pages)
	if err != nil {
		logger.Log.Error("Ошибка при распаковке данных (Pages):", err)
		return nil, err
	}

	chapterPath := path.Join(config.Cfg.Savepath, downData.SavePath, curChapter.Path)

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
		client.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/109.0"
		req, err := grab.NewRequest(chapterPath, imgURL)
		req.HTTPRequest.Header.Set("Referer", chapterURL)
		if err != nil {
			logger.Log.Error("Ошибка при скачивании страницы:", err)
			data.WSChan <- data.WSData{
				Cmd: "updateLog",
				Payload: map[string]interface{}{
					"type": "err",
					"text": "-- Ошибка при скачивании страницы:" + err.Error(),
				},
			}
			continue
		}
		resp := client.Do(req)
		if resp.Err() != nil {
			logger.Log.Error("Ошибка при скачивании страницы:", resp.Err())
			data.WSChan <- data.WSData{
				Cmd: "updateLog",
				Payload: map[string]interface{}{
					"type": "err",
					"text": "-- Ошибка при скачивании страницы:" + err.Error(),
				},
			}
			continue
		}
		savedFiles = append(savedFiles, resp.Filename)

		time.Sleep(time.Duration(config.Cfg.Mangalib.TimeoutImage) * time.Millisecond)
	}

	if downData.CBZ == "1" {
		data.WSChan <- data.WSData{
			Cmd: "updateLog",
			Payload: map[string]interface{}{
				"type": "std",
				"text": "- Создаю CBZ для главы",
			},
		}

		tools.CreateCBZ(chapterPath)
	}

	if downData.PDFch == "1" {
		data.WSChan <- data.WSData{
			Cmd: "updateLog",
			Payload: map[string]interface{}{
				"type": "std",
				"text": "- Создаю PDF для главы",
			},
		}

		pdf.CreatePDF(chapterPath, savedFiles)
	}

	if downData.PDFvol != "1" && downData.Del == "1" {
		err := os.RemoveAll(chapterPath)
		if err != nil {
			logger.Log.Error("Ошибка при удалении файлов:", err)
		}
	}

	return savedFiles, nil
}
