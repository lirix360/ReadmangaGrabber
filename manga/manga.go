package manga

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/goware/urlx"

	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/mangalib"
	"github.com/lirix360/ReadmangaGrabber/readmanga"
)

// GetChaptersList - ...
func GetChaptersList(w http.ResponseWriter, r *http.Request) {
	var err error
	var hasError bool
	var errText = "При получении списка глав произошла ошибка. Подробности в лог-файле."
	var chaptersList []data.ChaptersList

	mangaURL := r.FormValue("mangaURL")

	url, _ := urlx.Parse(mangaURL)
	host, _, _ := urlx.SplitHostPort(url)

	switch host {
	case "mangalib.me":
		chaptersList, err = mangalib.GetChaptersList(mangaURL)
		if err != nil {
			hasError = true
			log.Println("Ошибка при получении списка глав:", err)
		}
	case "readmanga.live", "mintmanga.live", "selfmanga.live", "wwv.allhen.live":
		chaptersList, err = readmanga.GetChaptersList(mangaURL)
		if err != nil {
			hasError = true
			log.Println("Ошибка при получении списка глав:", err)
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
		} else {
			resp["status"] = "error"
			resp["payload"] = "Глав не найдено. Проверьте правильность ввода адреса манги."
		}
	}

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}

// DownloadManga - ...
func DownloadManga(w http.ResponseWriter, r *http.Request) {
	downloadOpts := data.DownloadOpts{
		Type:     r.FormValue("downloadType"),
		MangaURL: r.FormValue("mangaURL"),
		Chapters: r.FormValue("selectedChapters"),
		PDF:      r.FormValue("optPDF"),
		CBZ:      r.FormValue("optCBZ"),
		Del:      r.FormValue("optDEL"),
	}

	url, _ := urlx.Parse(downloadOpts.MangaURL)
	host, _, _ := urlx.SplitHostPort(url)

	downloadOpts.SavePath = strings.Trim(url.Path, "/")

	switch host {
	case "mangalib.me":
		go mangalib.DownloadManga(downloadOpts)
	case "readmanga.live", "mintmanga.live", "selfmanga.live", "wwv.allhen.live":
		go readmanga.DownloadManga(downloadOpts)
	}

	resp := make(map[string]interface{})
	resp["status"] = "OK"

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}
