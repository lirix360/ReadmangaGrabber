package main

import (
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/olahol/melody.v1"

	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/manga"
	"github.com/lirix360/ReadmangaGrabber/tools"
)

//go:embed index.html
var webUI embed.FS

func main() {
	var err error

	r := mux.NewRouter()
	m := melody.New()

	r.HandleFunc("/getChaptersList", manga.GetChaptersList)
	r.HandleFunc("/downloadManga", manga.DownloadManga)

	r.HandleFunc("/closeApp", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Закрытие приложения...")
		os.Exit(0)
	})

	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		err := m.HandleRequest(w, r)
		if err != nil {
			log.Println(err)
		}
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		log.Println(string(msg))
		m.Broadcast(msg)
	})

	go func() {
		for {
			msgData := <-data.WSChan
			wsData, err := json.Marshal(msgData)
			if err != nil {
				log.Println("Ошибка при сериализации данных для отправки через WS:", err)
				continue
			}
			err = m.Broadcast(wsData)
			if err != nil {
				log.Println("Ошибка при отправке данных через WS:", err)
				continue
			}
		}
	}()

	r.PathPrefix("/").Handler(http.FileServer(http.FS(webUI)))

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:8888",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	err = tools.OpenBrowser("http://127.0.0.1:8888/")
	if err != nil {
		log.Fatalln(err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		<-signalChan
		data.WSChan <- data.WSData{Cmd: "closeApp"}
		log.Println("Закрытие приложения...")
		// time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	log.Fatal("Ошибка при запуске веб-сервера:", srv.ListenAndServe())
}
