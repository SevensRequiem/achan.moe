package search

import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

// SearchResult represents a single search result
type SearchResult struct {
	FileName string
	Content  interface{}
}

// searchFiles searches for the searchTerm in all .gob files in the specified boardID directory
func searchFiles(boardID, searchTerm string) ([]SearchResult, error) {
	var results []SearchResult
	dir := fmt.Sprintf("boards/%s/", boardID)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".gob" {
			filePath := filepath.Join(dir, file.Name())
			f, err := os.Open(filePath)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			var data interface{}
			decoder := gob.NewDecoder(f)
			if err := decoder.Decode(&data); err != nil {
				return nil, err
			}

			// Convert data to string and search for the term
			dataStr := fmt.Sprintf("%v", data)
			if strings.Contains(dataStr, searchTerm) {
				results = append(results, SearchResult{FileName: file.Name(), Content: data})
			}
		}
	}

	return results, nil
}

// SearchHandler handles the search request
func SearchHandler(c echo.Context) error {
	boardID := c.Param("boardid")
	searchTerm := c.QueryParam("q")

	results, err := searchFiles(boardID, searchTerm)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, results)
}
