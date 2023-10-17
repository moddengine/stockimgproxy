package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/andybalholm/brotli"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Pexels struct {
		Key string `json:"key"`
	} `json:"pexels.com"`
	Unsplash struct {
		AccessKey string `json:"access"`
		SecretKey string `json:"secret"`
	} `json:"unsplash.com"`
	Pixabay struct {
		Key string `json:"key"`
	} `json:"pixabay.com"`
	Debug struct {
		PrettyJson bool `json:"prettyJson"`
	}
}

func processError(err error) {
	fmt.Println(err.Error())
	os.Exit(2)
}

func loadConfig(cfg *Config) {

	f, err := os.Open("conf/config.json")
	if err != nil {
		processError(err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	switch err := decoder.Decode(&cfg).(type) {
	case *json.SyntaxError:
		f.Seek(0, io.SeekStart)
		pos := findPos(bufio.NewReader(f), int(err.Offset))
		log.Panicf("Unable to decode configuration file (Line: %d, Pos: %d); - %v\n", pos.line, pos.pos, err.Error())
	}
}

type FilePos struct {
	line int
	pos  int
}

func findPos(file *bufio.Reader, offset int) FilePos {
	p := FilePos{line: 1, pos: offset}
	var lineLen int
	for line, err := file.ReadBytes('\n'); len(line) > 0 && err == nil; line, err = file.ReadBytes('\n') {
		if p.pos < len(line) {
			return p
		}
		lineLen += len(line)
		if line[len(line)-1] == '\n' {
			p.line += 1
			p.pos -= lineLen
			lineLen = 0
		}
	}
	return p
}

func main() {
	var cfg Config
	loadConfig(&cfg)
	reqCache := NewReqCache(&cfg)

	var apis []ImageSearcher

	if cfg.Pixabay.Key != "" {
		apiPixabay := NewPixabayApi(&cfg, reqCache)
		apis = append(apis, &apiPixabay)
	}
	if cfg.Pexels.Key != "" {
		apiPexels := NewPexelsApi(&cfg, reqCache)
		apis = append(apis, &apiPexels)
	}
	if cfg.Unsplash.AccessKey != "" {
		apiUnsplash := NewUnsplashApi(&cfg, reqCache)
		apis = append(apis, &apiUnsplash)
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Not Found")
	})

	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		q, hasQ := r.URL.Query()["q"]
		if !hasQ {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Query Search Parameter ?q= missing")
			return
		}
		search := strings.Join(q, " ")
		var page int = 1
		qPage, hasPage := r.URL.Query()["page"]
		if hasPage {
			n, err := strconv.ParseInt(qPage[0], 10, 0)
			if err != nil {
				page = 1
			} else {
				page = int(n)
			}
		}
		chRes := make(chan ApiResult)

		var reqCount int

		for num, api := range apis {
			start := 0
			for _, src := range GetResPages(page, PageSize, api.PageSize()) {
				api := api
				src := src
				num := num
				reqCount += 1
				s := start
				go func() {
					chRes <- ApiResult{
						Num:    num,
						Page:   src,
						Result: api.Search(src.Page, search),
						Start:  s,
					}
				}()
				start += src.Last - src.First
			}

		}

		results := make([]ImageData, len(apis)*PageSize)

		ok := 0

		for rq := 0; rq < reqCount; rq++ {
			res := <-chRes
			if res.Result.err == nil {
				ok = ok + 1
			}
			first := min(len(res.Result.images), res.Page.First)
			last := min(len(res.Result.images), res.Page.Last)
			items := res.Result.images[first:last]
			//println("Output", res.Start, res.Num, len(items), first, last)
			for idx, item := range items {
				results[(res.Start+idx)*len(apis)+res.Num] = item
			}
		}
		body := brotli.HTTPCompressor(w, r)
		defer body.Close()
		if ok == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			log.Println(body, "Error connecting to upstream services")
			return
		}
		enc := json.NewEncoder(body)
		indent := ""
		if cfg.Debug.PrettyJson {
			indent = "  "
		}
		enc.SetIndent("", indent)
		enc.Encode(results)

	})
	log.Println("Starting Server on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

type ApiResult struct {
	Num    int
	Page   PageSrc
	Result ImageSearchResult
	Start  int
}
