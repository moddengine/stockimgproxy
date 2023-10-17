package main

type ImageData struct {
	Id          string  `json:"id"`
	Name        string  `json:"tags"`
	Source      string  `json:"source"`
	SourceUrl   string  `json:"sourceUrl"`
	Artist      string  `json:"artist"`
	Aspect      float32 `json:"aspect"`
	PreviewUrl  string  `json:"previewUrl"`
	DownloadUrl string  `json:"downloadUrl"`
}

type ImageSearcher interface {
	Search(page int, query string) ImageSearchResult
	Type() string
	TTL() int
	PageSize() int
}

type ImageSearchResult struct {
	err    *error
	images []ImageData
}
