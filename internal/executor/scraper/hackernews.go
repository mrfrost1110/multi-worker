package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/multi-worker/internal/model"
)

// HackerNewsJobsScraper scrapes jobs from HN Who's Hiring threads
type HackerNewsJobsScraper struct {
	client *HTTPClient
}

func NewHackerNewsJobsScraper(client *HTTPClient) *HackerNewsJobsScraper {
	return &HackerNewsJobsScraper{client: client}
}

func (s *HackerNewsJobsScraper) Name() string {
	return "hackernews_jobs"
}

func (s *HackerNewsJobsScraper) Category() string {
	return "jobs"
}

type hnItem struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	By          string `json:"by"`
	Time        int64  `json:"time"`
	Text        string `json:"text"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Descendants int    `json:"descendants"`
	Kids        []int  `json:"kids"`
}

func (s *HackerNewsJobsScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	// Get job stories
	data, err := s.client.GetJSON(ctx, "https://hacker-news.firebaseio.com/v0/jobstories.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HN job stories: %w", err)
	}

	var jobIDs []int
	if err := json.Unmarshal(data, &jobIDs); err != nil {
		return nil, fmt.Errorf("failed to parse job stories: %w", err)
	}

	if limit > 0 && len(jobIDs) > limit {
		jobIDs = jobIDs[:limit]
	}

	var items []model.ScrapedItem
	for _, id := range jobIDs {
		itemData, err := s.client.GetJSON(ctx, fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id))
		if err != nil {
			continue
		}

		var item hnItem
		if err := json.Unmarshal(itemData, &item); err != nil {
			continue
		}

		// Filter by query if provided
		if query != "" {
			lowerQuery := strings.ToLower(query)
			if !strings.Contains(strings.ToLower(item.Title), lowerQuery) &&
				!strings.Contains(strings.ToLower(item.Text), lowerQuery) {
				continue
			}
		}

		company := extractCompanyFromTitle(item.Title)

		items = append(items, model.ScrapedItem{
			ID:          fmt.Sprintf("%d", item.ID),
			Title:       item.Title,
			Description: truncateText(stripHTML(item.Text), 500),
			URL:         item.URL,
			Source:      "HackerNews Jobs",
			Category:    "jobs",
			Company:     company,
		})
	}

	return items, nil
}

// HackerNewsScraper scrapes tech news from HN front page
type HackerNewsScraper struct {
	client *HTTPClient
}

func NewHackerNewsScraper(client *HTTPClient) *HackerNewsScraper {
	return &HackerNewsScraper{client: client}
}

func (s *HackerNewsScraper) Name() string {
	return "hackernews"
}

func (s *HackerNewsScraper) Category() string {
	return "news"
}

func (s *HackerNewsScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	data, err := s.client.GetJSON(ctx, "https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HN top stories: %w", err)
	}

	var storyIDs []int
	if err := json.Unmarshal(data, &storyIDs); err != nil {
		return nil, fmt.Errorf("failed to parse story IDs: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}
	if len(storyIDs) > limit*2 {
		storyIDs = storyIDs[:limit*2]
	}

	var items []model.ScrapedItem
	for _, id := range storyIDs {
		if len(items) >= limit {
			break
		}

		itemData, err := s.client.GetJSON(ctx, fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id))
		if err != nil {
			continue
		}

		var item hnItem
		if err := json.Unmarshal(itemData, &item); err != nil {
			continue
		}

		if query != "" && !strings.Contains(strings.ToLower(item.Title), strings.ToLower(query)) {
			continue
		}

		url := item.URL
		if url == "" {
			url = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", item.ID)
		}

		items = append(items, model.ScrapedItem{
			ID:          fmt.Sprintf("%d", item.ID),
			Title:       item.Title,
			Description: fmt.Sprintf("%d points, %d comments", item.Descendants, len(item.Kids)),
			URL:         url,
			Source:      "HackerNews",
			Category:    "news",
			Extra: map[string]interface{}{
				"points":   item.Descendants,
				"comments": len(item.Kids),
			},
		})
	}

	return items, nil
}

func extractCompanyFromTitle(title string) string {
	// Common pattern: "Company (YC XX) is hiring..."
	re := regexp.MustCompile(`^([^(]+)`)
	matches := re.FindStringSubmatch(title)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func stripHTML(html string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(html, "")
}
