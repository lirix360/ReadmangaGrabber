package manga

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/goware/urlx"

	"github.com/lirix360/ReadmangaGrabber/config"
	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/mangalib"
	"github.com/lirix360/ReadmangaGrabber/readmanga"
	"github.com/lirix360/ReadmangaGrabber/tools"
)

func GetChaptersList(w http.ResponseWriter, r *http.Request) {
	var err error
	var isMtr bool
	var userHash string
	var rawChaptersList []data.ChaptersList
	chaptersList := make(map[string][]data.ChaptersList)
	var transList []data.RMTranslators

	url, _ := urlx.Parse(r.FormValue("mangaURL"))
	host, _, _ := urlx.SplitHostPort(url)

	mangaURL := strings.Split(url.String(), "?")[0]

	if tools.CheckSource(config.Cfg.CurrentURLs.MangaLib, host) {
		rawChaptersList, err = mangalib.GetChaptersList(mangaURL)
		if err != nil {
			slog.Error(
				"Ошибка при получении списка глав",
				slog.String("Message", err.Error()),
			)
			tools.SendError("При получении списка глав произошла ошибка. Подробности в лог-файле.", w)
			return
		}

		for _, ch := range rawChaptersList {
			parts := strings.Split(ch.Path, "/")
			volNum := strings.TrimLeft(parts[0], "v")
			chaptersList[volNum] = append(chaptersList[volNum], ch)
		}
	} else if tools.CheckSource(config.Cfg.CurrentURLs.ReadManga, host) {
		rawChaptersList, transList, isMtr, userHash, err = readmanga.GetChaptersList(mangaURL)
		if err != nil {
			slog.Error(
				"Ошибка при получении списка глав",
				slog.String("Message", err.Error()),
			)
			tools.SendError("При получении списка глав произошла ошибка. Подробности в лог-файле.", w)
			return
		}

		for _, ch := range rawChaptersList {
			parts := strings.Split(ch.Path, "/")
			volNum := strings.TrimLeft(parts[0], "vol")
			chaptersList[volNum] = append(chaptersList[volNum], ch)
		}
	} else {
		slog.Error(
			"Ошибка при получении списка глав",
			slog.String("Message", "Указанный адрес не поддерживается"),
		)
		tools.SendError("Указанный вами адрес не поддерживается.", w)
		return
	}

	resp := make(map[string]interface{})

	if len(chaptersList) > 0 {
		resp["status"] = "success"
		resp["is_mtr"] = isMtr
		resp["user_hash"] = userHash
		resp["payload"] = chaptersList
		resp["translators"] = transList
	} else {
		resp["status"] = "error"
		resp["errtext"] = "Глав не найдено. Проверьте правильность ввода адреса манги."
	}

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}

func DownloadManga(w http.ResponseWriter, r *http.Request) {
	isMtr := false

	if r.FormValue("isMtr") == "true" {
		isMtr = true
	}

	downloadOpts := data.DownloadOpts{
		Mtr:       isMtr,
		Type:      r.FormValue("downloadType"),
		Chapters:  r.FormValue("selectedChapters"),
		PDFch:     r.FormValue("optPDFch"),
		PDFvol:    r.FormValue("optPDFvol"),
		PDFall:    r.FormValue("optPDFall"),
		CBZ:       r.FormValue("optCBZ"),
		Del:       r.FormValue("optDEL"),
		PrefTrans: r.FormValue("optPrefTrans"),
		UserHash:  r.FormValue("userHash"),
	}

	url, _ := urlx.Parse(r.FormValue("mangaURL"))
	host, _, _ := urlx.SplitHostPort(url)

	downloadOpts.MangaURL = strings.Split(url.String(), "?")[0]
	downloadOpts.SavePath = strings.Trim(url.Path, "/")

	if tools.CheckSource(config.Cfg.CurrentURLs.MangaLib, host) {
		go mangalib.DownloadManga(downloadOpts)
	} else if tools.CheckSource(config.Cfg.CurrentURLs.ReadManga, host) {
		go readmanga.DownloadManga(downloadOpts)
	}

	resp := make(map[string]interface{})
	resp["status"] = "OK"

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}
