package favs

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/goware/urlx"

	bolt "go.etcd.io/bbolt"

	"github.com/lirix360/ReadmangaGrabber/config"
	"github.com/lirix360/ReadmangaGrabber/db"
	"github.com/lirix360/ReadmangaGrabber/mangalib"
	"github.com/lirix360/ReadmangaGrabber/readmanga"
	"github.com/lirix360/ReadmangaGrabber/tools"
)

type FavData struct {
	ID   string `json:"id"`
	Lib  string `json:"lib"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func LoadFavs(w http.ResponseWriter, r *http.Request) {
	favsData := make(map[string]string)

	err := db.DBconn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("MangaFavs"))

		err := b.ForEach(func(k, v []byte) error {
			favData := FavData{}

			err := json.Unmarshal(v, &favData)
			if err != nil {
				return err
			}

			favsData[favData.ID] = favData.Name + " (" + favData.Lib + ")"

			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		slog.Error(
			"Ошибка при получении избранного из БД",
			slog.String("Message", err.Error()),
		)
		tools.SendError("Ошибка при получении избранного из БД.", w)
		return
	}

	resp := make(map[string]interface{})

	resp["status"] = "success"
	resp["favs"] = favsData

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}

func GetFav(w http.ResponseWriter, r *http.Request) {
	var err error

	err = r.ParseForm()
	if err != nil {
		slog.Error(
			"Ошибка при парсинге формы",
			slog.String("Message", err.Error()),
		)
		tools.SendError("Ошибка при парсинге формы.", w)
		return
	}

	favData := FavData{}

	err = db.DBconn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("MangaFavs"))
		v := b.Get([]byte(r.FormValue("favID")))

		err := json.Unmarshal(v, &favData)
		return err
	})
	if err != nil {
		slog.Error(
			"Ошибка при получении избранного из БД",
			slog.String("Message", err.Error()),
		)
		tools.SendError("Ошибка при получении избранного из БД.", w)
		return
	}

	resp := make(map[string]interface{})

	resp["status"] = "success"
	resp["fav"] = favData

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}

func SaveFav(w http.ResponseWriter, r *http.Request) {
	var err error

	err = r.ParseForm()
	if err != nil {
		slog.Error(
			"Ошибка при парсинге формы",
			slog.String("Message", err.Error()),
		)
		tools.SendError("Ошибка при парсинге формы.", w)
		return
	}

	url, _ := urlx.Parse(r.FormValue("mangaURL"))
	host, _, _ := urlx.SplitHostPort(url)

	mangaURL := strings.Split(url.String(), "?")[0]
	mangaID := tools.GetMD5(mangaURL)

	favData := FavData{}

	if slices.Contains(config.Cfg.CurrentURLs.MangaLib, host) {
		mangaInfo, err := mangalib.GetMangaInfo(mangaURL)
		if err != nil {
			slog.Error(
				"Ошибка при получении информации о манге",
				slog.String("Message", err.Error()),
			)
			tools.SendError("Ошибка при получении информации о манге.", w)
			return
		}

		favData.ID = mangaID
		favData.Lib = "MangaLib"
		favData.URL = mangaURL

		if config.Cfg.FavTitle == "ru" {
			favData.Name = mangaInfo.TitleRu
		} else {
			favData.Name = mangaInfo.TitleOrig
		}
	} else if slices.Contains(config.Cfg.CurrentURLs.ReadManga, host) {
		mangaInfo, err := readmanga.GetMangaInfo(mangaURL)
		if err != nil {
			slog.Error(
				"Ошибка при получении информации о манге",
				slog.String("Message", err.Error()),
			)
			tools.SendError("Ошибка при получении информации о манге.", w)
			return
		}

		favData.ID = mangaID
		favData.Lib = "ReadManga"
		favData.URL = mangaURL

		if config.Cfg.FavTitle == "ru" {
			favData.Name = mangaInfo.TitleRu
		} else {
			favData.Name = mangaInfo.TitleOrig
		}
	} else {
		tools.SendError("Указанный вами адрес не поддерживается.", w)
		return
	}

	favDataJSON, err := json.Marshal(favData)
	if err != nil {
		slog.Error(
			"Ошибка при запаковке данных для БД",
			slog.String("Message", err.Error()),
		)
		tools.SendError("Ошибка при запаковке данных для БД.", w)
		return
	}

	err = db.DBconn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("MangaFavs"))
		err := b.Put([]byte(mangaID), favDataJSON)
		return err
	})
	if err != nil {
		slog.Error(
			"Ошибка при вставке данных в БД",
			slog.String("Message", err.Error()),
		)
		tools.SendError("Ошибка при вставке данных в БД.", w)
		return
	}

	resp := make(map[string]interface{})

	resp["status"] = "success"
	resp["fav"] = favData

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}

func DeleteFav(w http.ResponseWriter, r *http.Request) {
	var err error

	err = r.ParseForm()
	if err != nil {
		slog.Error(
			"Ошибка при парсинге формы",
			slog.String("Message", err.Error()),
		)
		tools.SendError("Ошибка при парсинге формы.", w)
		return
	}

	mangaID := r.FormValue("favID")

	err = db.DBconn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("MangaFavs"))
		err := b.Delete([]byte(mangaID))
		return err
	})
	if err != nil {
		slog.Error(
			"Ошибка при удалении манги из БД",
			slog.String("Message", err.Error()),
		)
		tools.SendError("Ошибка при удалении манги из БД.", w)
		return
	}

	resp := make(map[string]interface{})

	resp["status"] = "success"

	respData, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}
