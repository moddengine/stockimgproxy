package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type UnsplashPhoto struct {
	Id          string             `json:"id"`
	Width       float32            `json:"width"`
	Height      float32            `json:"height"`
	Description string             `json:"description"`
	User        UnsplashUser       `json:"user"`
	Urls        UnsplashUrls       `json:"urls"`
	Links       UnsplashPhotoLinks `json:"links"`
}

type UnsplashUser struct {
	Id       string            `json:"id"`
	Username string            `json:"username"`
	Name     string            `json:"name"`
	Links    UnsplashUserLinks `json:"links"`
}

type UnsplashPhotoLinks struct {
	Self     string `json:"self"`
	Html     string `json:"html"`
	Download string `json:"download""`
}

type UnsplashUserLinks struct {
	Self   string `json:"self"`
	Html   string `json:"html"`
	Photos string `json:"photos"`
	Likes  string `json:"likes"`
}

type UnsplashUrls struct {
	Regular string `json:"regular"`
	Raw     string `json:"raw"`
}

type UnsplashSearchResult struct {
	Total      int             `json:"total"`
	TotalPages int             `json:"total_pages"`
	Results    []UnsplashPhoto `json:"results"`
}

type UnsplashApi struct {
	Http      http.Client
	cache     *ReqCache
	accessKey string
	baseUrl   string
	log       *log.Logger
}

func NewUnsplashApi(cfg *Config, cache *ReqCache) UnsplashApi {

	return UnsplashApi{
		cache:     cache,
		accessKey: cfg.Unsplash.AccessKey,
		baseUrl:   "https://api.unsplash.com/search/photos",
		log:       log.New(os.Stderr, "(unsplash)", log.LstdFlags),
	}
}

func (unsp *UnsplashApi) Type() string {
	return "unsplash"
}

func (unsp *UnsplashApi) TTL() int {
	return 86400
}
func (unsp *UnsplashApi) PageSize() int { return 30 }

func (unsp *UnsplashApi) Search(page int, query string) ImageSearchResult {
	qParam := url.Values{}
	qParam.Add("query", query)
	qParam.Add("page", strconv.Itoa(page))
	qParam.Add("per_page", strconv.Itoa(unsp.PageSize()))
	getReq, err := http.NewRequest(http.MethodGet, unsp.baseUrl+"?"+qParam.Encode(), nil)
	if err != nil {
		unsp.log.Println("Failed to create http request:", err.Error())
		return ImageSearchResult{err: &err, images: []ImageData{}}
	}
	getReq.Header.Set("Accept-Version", "v1")
	getReq.Header.Set("Authorization", "Client-ID "+unsp.accessKey)
	req, err := unsp.cache.CachedFetch(getReq, &unsp.Http)
	if err != nil {
		unsp.log.Println("(unsplash) Failed to fetch:", err.Error())
		return ImageSearchResult{err: &err, images: []ImageData{}}
	}
	defer req.Body.Close()

	data := UnsplashSearchResult{}
	err = json.NewDecoder(req.Body).Decode(&data)
	if err != nil {
		unsp.log.Println("Failed to decode response", err.Error())
		return ImageSearchResult{err: &err, images: []ImageData{}}
	}
	output := make([]ImageData, len(data.Results))
	for i, el := range data.Results {
		output[i].Id = "unsplash/" + el.Id
		output[i].Name = el.Description
		output[i].Source = "Unsplash"
		output[i].SourceUrl = el.Links.Html
		output[i].Artist = el.User.Name
		output[i].Aspect = el.Width / el.Height
		output[i].DownloadUrl = el.Urls.Raw
		output[i].PreviewUrl = el.Urls.Regular
	}
	return ImageSearchResult{err: nil, images: output}
}
