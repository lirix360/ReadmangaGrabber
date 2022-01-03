package tools

import (
	"bytes"
	"compress/flate"
	"io"
	"net/http"
	"os"

	"github.com/headzoo/surf"
	"github.com/mholt/archiver"

	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/logger"
)

func ReverseList(chaptersList []data.ChaptersList) []data.ChaptersList {
	newChaptersList := make([]data.ChaptersList, 0, len(chaptersList))

	for i := len(chaptersList) - 1; i >= 0; i-- {
		newChaptersList = append(newChaptersList, chaptersList[i])
	}

	return newChaptersList
}

func GetPage(pageURL string) (io.ReadCloser, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:92.0) Gecko/20100101 Firefox/92.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func GetPageCF(pageURL string) (string, error) {
	var body bytes.Buffer
	bow := surf.NewBrowser()

	err := bow.Open(pageURL)
	if err != nil {
		return "", err
	}

	bow.Download(&body)

	return body.String(), nil
}

func SavePage(body string) {
	file, err := os.Create("saved.html")
	if err != nil {
		logger.Log.Error(err)
	} else {
		file.WriteString(body)
	}
	file.Close()
}

func CreateCBZ(chapterPath string) error {
	z := archiver.Zip{
		CompressionLevel:       flate.NoCompression,
		MkdirAll:               true,
		SelectiveCompression:   true,
		ContinueOnError:        false,
		OverwriteExisting:      true,
		ImplicitTopLevelFolder: false,
	}

	err := z.Archive([]string{chapterPath}, chapterPath+".zip")
	if err != nil {
		logger.Log.Error("Ошибка при создании архива ("+chapterPath+".zip):", err)
		data.WSChan <- data.WSData{
			Cmd: "updateLog",
			Payload: map[string]interface{}{
				"type": "err",
				"text": "-- Ошибка при создании CBZ:" + err.Error(),
			},
		}
		return err
	}
	defer z.Close()

	err = os.Rename(chapterPath+".zip", chapterPath+".cbz")
	if err != nil {
		logger.Log.Error("Ошибка при переименовании архива ("+chapterPath+".zip):", err)
		data.WSChan <- data.WSData{
			Cmd: "updateLog",
			Payload: map[string]interface{}{
				"type": "err",
				"text": "-- Ошибка при создании CBZ:" + err.Error(),
			},
		}
		return err
	}

	return nil
}

func GetPercent(cur, total int) int {
	var percent int

	if cur == total {
		percent = 100
	} else {
		percent = (100 / total) * cur
	}

	return percent
}
