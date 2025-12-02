package scraper

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/multi-worker/internal/model"
)

// WeWorkRemotelyScraper scrapes jobs from WeWorkRemotely
type WeWorkRemotelyScraper struct {
	client *HTTPClient
}

func NewWeWorkRemotelyScraper(client *HTTPClient) *WeWorkRemotelyScraper {
	return &WeWorkRemotelyScraper{client: client}
}

func (s *WeWorkRemotelyScraper) Name() string {
	return "weworkremotely"
}

func (s *WeWorkRemotelyScraper) Category() string {
	return "jobs"
}

func (s *WeWorkRemotelyScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	// WeWorkRemotely has an RSS feed which is easier to parse
	url := "https://weworkremotely.com/categories/remote-programming-jobs.rss"
	if query != "" {
		// They have category-specific feeds
		categoryMap := map[string]string{
			"programming": "remote-programming-jobs",
			"design":      "remote-design-jobs",
			"devops":      "remote-devops-sysadmin-jobs",
			"management":  "remote-management-executive-jobs",
			"marketing":   "remote-marketing-jobs",
			"sales":       "remote-customer-support-jobs",
		}
		if cat, ok := categoryMap[strings.ToLower(query)]; ok {
			url = fmt.Sprintf("https://weworkremotely.com/categories/%s.rss", cat)
		}
	}

	data, err := s.client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch WeWorkRemotely: %w", err)
	}

	// Parse RSS
	items := parseRSSItems(string(data), limit)

	var result []model.ScrapedItem
	for _, item := range items {
		// Filter by query if provided
		if query != "" {
			lowerQuery := strings.ToLower(query)
			if !strings.Contains(strings.ToLower(item.Title), lowerQuery) &&
				!strings.Contains(strings.ToLower(item.Description), lowerQuery) {
				continue
			}
		}

		result = append(result, model.ScrapedItem{
			ID:          item.GUID,
			Title:       item.Title,
			Description: truncateText(stripHTML(item.Description), 500),
			URL:         item.Link,
			Source:      "WeWorkRemotely",
			Category:    "jobs",
			PostedAt:    item.PubDate,
		})
	}

	return result, nil
}

// GitHubJobsScraper - GitHub Jobs was deprecated, using placeholder
type GitHubJobsScraper struct {
	client *HTTPClient
}

func NewGitHubJobsScraper(client *HTTPClient) *GitHubJobsScraper {
	return &GitHubJobsScraper{client: client}
}

func (s *GitHubJobsScraper) Name() string {
	return "github_jobs"
}

func (s *GitHubJobsScraper) Category() string {
	return "jobs"
}

func (s *GitHubJobsScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	// GitHub Jobs was discontinued, redirecting to alternative
	return []model.ScrapedItem{
		{
			ID:          "github-redirect",
			Title:       "GitHub Jobs has been discontinued",
			Description: "Consider using other job sources like RemoteOK or WeWorkRemotely",
			URL:         "https://github.com/about/careers",
			Source:      "GitHub",
			Category:    "jobs",
		},
	}, nil
}

// StackOverflowJobsScraper - SO Jobs was also deprecated
type StackOverflowJobsScraper struct {
	client *HTTPClient
}

func NewStackOverflowJobsScraper(client *HTTPClient) *StackOverflowJobsScraper {
	return &StackOverflowJobsScraper{client: client}
}

func (s *StackOverflowJobsScraper) Name() string {
	return "stackoverflow_jobs"
}

func (s *StackOverflowJobsScraper) Category() string {
	return "jobs"
}

func (s *StackOverflowJobsScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	return []model.ScrapedItem{
		{
			ID:          "so-redirect",
			Title:       "Stack Overflow Jobs has been discontinued",
			Description: "Consider using other job sources",
			URL:         "https://stackoverflow.com/jobs",
			Source:      "StackOverflow",
			Category:    "jobs",
		},
	}, nil
}

type rssItem struct {
	Title       string
	Link        string
	Description string
	PubDate     string
	GUID        string
}

func parseRSSItems(rss string, limit int) []rssItem {
	var items []rssItem

	// Simple regex-based RSS parsing
	itemRe := regexp.MustCompile(`(?s)<item>(.*?)</item>`)
	titleRe := regexp.MustCompile(`(?s)<title>(?:<!\[CDATA\[)?(.*?)(?:\]\]>)?</title>`)
	linkRe := regexp.MustCompile(`(?s)<link>(?:<!\[CDATA\[)?(.*?)(?:\]\]>)?</link>`)
	descRe := regexp.MustCompile(`(?s)<description>(?:<!\[CDATA\[)?(.*?)(?:\]\]>)?</description>`)
	pubDateRe := regexp.MustCompile(`(?s)<pubDate>(.*?)</pubDate>`)
	guidRe := regexp.MustCompile(`(?s)<guid[^>]*>(.*?)</guid>`)

	itemMatches := itemRe.FindAllStringSubmatch(rss, -1)

	for i, match := range itemMatches {
		if limit > 0 && i >= limit {
			break
		}

		item := rssItem{}
		if m := titleRe.FindStringSubmatch(match[1]); len(m) > 1 {
			item.Title = strings.TrimSpace(m[1])
		}
		if m := linkRe.FindStringSubmatch(match[1]); len(m) > 1 {
			item.Link = strings.TrimSpace(m[1])
		}
		if m := descRe.FindStringSubmatch(match[1]); len(m) > 1 {
			item.Description = strings.TrimSpace(m[1])
		}
		if m := pubDateRe.FindStringSubmatch(match[1]); len(m) > 1 {
			item.PubDate = strings.TrimSpace(m[1])
		}
		if m := guidRe.FindStringSubmatch(match[1]); len(m) > 1 {
			item.GUID = strings.TrimSpace(m[1])
		}

		items = append(items, item)
	}

	return items
}
