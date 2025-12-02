package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/multi-worker/internal/model"
	"github.com/multi-worker/internal/storage"
)

// Executor handles RSS feed reading in pipelines
type Executor struct {
	client *http.Client
	cache  *storage.CacheRepository
}

// NewExecutor creates a new RSS executor
func NewExecutor(cache *storage.CacheRepository) *Executor {
	return &Executor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: cache,
	}
}

func (e *Executor) Type() string {
	return "rss"
}

func (e *Executor) Validate(config map[string]interface{}) error {
	if _, ok := config["url"]; !ok {
		if _, ok := config["urls"]; !ok {
			return fmt.Errorf("rss requires 'url' or 'urls' in config")
		}
	}
	return nil
}

func (e *Executor) Execute(ctx context.Context, input *model.ExecutorResult, config map[string]interface{}) (*model.ExecutorResult, error) {
	// Get URLs to fetch
	var urls []string
	if url, ok := config["url"].(string); ok {
		urls = append(urls, url)
	}
	if urlsArr, ok := config["urls"].([]interface{}); ok {
		for _, u := range urlsArr {
			if str, ok := u.(string); ok {
				urls = append(urls, str)
			}
		}
	}

	// Get limit
	limit := 20
	if l, ok := config["limit"].(float64); ok {
		limit = int(l)
	}

	// Get filter keywords
	var keywords []string
	if kws, ok := config["keywords"].([]interface{}); ok {
		for _, k := range kws {
			if s, ok := k.(string); ok {
				keywords = append(keywords, strings.ToLower(s))
			}
		}
	}

	// Get task ID for caching
	taskID, _ := config["task_id"].(string)

	// Fetch all RSS feeds
	var allItems []model.RSSItem
	var errors []string

	for _, url := range urls {
		items, err := e.fetchFeed(ctx, url, limit)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", url, err))
			continue
		}

		// Filter by keywords if provided
		if len(keywords) > 0 {
			items = filterByKeywords(items, keywords)
		}

		// Deduplicate using cache
		if e.cache != nil && taskID != "" {
			items = e.filterNewItems(ctx, items, taskID)
		}

		allItems = append(allItems, items...)
	}

	// Limit total results
	if len(allItems) > limit {
		allItems = allItems[:limit]
	}

	metadata := map[string]interface{}{
		"feeds":       urls,
		"total_items": len(allItems),
	}
	if len(errors) > 0 {
		metadata["errors"] = errors
	}

	return &model.ExecutorResult{
		Data:      allItems,
		Metadata:  metadata,
		ItemCount: len(allItems),
	}, nil
}

// RSS/Atom feed structures
type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	PubDate     string   `xml:"pubDate"`
	GUID        string   `xml:"guid"`
	Author      string   `xml:"author"`
	Categories  []string `xml:"category"`
}

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string     `xml:"title"`
	Link    atomLink   `xml:"link"`
	Summary string     `xml:"summary"`
	Content string     `xml:"content"`
	Updated string     `xml:"updated"`
	ID      string     `xml:"id"`
	Author  atomAuthor `xml:"author"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

func (e *Executor) fetchFeed(ctx context.Context, url string, limit int) ([]model.RSSItem, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MultiWorker/1.0)")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Try RSS first
	var rss rssFeed
	if err := xml.Unmarshal(body, &rss); err == nil && len(rss.Channel.Items) > 0 {
		return convertRSSItems(rss.Channel.Items, rss.Channel.Title, limit), nil
	}

	// Try Atom
	var atom atomFeed
	if err := xml.Unmarshal(body, &atom); err == nil && len(atom.Entries) > 0 {
		return convertAtomItems(atom.Entries, atom.Title, limit), nil
	}

	return nil, fmt.Errorf("could not parse feed as RSS or Atom")
}

func convertRSSItems(items []rssItem, source string, limit int) []model.RSSItem {
	var result []model.RSSItem

	for i, item := range items {
		if limit > 0 && i >= limit {
			break
		}

		id := item.GUID
		if id == "" {
			id = item.Link
		}

		result = append(result, model.RSSItem{
			ID:          id,
			Title:       item.Title,
			Description: stripHTMLTags(item.Description),
			Link:        item.Link,
			Source:      source,
			PubDate:     item.PubDate,
			Categories:  item.Categories,
			Author:      item.Author,
		})
	}

	return result
}

func convertAtomItems(entries []atomEntry, source string, limit int) []model.RSSItem {
	var result []model.RSSItem

	for i, entry := range entries {
		if limit > 0 && i >= limit {
			break
		}

		description := entry.Summary
		if description == "" {
			description = entry.Content
		}

		result = append(result, model.RSSItem{
			ID:          entry.ID,
			Title:       entry.Title,
			Description: stripHTMLTags(description),
			Link:        entry.Link.Href,
			Source:      source,
			PubDate:     entry.Updated,
			Author:      entry.Author.Name,
		})
	}

	return result
}

func filterByKeywords(items []model.RSSItem, keywords []string) []model.RSSItem {
	var filtered []model.RSSItem

	for _, item := range items {
		text := strings.ToLower(item.Title + " " + item.Description)
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				filtered = append(filtered, item)
				break
			}
		}
	}

	return filtered
}

func (e *Executor) filterNewItems(ctx context.Context, items []model.RSSItem, taskID string) []model.RSSItem {
	var newItems []model.RSSItem
	var newHashes []string

	for _, item := range items {
		content := item.Link
		if content == "" {
			content = item.ID
		}
		hash := e.cache.HashContent(content)

		exists, err := e.cache.ExistsForTask(ctx, hash, taskID)
		if err != nil || exists {
			continue
		}

		newItems = append(newItems, item)
		newHashes = append(newHashes, hash)
	}

	if len(newHashes) > 0 {
		e.cache.AddBatch(ctx, newHashes, "rss", taskID)
	}

	return newItems
}

func stripHTMLTags(s string) string {
	// Simple HTML tag removal
	var result strings.Builder
	inTag := false

	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}

	return strings.TrimSpace(result.String())
}

// Common RSS feed URLs for tech news
var CommonFeeds = map[string]string{
	"hackernews":    "https://hnrss.org/frontpage",
	"techcrunch":    "https://techcrunch.com/feed/",
	"theverge":      "https://www.theverge.com/rss/index.xml",
	"arstechnica":   "https://feeds.arstechnica.com/arstechnica/index",
	"wired":         "https://www.wired.com/feed/rss",
	"devto":         "https://dev.to/feed",
	"lobsters":      "https://lobste.rs/rss",
	"reddit_prog":   "https://www.reddit.com/r/programming/.rss",
	"reddit_golang": "https://www.reddit.com/r/golang/.rss",
	"reddit_rust":   "https://www.reddit.com/r/rust/.rss",
	"reddit_python": "https://www.reddit.com/r/python/.rss",
	"golang_blog":   "https://go.dev/blog/feed.atom",
	"rust_blog":     "https://blog.rust-lang.org/feed.xml",
}
