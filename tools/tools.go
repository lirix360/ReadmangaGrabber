package tools

import (
	"bytes"
	"compress/flate"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"os"
	"strings"

	"github.com/aki237/nscjar"
	"github.com/goware/urlx"
	"github.com/headzoo/surf"
	"github.com/mholt/archiver"

	"github.com/lirix360/ReadmangaGrabber/config"
	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/logger"
)

func GetAppVer(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]interface{})

	resp["status"] = "success"
	resp["appver"] = config.APPver

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}

func GetMD5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func SendError(errText string, w http.ResponseWriter) {
	resp := make(map[string]interface{})

	resp["status"] = "error"
	resp["errtext"] = errText

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}

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

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:108.0) Gecko/20100101 Firefox/108.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func GetPageWithCookies(pageURL string) (io.ReadCloser, error) {
	url, _ := urlx.Parse(pageURL)
	host, _, _ := urlx.SplitHostPort(url)

	cookieFile := ""

	switch host {
	case "readmanga.live":
		cookieFile = "readmanga.txt"
	case "mintmanga.live":
		cookieFile = "mintmanga.txt"
	case "selfmanga.live":
		cookieFile = "selfmanga.txt"
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:108.0) Gecko/20100101 Firefox/108.0")

	if IsFileExist(cookieFile) {
		f, err := os.Open(cookieFile)
		if err != nil {
			return nil, err
		}

		jar := nscjar.Parser{}

		cookies, err := jar.Unmarshal(f)
		if err != nil {
			return nil, err
		}

		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func GetPageCF(pageURL string) (io.ReadCloser, error) {
	var body bytes.Buffer
	bow := surf.NewBrowser()

	err := bow.Open(pageURL)
	if err != nil {
		return nil, err
	}

	bow.Download(&body)

	return io.NopCloser(strings.NewReader(body.String())), nil
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
		percent = int(math.Round((100 / float64(total)) * float64(cur)))
	}

	return percent
}

func IsFileExist(filePath string) bool {
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

func RemoveDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
