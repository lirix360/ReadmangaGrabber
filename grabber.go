package main

import (
    "fmt"
    "flag"
    "os"
    "regexp"
    "time"
    "io"
    "io/ioutil"
    "net/url"
    "net/http"
    "strings"
    "github.com/PuerkitoBio/goquery"
    "github.com/mgutz/ansi"
)

var mangaChapters []string

func main() {
    fmt.Println()
    
    flag.Usage = func() {
        fmt.Println(ansi.Red, "Использование: " + os.Args[0] + " -url адрес_манги [список глав для скачивания]\n", ansi.Reset)
    }
    
    urlPtr := flag.String("url", "", "Адрес страницы описания манги или отдельной главы главы")
    
    flag.Parse()
    
    if *urlPtr == "" {
        fmt.Println(ansi.Red, "Не указан адрес манги!\n", ansi.Reset)
        os.Exit(0)
    }
    
    urlParts, _ := url.Parse(*urlPtr)
    
    if urlParts.Host != "readmanga.me" && urlParts.Host != "mintmanga.com" && urlParts.Host != "selfmanga.ru" {
        fmt.Println(ansi.Red, "Указан некорректный адрес манги! Скачивание доступно только с сайтов readmanga.me, mintmanga.com и selfmanga.ru.\n", ansi.Reset)
        os.Exit(0)
    }
    
    pathParts := strings.Split(strings.Trim(urlParts.Path, "/"), "/")
    
    if len(pathParts) == 1 {
        if len(flag.Args()) > 0 {
            mangaChapters = flag.Args()
        } else {
            getChapters(*urlPtr)
        }
    } else if len(pathParts) == 3 {
        mangaChapters = append(mangaChapters, pathParts[1] + "/" + pathParts[2])
    } else {
        fmt.Println(ansi.Red, "Указан некорректный адрес манги!\n", ansi.Reset)
        os.Exit(0)
    }
    
    if len(mangaChapters) > 0 {
        fmt.Println(ansi.Green, "Начинаю скачивание.", ansi.Reset)
        
        downloadChapters(urlParts.Host, pathParts[0])
        
        fmt.Println(ansi.Green, "Скачивание завершено.", ansi.Reset)
    } else {
        fmt.Println(ansi.Red, "Не найдено глав для скачивания!\n", ansi.Reset)
        os.Exit(0)
    }
    
    fmt.Println()
}

func getChapters(mangaUrl string) {
    mangaPage, _ := goquery.NewDocument(mangaUrl)
    
    mangaPage.Find(".chapters-link a").Each(func(i int, s *goquery.Selection) {
        link, _ := s.Attr("href")
        
        linkPaths := strings.Split(strings.Trim(link, "/"), "/")
        
        mangaChapters = append(mangaChapters, linkPaths[1] + "/" + linkPaths[2])
    })
    
    for left, right := 0, len(mangaChapters)-1; left < right; left, right = left+1, right-1 {
        mangaChapters[left], mangaChapters[right] = mangaChapters[right], mangaChapters[left]
    }
}

func downloadChapters(mangaHost string, mangaName string) {
    url := "http://" + mangaHost + "/" + mangaName + "/"
    
    for i := 0; i < len(mangaChapters); i++ {
        imageLinks := getImageLinks(url + mangaChapters[i])
        
        if len(imageLinks) > 0 {
            fmt.Println(ansi.Green, "Скачиваю главу " + mangaChapters[i] + ".", ansi.Reset)
            
            for x := 0; x < len(imageLinks); x++ {
                downloadFile(imageLinks[x], mangaName, mangaChapters[i])
                
                time.Sleep(200 * time.Millisecond)
            }
        } else {
            fmt.Println(ansi.Red, "В главе " + mangaChapters[i] + " не найдено страниц для скачивания!", ansi.Reset)
        }
    }
}

func getImageLinks(chapterUrl string) []string {
    resp, _ := http.Get(chapterUrl)
    
    pageBody, _ := ioutil.ReadAll(resp.Body)
    
    resp.Body.Close()
    
    var imageLinks []string
    
    r := regexp.MustCompile(`rm_h\.init(.+);`)
    r2 := regexp.MustCompile(`\[.+\]`)
    
    imagePartsString := strings.Trim(r2.FindString(r.FindString(string(pageBody))), "[]")
    
    if imagePartsString != "" {
        imageParts := strings.Split(imagePartsString, "],[")
        
        for i := 0; i < len(imageParts); i++ {
            tmpParts := strings.Split(imageParts[i], ",")
            
            imageLinks = append(imageLinks, strings.Trim(tmpParts[1], "\"'") + strings.Trim(tmpParts[0], "\"'") + strings.Trim(tmpParts[2], "\"'"))
        }
    }
    
    return imageLinks
}

func downloadFile(fileUrl string, mangaName string, mangaChapter string) (int64, error) {
    urlParts := strings.Split(fileUrl, "/")
    
    if _, err := os.Stat("Downloaded manga/" + mangaName + "/" + mangaChapter); os.IsNotExist(err) {
        os.MkdirAll("Downloaded manga/" + mangaName + "/" + mangaChapter, 0755)
    }
    
    fp, err := os.OpenFile("Downloaded manga/" + mangaName + "/" + mangaChapter + "/" + urlParts[len(urlParts)-1], os.O_RDWR|os.O_CREATE, os.ModePerm)
    if err != nil  {
        return 0, err
    }
    defer fp.Close()
    
    resp, err := http.Get(fileUrl)
    if err != nil  {
        return 0, err
    }
    defer resp.Body.Close()
    
    w, err := io.Copy(fp, resp.Body)
    if err != nil  {
        return 0, err
    }
    
    return w, nil
}