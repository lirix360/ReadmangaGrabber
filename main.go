package main

import (
	"embed"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/olahol/melody.v1"

	browser "github.com/pkg/browser"

	"github.com/lirix360/ReadmangaGrabber/config"
	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/db"
	"github.com/lirix360/ReadmangaGrabber/favs"
	"github.com/lirix360/ReadmangaGrabber/history"
	"github.com/lirix360/ReadmangaGrabber/manga"
	"github.com/lirix360/ReadmangaGrabber/tools"
)

//go:embed index.html
//go:embed assets/*
var webUI embed.FS

func init() {
	var err error
	logFileName := "grabber_log.log"

	if _, err = os.Stat(logFileName); err == nil {
		err = os.Remove(logFileName)
		if err != nil {
			slog.Error(
				"Ошибка при удалении старого лог-файла",
				slog.String("Message", err.Error()),
			)
			os.Exit(1)
		}
	}

	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		slog.Error(
			"Ошибка при открытии лог-файла",
			slog.String("Message", err.Error()),
		)
		os.Exit(1)
	}

	w := io.MultiWriter(os.Stdout, logFile)

	logger := slog.New(slog.NewTextHandler(w, nil))

	slog.SetDefault(logger)
}

func main() {
	var err error

	slog.Info("Запуск приложения")

	r := mux.NewRouter()
	m := melody.New()

	r.HandleFunc("/checkUpdate", tools.CheckUpdate)
	r.HandleFunc("/checkAuth", tools.CheckAuth)

	r.HandleFunc("/saveConfig", config.SaveConfig)
	r.HandleFunc("/loadConfig", config.LoadConfig)

	r.HandleFunc("/favsLoad", favs.LoadFavs)
	r.HandleFunc("/favsGet", favs.GetFav)
	r.HandleFunc("/favsSave", favs.SaveFav)
	r.HandleFunc("/favsDelete", favs.DeleteFav)

	r.HandleFunc("/loadHistory", history.LoadHistoryWeb)
	r.HandleFunc("/saveHistory", history.SaveHistoryWeb)

	r.HandleFunc("/getChaptersList", manga.GetChaptersList)
	r.HandleFunc("/downloadManga", manga.DownloadManga)

	r.HandleFunc("/closeApp", func(w http.ResponseWriter, r *http.Request) {
		db.DBconn.Close()
		slog.Info("Закрытие приложения")
		os.Exit(0)
	})

	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		err := m.HandleRequest(w, r)
		if err != nil {
			slog.Error(
				"Ошибка при обработке данных WS",
				slog.String("Message", err.Error()),
			)
		}
	})

	go func() {
		for {
			msgData := <-data.WSChan
			wsData, err := json.Marshal(msgData)
			if err != nil {
				slog.Error(
					"Ошибка при сериализации данных для отправки через WS",
					slog.String("Message", err.Error()),
				)
				continue
			}
			err = m.Broadcast(wsData)
			if err != nil {
				slog.Error(
					"Ошибка при сериализации данных для отправки через WS",
					slog.String("Message", err.Error()),
				)
				continue
			}
		}
	}()

	r.PathPrefix("/").Handler(http.FileServer(http.FS(webUI)))

	srv := &http.Server{
		Handler:      r,
		Addr:         config.Cfg.Server.Addr + ":" + config.Cfg.Server.Port,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	if config.Cfg.ShowGUI {
		err = browser.OpenURL("http://" + config.Cfg.Server.Addr + ":" + config.Cfg.Server.Port + "/")
		if err != nil {
			slog.Error(
				"Ошибка при открытии браузера",
				slog.String("Message", err.Error()),
			)
			os.Exit(1)
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		<-signalChan
		data.WSChan <- data.WSData{Cmd: "closeApp"}
		db.DBconn.Close()
		slog.Info("Закрытие приложения")
		os.Exit(0)
	}()

	err = srv.ListenAndServe()
	if err != nil {
		slog.Error(
			"Ошибка при запуске веб-сервера",
			slog.String("Message", err.Error()),
		)
		os.Exit(1)
	}
}
