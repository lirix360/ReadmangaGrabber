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
	"strconv"
	"strings"

	"github.com/aki237/nscjar"
	"github.com/goware/urlx"
	"github.com/headzoo/surf"
	"github.com/mholt/archiver"

	"github.com/lirix360/ReadmangaGrabber/config"
	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/logger"
)

func CheckUpdate(w http.ResponseWriter, r *http.Request) {
	tmpData := map[string]string{}
	jsonResp := make(map[string]interface{})

	hasError := false

	resp, err := http.Get("https://raw.githubusercontent.com/lirix360/ReadmangaGrabber/master/version.json")
	if err != nil {
		logger.Log.Error(err)
		hasError = true
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error(err)
		hasError = true
	}

	err = json.Unmarshal(body, &tmpData)
	if err != nil {
		logger.Log.Error(err)
		hasError = true
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
		logger.Log.Error("Ошибка при парсинге формы:", err)
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

func GetPage(pageURL string) (io.ReadCloser, error) {
	url, _ := urlx.Parse(pageURL)
	host, _, _ := urlx.SplitHostPort(url)

	cookieFile := host + ".txt"

	client := &http.Client{}

	if config.Cfg.Proxy.Use.Readmanga {
		proxyUrl, err := url.Parse(config.Cfg.Proxy.Type + "://" + config.Cfg.Proxy.Addr + ":" + config.Cfg.Proxy.Port)
		logger.Log.Info("Proxy:", proxyUrl.String())
		if err != nil {
			return nil, err
		}

		client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
	}

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		logger.Log.Error("Ошибка при инициализации запроса:", err)
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/109.0")

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
		logger.Log.Error("Ошибка при выполнении запроса:", err)
		return nil, err
	}

	return resp.Body, nil
}

func GetPageCF(pageURL string) (io.ReadCloser, error) {
	var body bytes.Buffer
	bow := surf.NewBrowser()

	url, _ := urlx.Parse(pageURL)
	host, _, _ := urlx.SplitHostPort(url)

	cookieFile := host + ".txt"

	if config.Cfg.Proxy.Use.Mangalib {
		proxyUrl, err := url.Parse(config.Cfg.Proxy.Type + "://" + config.Cfg.Proxy.Addr + ":" + config.Cfg.Proxy.Port)
		logger.Log.Info("Proxy:", proxyUrl.String())
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
		logger.Log.Error("Ошибка при инициализации запроса:", err)
		return nil, err
	}

	_, err = bow.Download(&body)
	if err != nil {
		logger.Log.Error("Ошибка при выполнении запроса:", err)
		return nil, err
	}

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
