package readmanga

import (
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cavaliergopher/grab/v3"
	"github.com/goware/urlx"

	"github.com/lirix360/ReadmangaGrabber/config"
	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/history"
	"github.com/lirix360/ReadmangaGrabber/logger"
	"github.com/lirix360/ReadmangaGrabber/pdf"
	"github.com/lirix360/ReadmangaGrabber/tools"
)

func GetMangaInfo(mangaURL string) (data.MangaInfo, error) {
	var err error
	var mangaInfo data.MangaInfo

	pageBody, err := tools.GetPage(mangaURL)
	if err != nil {
		return mangaInfo, err
	}

	chaptersPage, err := goquery.NewDocumentFromReader(pageBody)
	if err != nil {
		return mangaInfo, err
	}

	origTitle := chaptersPage.Find(".original-name").Text()

	if origTitle == "" && chaptersPage.Find(".eng-name").Text() != "" {
		origTitle = chaptersPage.Find(".eng-name").Text()
	}

	if origTitle == "" {
		origTitle = chaptersPage.Find(".name").Text()
	}

	mangaInfo.TitleOrig = origTitle
	mangaInfo.TitleRu = chaptersPage.Find(".name").Text()

	return mangaInfo, nil
}

func GetChaptersList(mangaURL string) ([]data.ChaptersList, []data.RMTranslators, error) {
	var err error
	var chaptersList []data.ChaptersList
	var transList []data.RMTranslators

	pageBody, err := tools.GetPage(mangaURL)
	if err != nil {
		return chaptersList, transList, err
	}

	chaptersPage, err := goquery.NewDocumentFromReader(pageBody)
	if err != nil {
		return chaptersList, transList, err
	}

	chaptersPage.Find(".chapters-link a").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		path := strings.Split(strings.Trim(href, "/"), "/")

		chapter := data.ChaptersList{
			Title: strings.Trim(s.Text(), "\n "),
			Path:  path[1] + "/" + path[2],
		}

		chaptersList = append(chaptersList, chapter)
	})

	chaptersPage.Find("#translation > option").Each(func(i int, s *goquery.Selection) {
		trID, _ := s.Attr("value")
		trName := s.Text()

		trans := data.RMTranslators{
			ID:   trID,
			Name: trName,
		}

		transList = append(transList, trans)
	})

	return tools.ReverseList(chaptersList), transList, nil
}

func DownloadManga(downData data.DownloadOpts) error {
	var err error
	var chaptersList []data.ChaptersList
	var saveChapters []string
	savedFilesByVol := make(map[string][]string)

	switch downData.Type {
	case "all":
		chaptersList, _, err = GetChaptersList(downData.MangaURL)
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

		time.Sleep(time.Duration(config.Cfg.Readmanga.TimeoutChapter) * time.Microsecond)

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

	chapterURL := strings.TrimRight(downData.MangaURL, "/") + "/" + curChapter.Path

	refURL, _ := urlx.Parse(downData.MangaURL)

	var imageLinks []string

	ptOpt := ""

	if downData.PrefTrans != "" {
		ptOpt = "&tran=" + downData.PrefTrans
	}

	page, err := tools.GetPage(chapterURL + "?mtr=1" + ptOpt)
	if err != nil {
		logger.Log.Error("Ошибка при получении страниц:", err)
		return nil, err
	}

	pageBody, err := ioutil.ReadAll(page)
	if err != nil {
		logger.Log.Error("Ошибка при получении страниц:", err)
		return nil, err
	}

	r := regexp.MustCompile(`rm_h\.initReader\(\s\[\d,\d\],\s\[(.+)\],\s0,\sfalse.+\);`)

	chList := r.FindStringSubmatch(string(pageBody))

	if len(chList) > 0 && chList[1] != "" {
		imageParts := strings.Split(strings.Trim(chList[1], "[]"), "],[")

		for i := 0; i < len(imageParts); i++ {
			tmpParts := strings.Split(imageParts[i], ",")

			imageLinks = append(imageLinks, strings.Trim(tmpParts[0], "\"'")+strings.Trim(tmpParts[2], "\"'"))
		}
	}

	chapterPath := path.Join(config.Cfg.Savepath, downData.SavePath, curChapter.Path)

	if _, err := os.Stat(chapterPath); os.IsNotExist(err) {
		os.MkdirAll(chapterPath, 0755)
	}

	var savedFiles []string

	for _, imgURL := range imageLinks {
		client := grab.NewClient()
		client.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:92.0) Gecko/20100101 Firefox/92.0"

		req, err := grab.NewRequest(chapterPath, imgURL)
		if err != nil {
			logger.Log.Error("Ошибка при скачивании страницы:", err)
			return nil, err
		}

		req.HTTPRequest.Header.Set("Referer", refURL.Scheme+"://"+refURL.Host+"/")

		resp := client.Do(req)
		if resp.Err() != nil {
			logger.Log.Error("Ошибка при скачивании страницы:", resp.Err())
			return nil, err
		}

		savedFiles = append(savedFiles, resp.Filename)

		time.Sleep(time.Duration(config.Cfg.Readmanga.TimeoutImage) * time.Microsecond)
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
