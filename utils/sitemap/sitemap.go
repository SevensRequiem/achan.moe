package sitemap

import (
	"encoding/xml"
	"os"
)

// Sitemap represents a sitemap with a list of URLs.
type Sitemap struct {
	XMLName xml.Name `xml:"urlset"`
	XMLNS   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

// URL represents a URL in a sitemap.
type URL struct {
	Loc        string `xml:"loc"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

// AddURL adds a new URL to the sitemap.
func (s *Sitemap) AddURL(loc, changeFreq, priority string) {
	sitemapfile := "static/sitemap.xml"

	// Attempt to open the existing sitemap file
	file, err := os.Open(sitemapfile)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err) // File exists but cannot be opened
		}
	} else {
		// File exists, decode its content
		defer file.Close()
		if err := xml.NewDecoder(file).Decode(s); err != nil {
			panic(err) // Error decoding file
		}
	}

	// Append to the slice of URLs
	s.URLs = append(s.URLs, URL{Loc: loc, ChangeFreq: changeFreq, Priority: priority})

	// Open file with os.Create to overwrite or create a new file
	file, err = os.Create(sitemapfile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Encode the updated sitemap to the file
	enc := xml.NewEncoder(file)
	enc.Indent("", "  ")
	if err := enc.Encode(s); err != nil {
		panic(err)
	}
}
