package config

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/lirix360/ReadmangaGrabber/logger"
)

type GrabberConfig struct {
	Savepath  string `json:"savepath"`
	FavTitle  string `json:"fav_title"`
	Readmanga struct {
		TimeoutImage   int `json:"timeout_image"`
		TimeoutChapter int `json:"timeout_chapter"`
	} `json:"readmanga"`
	Mangalib struct {
		TimeoutImage   int `json:"timeout_image"`
		TimeoutChapter int `json:"timeout_chapter"`
	} `json:"mangalib"`
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
}

func createConfig(filePath string) {
	newCfg := GrabberConfig{}

	newCfg.Savepath = "Manga/"
	newCfg.FavTitle = "ru"
	newCfg.Readmanga.TimeoutImage = 300
	newCfg.Readmanga.TimeoutChapter = 1000
	newCfg.Mangalib.TimeoutImage = 500
	newCfg.Mangalib.TimeoutChapter = 1000

	writeConfig("grabber_config.json", newCfg)
}

func readConfig(filePath string) error {
	credFile, err := ioutil.ReadFile(filePath)
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

	err = ioutil.WriteFile(filePath, configJSON, 0644)
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
	Cfg.Mangalib.TimeoutChapter = 1000
	Cfg.Mangalib.TimeoutImage = 500

	writeConfig("grabber_config.json", Cfg)
}
