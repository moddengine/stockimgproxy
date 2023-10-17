package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type PexelsPhoto struct {
	Id             int            `json:"id"`
	Width          float32        `json:"width"`
	Height         float32        `json:"height"`
	Url            string         `json:"url"`
	Alt            string         `json:"alt"`
	Photographer   string         `json:"photographer"`
	PhotographerId int            `json:"photographer_id"`
	Src            PexelsPhotoSrc `json:"src"`
}

type PexelsPhotoSrc struct {
	Original string `json:"original"`
	Large    string `json:"large"`
}

type PexelsSearchResult struct {
	TotalResults int           `json:"total_results"`
	Page         int           `json:"page"`
	PerPage      int           `json:"per_page"`
	Photos       []PexelsPhoto `json:"photos"`
}

type PexelsApi struct {
	Http    http.Client
	cache   *ReqCache
	apiKey  string
	baseUrl string
	log     *log.Logger
}

func NewPexelsApi(cfg *Config, cache *ReqCache) PexelsApi {
	return PexelsApi{
		apiKey:  cfg.Pexels.Key,
		cache:   cache,
		baseUrl: "https://api.pexels.com/v1/search",
		log:     log.New(os.Stderr, "(pexels)", log.LstdFlags),
	}
}

func (api *PexelsApi) Type() string {
	return "pexels"
}

func (api *PexelsApi) TTL() int {
	return 86400
}

func (api *PexelsApi) PageSize() int { return 80 }

func (api *PexelsApi) Search(Page int, query string) ImageSearchResult {
	qParam := url.Values{}
	qParam.Add("key", api.apiKey)
	qParam.Add("query", query)
	qParam.Add("page", strconv.Itoa(Page))
	qParam.Add("per_page", strconv.Itoa(api.PageSize()))
	getReq, err := http.NewRequest(http.MethodGet, api.baseUrl+"?"+qParam.Encode(), nil)
	if err != nil {
		api.log.Println("Failed to create http request:", err)
		return ImageSearchResult{err: &err, images: []ImageData{}}
	}
	getReq.Header.Set("Authorization", api.apiKey)
	req, err := api.cache.CachedFetch(getReq, &api.Http)
	if err != nil {
		api.log.Println("Failed to fetch:", err)
		return ImageSearchResult{err: &err, images: []ImageData{}}
	}
	defer req.Body.Close()

	data := PexelsSearchResult{}
	err = json.NewDecoder(req.Body).Decode(&data)
	if err != nil {
		api.log.Println("Failed to decode response", err)
		return ImageSearchResult{err: &err, images: []ImageData{}}
	}
	output := make([]ImageData, len(data.Photos))
	for i, el := range data.Photos {
		output[i].Id = "pexels/" + strconv.Itoa(el.Id)
		output[i].Name = el.Alt
		output[i].Source = "Pexels"
		output[i].SourceUrl = el.Url
		output[i].Artist = el.Photographer
		output[i].Aspect = el.Width / el.Height
		output[i].DownloadUrl = el.Src.Original
		output[i].PreviewUrl = el.Src.Large
	}
	return ImageSearchResult{err: nil, images: output}
}
