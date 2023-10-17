package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type PixabaySearchItem struct {
	Id              int     `json:"id"`
	Tags            string  `json:"tags"`
	WebFormatUrl    string  `json:"webformatURL"`
	WebFormatWidth  float32 `json:"webformatWidth"`
	WebFormatHeight float32 `json:"webformatHeight"`
	ImageUrl        string  `json:"imageURL"`
	UserId          int     `json:"user_id"`
	User            string  `json:"user"`
	PageUrl         string  `json:"pageURL"`
}

type PixabaySearchResult struct {
	Total     int                 `json:"total"`
	TotalHits int                 `json:"totalHits"`
	Hits      []PixabaySearchItem `json:"hits"`
}

type PixabayApi struct {
	Http   http.Client
	cache  *ReqCache
	apiKey string
	log    *log.Logger
}

func NewPixabayApi(cfg *Config, cache *ReqCache) PixabayApi {
	api := PixabayApi{
		cache:  cache,
		apiKey: cfg.Pixabay.Key,
		log:    log.New(os.Stderr, "(pixabay)", log.LstdFlags),
	}

	return api
}

func (api *PixabayApi) Type() string {
	return "pixabay"
}

func (api *PixabayApi) TTL() int {
	return 86400
}

func (api *PixabayApi) PageSize() int { return 100 }

func (api *PixabayApi) Search(page int, query string) ImageSearchResult {
	baseUrl := "https://pixabay.com/api/"
	qParam := url.Values{}
	qParam.Add("key", api.apiKey)
	qParam.Add("q", query)
	qParam.Add("page", strconv.Itoa(page))
	qParam.Add("per_page", strconv.Itoa(api.PageSize()))
	getReq, err := http.NewRequest(http.MethodGet, baseUrl+"?"+qParam.Encode(), nil)
	if err != nil {
		api.log.Println("Failed to create http request:", err.Error())
		return ImageSearchResult{err: &err, images: []ImageData{}}
	}
	req, err := api.cache.CachedFetch(getReq, &api.Http)
	if err != nil {
		api.log.Println("Failed to fetch:", err.Error())
		return ImageSearchResult{err: &err, images: []ImageData{}}
	}
	defer req.Body.Close()

	data := PixabaySearchResult{}
	err = json.NewDecoder(req.Body).Decode(&data)
	if err != nil {
		api.log.Println("Failed to decode response", err.Error())
		return ImageSearchResult{err: &err, images: []ImageData{}}
	}
	output := make([]ImageData, len(data.Hits))
	for i, el := range data.Hits {
		output[i].Id = "pixabay/" + strconv.Itoa(el.Id)
		output[i].Name = el.Tags
		output[i].Source = "Pixabay"
		output[i].SourceUrl = el.PageUrl
		output[i].Artist = el.User
		output[i].Aspect = el.WebFormatWidth / el.WebFormatHeight
		output[i].DownloadUrl = el.ImageUrl
		output[i].PreviewUrl = el.WebFormatUrl
	}
	return ImageSearchResult{err: nil, images: output}
}
