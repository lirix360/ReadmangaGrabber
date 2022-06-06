package data

type MangaInfo struct {
	TitleOrig string `json:"title_orig"`
	TitleRu   string `json:"title_ru"`
}

type ChaptersList struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}

type RMTranslators struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WSData struct {
	Cmd     string      `json:"cmd"`
	Payload interface{} `json:"payload"`
}

type DownloadOpts struct {
	Type      string
	Chapters  string
	MangaURL  string
	PDF       string
	CBZ       string
	Del       string
	SavePath  string
	PrefTrans string
}

var WSChan = make(chan WSData, 10)

type PDF struct {
	DPI       float64
	MmInInch  float64
	A4Height  float64
	A4Width   float64
	MaxHeight float64
	MaxWidth  float64
}

var PDFOpts PDF

func init() {
	PDFOpts = PDF{
		DPI:       96,
		MmInInch:  25.4,
		A4Height:  297,
		A4Width:   210,
		MaxHeight: 1122,
		MaxWidth:  793,
	}
}
