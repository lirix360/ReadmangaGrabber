package config

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/logger"
)

type GrabberConfig struct {
	Savepath string `json:"savepath"`
	FavTitle string `json:"fav_title"`
	ShowGUI  bool   `json:"show_gui"`
	Server   struct {
		Addr string `json:"addr"`
		Port string `json:"port"`
	} `json:"server"`
	Readmanga struct {
		TimeoutImage   int `json:"timeout_image"`
		TimeoutChapter int `json:"timeout_chapter"`
	} `json:"readmanga"`
	Mangalib struct {
		TimeoutImage   int `json:"timeout_image"`
		TimeoutChapter int `json:"timeout_chapter"`
	} `json:"mangalib"`
	Proxy struct {
		Type string `json:"type"`
		Addr string `json:"addr"`
		Port string `json:"port"`
		Use  struct {
			Mangalib  bool `json:"mangalib"`
			Readmanga bool `json:"readmanga"`
		} `json:"use"`
	} `json:"proxy"`
	CurrentURLs data.CurrentURLS
}

var Cfg GrabberConfig

var DBver = "20220605"
var APPver = ""

func init() {
	if _, err := os.Stat("grabber_config.json"); os.IsNotExist(err) {
		createConfig("grabber_config.json")
	}

	err := readConfig("grabber_config.json")
	if err != nil {
		logger.Log.Fatal("Ошибка при чтении файла конфигурации:", err)
	}

	UpdateCfg()

	if Cfg.Readmanga.TimeoutImage < 500 {
		Cfg.Readmanga.TimeoutImage = 500
	}

	if Cfg.Readmanga.TimeoutChapter < 1000 {
		Cfg.Readmanga.TimeoutChapter = 1000
	}

	if Cfg.Mangalib.TimeoutImage < 500 {
		Cfg.Mangalib.TimeoutImage = 500
	}

	if Cfg.Mangalib.TimeoutChapter < 1000 {
		Cfg.Mangalib.TimeoutChapter = 1000
	}

	Cfg.CurrentURLs = GetURLs()

	if len(Cfg.CurrentURLs.MangaLib) == 0 || len(Cfg.CurrentURLs.ReadManga) == 0 {
		logger.Log.Fatal("Ошибка при получении списков текущих URL:", err)
	}
}

func createConfig(filePath string) {
	newCfg := GrabberConfig{}

	newCfg.Savepath = "Manga/"
	newCfg.FavTitle = "ru"
	newCfg.ShowGUI = true
	newCfg.Server.Addr = "127.0.0.1"
	newCfg.Server.Port = "8888"
	newCfg.Readmanga.TimeoutImage = 500
	newCfg.Readmanga.TimeoutChapter = 1000
	newCfg.Mangalib.TimeoutImage = 500
	newCfg.Mangalib.TimeoutChapter = 1000

	writeConfig("grabber_config.json", newCfg)
}

func readConfig(filePath string) error {
	credFile, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(strings.NewReader(string(credFile)))
	if err = dec.Decode(&Cfg); err != nil && err != io.EOF {
		return err
	}

	return nil
}

func writeConfig(filePath string, config GrabberConfig) error {
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, configJSON, 0644)
	if err != nil {
		return err
	}

	return nil
}

func LoadConfig(w http.ResponseWriter, r *http.Request) {
	cfgJSON, _ := json.Marshal(Cfg)

	w.Header().Set("Content-Type", "application/json")
	w.Write(cfgJSON)
}

func SaveConfig(w http.ResponseWriter, r *http.Request) {
	Cfg.Savepath = r.FormValue("savepath")
	Cfg.FavTitle = r.FormValue("fav_title")
	Cfg.Readmanga.TimeoutChapter, _ = strconv.Atoi(r.FormValue("readmanga_timeout_chapter"))
	Cfg.Readmanga.TimeoutImage, _ = strconv.Atoi(r.FormValue("readmanga_timeout_image"))
	Cfg.Mangalib.TimeoutChapter, _ = strconv.Atoi(r.FormValue("mangalib_timeout_chapter"))
	Cfg.Mangalib.TimeoutImage, _ = strconv.Atoi(r.FormValue("mangalib_timeout_image"))

	Cfg.Proxy.Type = r.FormValue("proxy_type")
	Cfg.Proxy.Addr = r.FormValue("proxy_addr")
	Cfg.Proxy.Port = r.FormValue("proxy_port")

	Cfg.Proxy.Use.Readmanga = false
	Cfg.Proxy.Use.Mangalib = false

	if r.FormValue("proxy_use_rm") == "1" {
		Cfg.Proxy.Use.Readmanga = true
	}

	if r.FormValue("proxy_use_ml") == "1" {
		Cfg.Proxy.Use.Mangalib = true
	}

	writeConfig("grabber_config.json", Cfg)
}

func UpdateCfg() {
	if Cfg.FavTitle == "" || Cfg.Server.Addr == "" || Cfg.Server.Port == "" {
		Cfg.FavTitle = "ru"
		Cfg.ShowGUI = true
		Cfg.Server.Addr = "127.0.0.1"
		Cfg.Server.Port = "8888"

		writeConfig("grabber_config.json", Cfg)

		readConfig("grabber_config.json")
	}
}

func GetURLs() data.CurrentURLS {
	tmpData := map[string]string{}
	curURLs := data.CurrentURLS{}

	resp, err := http.Get("https://raw.githubusercontent.com/lirix360/ReadmangaGrabber/master/lib_urls.json")
	if err != nil {
		logger.Log.Error(err)
		return curURLs
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error(err)
		return curURLs
	}

	json.Unmarshal(body, &tmpData)

	curURLs.MangaLib = strings.Split(tmpData["mangalib"], ", ")
	curURLs.ReadManga = strings.Split(tmpData["readmanga"], ", ")

	return curURLs
}
