package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andybalholm/brotli"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	_ "net/http/pprof"
	"net/url"
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
	Database string `json:"database"`
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

func initApi(cfg *Config, store *Store) []ImageSearcher {
	reqCache := NewReqCache(cfg, store)

	var apis []ImageSearcher

	if cfg.Pixabay.Key != "" {
		apiPixabay := NewPixabayApi(cfg, reqCache)
		apis = append(apis, &apiPixabay)
	}
	if cfg.Pexels.Key != "" {
		apiPexels := NewPexelsApi(cfg, reqCache)
		apis = append(apis, &apiPexels)
	}
	if cfg.Unsplash.AccessKey != "" {
		apiUnsplash := NewUnsplashApi(cfg, reqCache)
		apis = append(apis, &apiUnsplash)
	}
	return apis
}

type QueryParams struct {
	Page   int
	Search string
}

func parseURL(url *url.URL) (*QueryParams, error) {
	p := QueryParams{
		Page: 1,
	}
	q, hasQ := url.Query()["q"]
	if !hasQ {
		return nil, errors.New("query search parameter ?q= missing")
	}
	p.Search = strings.Trim(strings.Join(q, " "), " \t\n")
	if p.Search == "" {
		return nil, errors.New("query search cannot be empty")
	}
	qPage, hasPage := url.Query()["page"]
	if hasPage {
		n, err := strconv.ParseInt(qPage[0], 10, 0)
		if err != nil {
			p.Page = 1
		} else {
			p.Page = int(n)
		}
	}
	return &p, nil
}

func searchHandler(cfg *Config, apis []ImageSearcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := parseURL(r.URL)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Query Search Parameter ?q= missing")
			return
		}
		chRes := make(chan ApiResult)

		var reqCount int

		for num, api := range apis {
			start := 0
			for _, src := range GetResPages(query.Page, PageSize, api.PageSize()) {
				api := api
				src := src
				num := num
				reqCount += 1
				s := start
				go func() {
					chRes <- ApiResult{
						Num:    num,
						Page:   src,
						Result: api.Search(src.Page, query.Search),
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

	}
}

func httpAuth(next http.HandlerFunc, testUser func(user string, pass string) bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if ok {
			if testUser(user, pass) {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return

	}
}

func (cfg *Config) testAuth(user string, pass string) bool {
	return false
}

func main() {
	cfg := Config{}
	loadConfig(&cfg)

	store := NewStore(&cfg)

	apis := initApi(&cfg, store)

	defRoute := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Not Found")
	}
	search := httpAuth(searchHandler(&cfg, apis), store.TestUser)
	go func() {
		if _, err := os.Stat("sock/fcgi.sock"); os.IsNotExist(err) {
			os.Mkdir("sock", 0755)
		} else {
			os.Remove("sock/fcgi.sock")
		}
		fcgid := http.NewServeMux()
		fcgid.HandleFunc("/", defRoute)
		fcgid.HandleFunc("/search", search)

		sock, err := net.Listen("unix", "sock/fcgi.sock")
		if err != nil {
			log.Panicln("Unable to open socket", err.Error())
		}
		log.Println("Starting FastCGI Server on sock/fcgi.sock")
		err = fcgi.Serve(sock, fcgid)
		if err != nil {
			log.Panicln("Unable to bind to socket", err.Error())
		}
	}()

	httpServer := http.NewServeMux()
	httpServer.HandleFunc("/", defRoute)
	httpServer.HandleFunc("/search", search)

	log.Println("Starting HTTP Server on :8081")
	log.Fatal(http.ListenAndServe(":8081", httpServer))
}

type ApiResult struct {
	Num    int
	Page   PageSrc
	Result ImageSearchResult
	Start  int
}
