package main

import "fmt"

const PageSize int = 25

type PageSrc struct {
	Page  int
	First int
	Last  int
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func GetResPages(srcPage int, srcPageSize int, resPageSize int) []PageSrc {
	var startOffset = (srcPage - 1) * srcPageSize
	var endOffset = startOffset + srcPageSize
	var firstPage = 1 + (startOffset / resPageSize)
	var first = (firstPage - 1) * resPageSize
	var last = first + resPageSize
	//println("Output Offset:", startOffset, endOffset)
	//println("First Res ", first, last)
	pages := []PageSrc{{
		Page:  firstPage,
		First: startOffset - first,
		Last:  min(resPageSize, endOffset-first),
	}}
	for last < endOffset {
		remain := endOffset - (last / resPageSize * resPageSize)
		last += resPageSize
		lastPage := last / resPageSize
		pages = append(pages, PageSrc{
			Page:  lastPage,
			First: 0,
			Last:  min(resPageSize, remain),
		})
	}
	return pages
}

func (p *PageSrc) String() string {
	return fmt.Sprintf("#%d [%d:%d]", p.Page, p.First, p.Last)
}
