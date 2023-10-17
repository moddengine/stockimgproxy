package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPageOffset(t *testing.T) {
	list := GetResPages(1, 25, 30)
	assert.Equal(t, 1, len(list), "Number of pages")
	assert.Equal(t, 1, list[0].Page, "First Page = 1")
	assert.Equal(t, 0, list[0].First)
	assert.Equal(t, 25, list[0].Last, "First Page = 1")

	list = GetResPages(2, 25, 30)
	assert.Equal(t, 2, len(list), "Should be two pages returned")
	assert.Equal(t, 1, list[0].Page, "First Page = 1")
	assert.Equal(t, 25, list[0].First)
	assert.Equal(t, 30, list[0].Last)
	assert.Equal(t, 2, list[1].Page, "Second Page = 2")
	assert.Equal(t, 0, list[1].First)
	assert.Equal(t, 20, list[1].Last)

}

func TestPageOffset2(t *testing.T) {
	list := GetResPages(11, 25, 80)
	assert.Equal(t, 1, len(list))
	assert.Equal(t, 4, list[0].Page)
	assert.Equal(t, 10, list[0].First)
	assert.Equal(t, 35, list[0].Last)
	//   0    ------     80    ---------     160
	//   0 - 25 - 50 - 75 - 100
	list = GetResPages(4, 25, 80)
	assert.Equal(t, 2, len(list))
	assert.Equal(t, 1, list[0].Page)
	assert.Equal(t, 2, list[1].Page)
}

func TestPageOffset3(t *testing.T) {
	list := GetResPages(1, 100, 30)
	assert.Equal(t, 4, len(list))
	assert.Equal(t, 1, list[0].Page)
	assert.Equal(t, 0, list[0].First)
	assert.Equal(t, 30, list[0].Last)
	assert.Equal(t, 2, list[1].Page)
	assert.Equal(t, 0, list[1].First)
	assert.Equal(t, 30, list[1].Last)
	assert.Equal(t, 3, list[2].Page)
	assert.Equal(t, 0, list[2].First)
	assert.Equal(t, 30, list[2].Last)
	assert.Equal(t, 4, list[3].Page)
	assert.Equal(t, 0, list[3].First)
	assert.Equal(t, 10, list[3].Last)

	//   0    ------     80    ---------     160
	//   0 - 25 - 50 - 75 - 100
	list = GetResPages(4, 100, 80)
	assert.Equal(t, 2, len(list))
	assert.Equal(t, 4, list[0].Page)
	assert.Equal(t, 60, list[0].First)
	assert.Equal(t, 80, list[0].Last)
	assert.Equal(t, 5, list[1].Page)
	assert.Equal(t, 0, list[1].First)
	assert.Equal(t, 80, list[1].Last)
}
