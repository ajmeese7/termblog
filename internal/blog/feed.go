package blog

import (
	"fmt"
	"time"

	"github.com/gorilla/feeds"
)

// FeedGenerator generates RSS/Atom feeds
type FeedGenerator struct {
	title       string
	description string
	author      string
	baseURL     string
}

// NewFeedGenerator creates a new FeedGenerator
func NewFeedGenerator(title, description, author, baseURL string) *FeedGenerator {
	return &FeedGenerator{
		title:       title,
		description: description,
		author:      author,
		baseURL:     baseURL,
	}
}

// BaseURL returns the base URL for the feed
func (g *FeedGenerator) BaseURL() string {
	return g.baseURL
}

// GenerateRSS generates an RSS feed from posts
func (g *FeedGenerator) GenerateRSS(posts []*Post) (string, error) {
	feed := g.createFeed(posts)
	return feed.ToRss()
}

// GenerateAtom generates an Atom feed from posts
func (g *FeedGenerator) GenerateAtom(posts []*Post) (string, error) {
	feed := g.createFeed(posts)
	return feed.ToAtom()
}

// GenerateJSON generates a JSON feed from posts
func (g *FeedGenerator) GenerateJSON(posts []*Post) (string, error) {
	feed := g.createFeed(posts)
	return feed.ToJSON()
}

func (g *FeedGenerator) createFeed(posts []*Post) *feeds.Feed {
	now := time.Now()

	feed := &feeds.Feed{
		Title:       g.title,
		Link:        &feeds.Link{Href: g.baseURL},
		Description: g.description,
		Author:      &feeds.Author{Name: g.author},
		Created:     now,
		Updated:     now,
	}

	for _, post := range posts {
		if post.Draft {
			continue
		}

		published := now
		if post.PublishedAt != nil {
			published = *post.PublishedAt
		} else if !post.CreatedAt.IsZero() {
			published = post.CreatedAt
		}

		author := post.Author
		if author == "" {
			author = g.author
		}

		link := fmt.Sprintf("%s/posts/%s", g.baseURL, post.Slug)
		if post.CanonicalURL != "" {
			link = post.CanonicalURL
		}

		item := &feeds.Item{
			Title:       post.Title,
			Link:        &feeds.Link{Href: link},
			Description: post.Description,
			Author:      &feeds.Author{Name: author},
			Created:     published,
			Updated:     published,
		}

		// Add content if available (truncated for feed)
		if len(post.Content) > 500 {
			item.Content = post.Content[:500] + "..."
		} else {
			item.Content = post.Content
		}

		feed.Items = append(feed.Items, item)
	}

	// Update feed's Updated time to most recent post
	if len(feed.Items) > 0 {
		feed.Updated = feed.Items[0].Updated
	}

	return feed
}
