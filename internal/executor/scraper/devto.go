package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/multi-worker/internal/model"
)

// DevToScraper scrapes articles from Dev.to
type DevToScraper struct {
	client *HTTPClient
}

func NewDevToScraper(client *HTTPClient) *DevToScraper {
	return &DevToScraper{client: client}
}

func (s *DevToScraper) Name() string {
	return "devto"
}

func (s *DevToScraper) Category() string {
	return "news"
}

type devToArticle struct {
	ID              int    `json:"id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	URL             string `json:"url"`
	PublishedAt     string `json:"published_at"`
	TagList         []string `json:"tag_list"`
	User            struct {
		Name string `json:"name"`
	} `json:"user"`
	PositiveReactionsCount int `json:"positive_reactions_count"`
	CommentsCount          int `json:"comments_count"`
}

func (s *DevToScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	url := "https://dev.to/api/articles?per_page=50"
	if query != "" {
		url = fmt.Sprintf("https://dev.to/api/articles?tag=%s&per_page=50", strings.ReplaceAll(query, " ", ""))
	}

	data, err := s.client.GetJSON(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Dev.to: %w", err)
	}

	var articles []devToArticle
	if err := json.Unmarshal(data, &articles); err != nil {
		return nil, fmt.Errorf("failed to parse Dev.to response: %w", err)
	}

	if limit > 0 && len(articles) > limit {
		articles = articles[:limit]
	}

	var items []model.ScrapedItem
	for _, article := range articles {
		items = append(items, model.ScrapedItem{
			ID:          fmt.Sprintf("%d", article.ID),
			Title:       article.Title,
			Description: article.Description,
			URL:         article.URL,
			Source:      "Dev.to",
			Category:    "news",
			Tags:        article.TagList,
			PostedAt:    article.PublishedAt,
			Extra: map[string]interface{}{
				"author":    article.User.Name,
				"reactions": article.PositiveReactionsCount,
				"comments":  article.CommentsCount,
			},
		})
	}

	return items, nil
}

// ProductHuntScraper scrapes from Product Hunt
type ProductHuntScraper struct {
	client *HTTPClient
}

func NewProductHuntScraper(client *HTTPClient) *ProductHuntScraper {
	return &ProductHuntScraper{client: client}
}

func (s *ProductHuntScraper) Name() string {
	return "producthunt"
}

func (s *ProductHuntScraper) Category() string {
	return "news"
}

func (s *ProductHuntScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	// Product Hunt requires API key for proper access
	// This is a simplified implementation that might need adjustment
	return []model.ScrapedItem{
		{
			ID:          "ph-placeholder",
			Title:       "Product Hunt integration requires API key",
			Description: "Please configure PRODUCTHUNT_API_KEY for full access",
			URL:         "https://producthunt.com",
			Source:      "ProductHunt",
			Category:    "news",
		},
	}, nil
}
