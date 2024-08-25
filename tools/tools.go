package tools

import (
	"bytes"
	"compress/flate"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/aki237/nscjar"
	"github.com/goware/urlx"
	"github.com/headzoo/surf"
	"github.com/mholt/archiver/v3"

	"github.com/lirix360/ReadmangaGrabber/config"
	"github.com/lirix360/ReadmangaGrabber/data"
)

func CheckUpdate(w http.ResponseWriter, r *http.Request) {
	tmpData := map[string]string{}
	jsonResp := make(map[string]interface{})
	body := []byte{}

	hasError := false

	resp, err := http.Get("https://raw.githubusercontent.com/lirix360/ReadmangaGrabber/master/version.json")
	if err != nil {
		slog.Error(
			"Ошибка при запросе информации о версии",
			slog.String("Message", err.Error()),
		)
		hasError = true
	}
	defer resp.Body.Close()

	if !hasError {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			slog.Error(
				"Ошибка при получении информации о версии",
				slog.String("Message", err.Error()),
			)
			hasError = true
		}
	}

	if !hasError {
		err = json.Unmarshal(body, &tmpData)
		if err != nil {
			slog.Error(
				"Ошибка при обработке информации о версии",
				slog.String("Message", err.Error()),
			)
			hasError = true
		}
	}

	if !hasError {
		lastVer, _ := strconv.Atoi(tmpData["last_version"])
		appVer, _ := strconv.Atoi(config.APPver)

		jsonResp["status"] = "success"
		jsonResp["has_update"] = false

		if appVer < lastVer {
			jsonResp["has_update"] = true
		}
	} else {
		jsonResp["status"] = "error"
	}

	respData, _ := json.Marshal(jsonResp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}

func CheckAuth(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		slog.Error(
			"Ошибка при парсинге формы",
			slog.String("Message", err.Error()),
		)
		SendError("Ошибка при парсинге формы.", w)
		return
	}

	urlStr := r.FormValue("URL")

	url, _ := urlx.Parse(urlStr)
	host, _, _ := urlx.SplitHostPort(url)

	resp := make(map[string]interface{})
	resp["status"] = "error"

	if IsFileExist(host + ".txt") {
		resp["status"] = "success"
	}

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

func GetPageCF(pageURL string) (io.ReadCloser, error) {
	var body bytes.Buffer
	bow := surf.NewBrowser()
	bow.SetUserAgent(config.Cfg.UserAgent)

	url, _ := urlx.Parse(pageURL)
	host, _, _ := urlx.SplitHostPort(url)

	cookieFile := host + ".txt"

	useProxy := false

	if slices.Contains(config.Cfg.CurrentURLs.MangaLib, host) {
		useProxy = config.Cfg.Proxy.Use.Mangalib
	} else if slices.Contains(config.Cfg.CurrentURLs.ReadManga, host) {
		useProxy = config.Cfg.Proxy.Use.Readmanga
	}

	if useProxy {
		proxyUrl, err := url.Parse(config.Cfg.Proxy.Type + "://" + config.Cfg.Proxy.Addr + ":" + config.Cfg.Proxy.Port)
		slog.Info(
			"Используется прокси",
			slog.String("Server", proxyUrl.String()),
		)
		if err != nil {
			return nil, err
		}

		bow.SetTransport(&http.Transport{Proxy: http.ProxyURL(proxyUrl)})
	}

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

		url, _ := urlx.Parse(pageURL)

		bow.CookieJar().SetCookies(url, cookies)
	}

	err := bow.Open(pageURL)
	if err != nil {
		slog.Error(
			"Ошибка при инициализации запроса",
			slog.String("Message", err.Error()),
		)
		return nil, err
	}

	_, err = bow.Download(&body)
	if err != nil {
		slog.Error(
			"Ошибка при выполнении запроса",
			slog.String("Message", err.Error()),
		)
		return nil, err
	}

	return io.NopCloser(strings.NewReader(body.String())), nil
}

func SavePage(body string) {
	file, err := os.Create("saved.html")
	if err != nil {
		slog.Error(
			"Ошибка при сохранении страницы",
			slog.String("Message", err.Error()),
		)
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
		slog.Error(
			"Ошибка при создании архива ("+chapterPath+".zip)",
			slog.String("Message", err.Error()),
		)
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
		slog.Error(
			"Ошибка при переименовании архива ("+chapterPath+".zip)",
			slog.String("Message", err.Error()),
		)
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
