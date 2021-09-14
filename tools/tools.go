package tools

import (
	"compress/flate"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	scraper "github.com/byung82/go-cloudflare-scraper"
	"github.com/mholt/archiver"

	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/logger"
)

// ReverseList - ...
func ReverseList(chaptersList []data.ChaptersList) []data.ChaptersList {
	newChaptersList := make([]data.ChaptersList, 0, len(chaptersList))

	for i := len(chaptersList) - 1; i >= 0; i-- {
		newChaptersList = append(newChaptersList, chaptersList[i])
	}

	return newChaptersList
}

// GetPage - ...
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

// GetPageCF - ...
func GetPageCF(pageURL string) ([]byte, error) {
	var body []byte

	scraper, err := scraper.NewTransport(http.DefaultTransport)
	if err != nil {
		return body, err
	}

	c := http.Client{Transport: scraper}
	res, err := c.Get(pageURL)
	if err != nil {
		return body, err
	}
	defer res.Body.Close()

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return body, err
	}

	return body, nil
}

// CreateCBZ - ...
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

// GetPercent - ...
func GetPercent(cur, total int) int {
	var percent int

	if cur == total {
		percent = 100
	} else {
		percent = (100 / total) * cur
	}

	return percent
}
