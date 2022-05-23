package manga

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/goware/urlx"

	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/logger"
	"github.com/lirix360/ReadmangaGrabber/mangalib"
	"github.com/lirix360/ReadmangaGrabber/readmanga"
)

func GetChaptersList(w http.ResponseWriter, r *http.Request) {
	var err error
	var hasError bool
	var errText = "При получении списка глав произошла ошибка. Подробности в лог-файле."
	var chaptersList []data.ChaptersList
	var transList []data.RMTranslators

	url, _ := urlx.Parse(r.FormValue("mangaURL"))
	host, _, _ := urlx.SplitHostPort(url)

	mangaURL := strings.Split(url.String(), "?")[0]

	switch host {
	case "mangalib.me":
		chaptersList, err = mangalib.GetChaptersList(mangaURL)
		if err != nil {
			hasError = true
			logger.Log.Error("Ошибка при получении списка глав:", err)
		}
	case "readmanga.io", "mintmanga.live", "selfmanga.live", "23.allhen.online":
		chaptersList, transList, err = readmanga.GetChaptersList(mangaURL)
		if err != nil {
			hasError = true
			logger.Log.Error("Ошибка при получении списка глав:", err)
		}
	default:
		hasError = true
		errText = "Указанный вами адрес не поддерживается."
	}

	resp := make(map[string]interface{})

	if hasError {
		resp["status"] = "error"
		resp["payload"] = errText
	} else {
		if len(chaptersList) > 0 {
			resp["status"] = "success"
			resp["payload"] = chaptersList
			resp["translators"] = transList
		} else {
			resp["status"] = "error"
			resp["payload"] = "Глав не найдено. Проверьте правильность ввода адреса манги."
		}
	}

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}

func DownloadManga(w http.ResponseWriter, r *http.Request) {
	downloadOpts := data.DownloadOpts{
		Type:      r.FormValue("downloadType"),
		Chapters:  r.FormValue("selectedChapters"),
		PDF:       r.FormValue("optPDF"),
		CBZ:       r.FormValue("optCBZ"),
		Del:       r.FormValue("optDEL"),
		PrefTrans: r.FormValue("optPrefTrans"),
	}

	url, _ := urlx.Parse(r.FormValue("mangaURL"))
	host, _, _ := urlx.SplitHostPort(url)

	downloadOpts.MangaURL = strings.Split(url.String(), "?")[0]
	downloadOpts.SavePath = strings.Trim(url.Path, "/")

	switch host {
	case "mangalib.me":
		go mangalib.DownloadManga(downloadOpts)
	case "readmanga.io", "mintmanga.live", "selfmanga.live", "23.allhen.online":
		go readmanga.DownloadManga(downloadOpts)
	}

	resp := make(map[string]interface{})
	resp["status"] = "OK"

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}
