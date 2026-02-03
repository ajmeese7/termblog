package blog

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// URLSet represents the root element of a sitemap
type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	XMLNS   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

// URL represents a single URL entry in the sitemap
type URL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

// SitemapGenerator generates XML sitemaps
type SitemapGenerator struct {
	baseURL string
}

// NewSitemapGenerator creates a new sitemap generator
func NewSitemapGenerator(baseURL string) *SitemapGenerator {
	return &SitemapGenerator{baseURL: baseURL}
}

// Generate creates a sitemap XML string from posts
func (g *SitemapGenerator) Generate(posts []*Post) (string, error) {
	urlset := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	// Add homepage
	urlset.URLs = append(urlset.URLs, URL{
		Loc:        g.baseURL,
		ChangeFreq: "daily",
		Priority:   "1.0",
	})

	// Add each post
	for _, post := range posts {
		u := URL{
			Loc:        fmt.Sprintf("%s/posts/%s", g.baseURL, url.PathEscape(post.Slug)),
			ChangeFreq: "weekly",
			Priority:   "0.8",
		}

		// Add last modified date if available
		if post.PublishedAt != nil {
			u.LastMod = post.PublishedAt.Format(time.RFC3339)
		} else if !post.CreatedAt.IsZero() {
			u.LastMod = post.CreatedAt.Format(time.RFC3339)
		}

		urlset.URLs = append(urlset.URLs, u)
	}

	// Collect unique tags (normalized to lowercase) and add tag pages
	tagSet := make(map[string]struct{})
	for _, post := range posts {
		for _, tag := range post.Tags {
			tagSet[strings.ToLower(tag)] = struct{}{}
		}
	}
	for tag := range tagSet {
		urlset.URLs = append(urlset.URLs, URL{
			Loc:        fmt.Sprintf("%s/tags/%s", g.baseURL, url.PathEscape(tag)),
			ChangeFreq: "weekly",
			Priority:   "0.6",
		})
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(urlset, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal sitemap: %w", err)
	}

	return xml.Header + string(output), nil
}
