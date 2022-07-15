package main

import (
	"embed"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	"github.com/gorilla/mux"
	"gopkg.in/olahol/melody.v1"

	"github.com/lirix360/ReadmangaGrabber/config"
	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/db"
	"github.com/lirix360/ReadmangaGrabber/favs"
	"github.com/lirix360/ReadmangaGrabber/history"
	"github.com/lirix360/ReadmangaGrabber/logger"
	"github.com/lirix360/ReadmangaGrabber/manga"
	"github.com/lirix360/ReadmangaGrabber/tools"
)

//go:embed index.html
//go:embed assets/*
var webUI embed.FS

func main() {
	var err error

	logger.Log.Info("Запуск приложения!")

	r := mux.NewRouter()
	m := melody.New()

	r.HandleFunc("/getAppVer", tools.GetAppVer)

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
		logger.Log.Info("Закрытие приложения...")
		os.Exit(0)
	})

	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		err := m.HandleRequest(w, r)
		if err != nil {
			logger.Log.Error("Ошибка при обработке данных WS:", err)
		}
	})

	go func() {
		for {
			msgData := <-data.WSChan
			wsData, err := json.Marshal(msgData)
			if err != nil {
				logger.Log.Error("Ошибка при сериализации данных для отправки через WS:", err)
				continue
			}
			err = m.Broadcast(wsData)
			if err != nil {
				logger.Log.Error("Ошибка при отправке данных через WS:", err)
				continue
			}
		}
	}()

	r.PathPrefix("/").Handler(http.FileServer(http.FS(webUI)))

	a, err := astilectron.New(nil, astilectron.Options{
		AppName:           "Manga Graber",
		BaseDirectoryPath: ".",
	})
	if err != nil {
		logger.Log.Fatal("main: creating astilectron failed:", err)
	}
	defer a.Close()

	a.HandleSignals()

	srv := &http.Server{
		Handler:      r,
		Addr:         config.Cfg.Server.Addr + ":" + config.Cfg.Server.Port,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	// if config.Cfg.ShowGUI {
	// 	err = browser.OpenURL("http://" + config.Cfg.Server.Addr + ":" + config.Cfg.Server.Port + "/")
	// 	if err != nil {
	// 		logger.Log.Fatal("Ошибка при открытии браузера:", err)
	// 	}
	// }

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		<-signalChan
		// data.WSChan <- data.WSData{Cmd: "closeApp"}
		db.DBconn.Close()
		logger.Log.Info("Закрытие приложения...")
		os.Exit(0)
	}()

	go func() {
		err = srv.ListenAndServe()
		if err != nil {
			logger.Log.Fatal("Ошибка при запуске веб-сервера:", err)
		}
	}()

	if err = a.Start(); err != nil {
		logger.Log.Fatal("main: starting astilectron failed:", err)
	}

	var w *astilectron.Window
	if w, err = a.NewWindow("http://"+config.Cfg.Server.Addr+":"+config.Cfg.Server.Port+"/", &astilectron.WindowOptions{
		Center:    astikit.BoolPtr(true),
		Width:     astikit.IntPtr(1250),
		Height:    astikit.IntPtr(750),
		MinWidth:  astikit.IntPtr(1200),
		MinHeight: astikit.IntPtr(700),
		Icon:      astikit.StrPtr("./grabber3.ico"),
	}); err != nil {
		logger.Log.Fatal("main: new window failed:", err)
	}

	if err = w.Create(); err != nil {
		logger.Log.Fatal("main: creating window failed:", err)
	}

	// w.OpenDevTools()

	a.Wait()
}
