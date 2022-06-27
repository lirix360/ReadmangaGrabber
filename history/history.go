package history

import (
	"encoding/json"
	"net/http"
	"strings"

	bolt "go.etcd.io/bbolt"

	"github.com/goware/urlx"
	"github.com/lirix360/ReadmangaGrabber/db"
	"github.com/lirix360/ReadmangaGrabber/logger"
	"github.com/lirix360/ReadmangaGrabber/tools"
)

func LoadHistory(mangaID string) ([]string, error) {
	var err error
	var historyData []string

	err = db.DBconn.View(func(tx *bolt.Tx) error {
		var err error

		b := tx.Bucket([]byte("History"))
		v := b.Get([]byte(mangaID))

		if v != nil {
			err = json.Unmarshal(v, &historyData)
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	return historyData, nil
}

func LoadHistoryWeb(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logger.Log.Error("Ошибка при парсинге формы:", err)
		tools.SendError("Ошибка при парсинге формы.", w)
		return
	}

	url, _ := urlx.Parse(r.FormValue("mangaURL"))
	mangaURL := strings.Split(url.String(), "?")[0]
	mangaID := tools.GetMD5(mangaURL)

	historyData, err := LoadHistory(mangaID)
	if err != nil {
		logger.Log.Error("Ошибка при получении истории из БД:", err)
		tools.SendError("Ошибка при получении истории из БД.", w)
		return
	}

	resp := make(map[string]interface{})

	resp["status"] = "success"
	resp["history"] = historyData

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}

func SaveHistory(mangaID string, chapters []string) error {
	historyData, err := LoadHistory(mangaID)
	if err != nil {
		logger.Log.Error("Ошибка при получении истории из БД:", err)
		return err
	}

	summaryCh := append(chapters, historyData...)
	summaryCh = tools.RemoveDuplicateStr(summaryCh)

	chaptersJSON, err := json.Marshal(summaryCh)
	if err != nil {
		logger.Log.Error("Ошибка при запаковке данных для ДБ:", err)
		return err
	}

	err = db.DBconn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("History"))
		err := b.Put([]byte(mangaID), chaptersJSON)
		return err
	})
	if err != nil {
		logger.Log.Error("Ошибка при вставке данных в ДБ:", err)
		return err
	}

	return nil
}

func SaveHistoryWeb(w http.ResponseWriter, r *http.Request) {
	var chaptersList []string

	chaptersRaw := strings.Split(strings.Trim(r.FormValue("selectedChapters"), "[] \""), "\",\"")
	chaptersList = append(chaptersList, chaptersRaw...)

	url, _ := urlx.Parse(r.FormValue("mangaURL"))
	mangaURL := strings.Split(url.String(), "?")[0]
	mangaID := tools.GetMD5(mangaURL)

	err := SaveHistory(mangaID, chaptersList)
	if err != nil {
		tools.SendError("Ошибка при сохранении истории.", w)
		return
	}

	resp := make(map[string]interface{})

	resp["status"] = "success"

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}
