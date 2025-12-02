package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/multi-worker/internal/model"
)

// FreelancerScraper scrapes projects from Freelancer.com
type FreelancerScraper struct {
	client *HTTPClient
}

func NewFreelancerScraper(client *HTTPClient) *FreelancerScraper {
	return &FreelancerScraper{client: client}
}

func (s *FreelancerScraper) Name() string {
	return "freelancer"
}

func (s *FreelancerScraper) Category() string {
	return "freelance"
}

type freelancerResponse struct {
	Status string `json:"status"`
	Result struct {
		Projects []freelancerProject `json:"projects"`
	} `json:"result"`
}

type freelancerProject struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"seo_url"` // Description often not in public API
	Type          string `json:"type"`
	Status        string `json:"status"`
	SubmitDate    int64  `json:"submitdate"`
	Currency      struct {
		Code string `json:"code"`
	} `json:"currency"`
	Budget struct {
		Minimum float64 `json:"minimum"`
		Maximum float64 `json:"maximum"`
	} `json:"budget"`
	Jobs []struct {
		Name string `json:"name"`
	} `json:"jobs"`
}

func (s *FreelancerScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	// Note: Full Freelancer API requires OAuth
	// This uses the public projects endpoint with limited data
	baseURL := "https://www.freelancer.com/api/projects/0.1/projects/active"

	params := url.Values{}
	params.Add("limit", fmt.Sprintf("%d", limit))
	params.Add("compact", "true")
	if query != "" {
		params.Add("query", query)
	}

	fullURL := baseURL + "?" + params.Encode()

	data, err := s.client.GetJSON(ctx, fullURL)
	if err != nil {
		// API might be restricted, return placeholder
		return s.fallbackResponse(query)
	}

	var response freelancerResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return s.fallbackResponse(query)
	}

	var items []model.ScrapedItem
	for _, project := range response.Result.Projects {
		var tags []string
		for _, job := range project.Jobs {
			tags = append(tags, job.Name)
		}

		budget := ""
		if project.Budget.Maximum > 0 {
			budget = fmt.Sprintf("%s %.0f - %.0f",
				project.Currency.Code,
				project.Budget.Minimum,
				project.Budget.Maximum)
		}

		items = append(items, model.ScrapedItem{
			ID:          fmt.Sprintf("%d", project.ID),
			Title:       project.Title,
			Description: fmt.Sprintf("Type: %s", project.Type),
			URL:         fmt.Sprintf("https://www.freelancer.com/projects/%s/%d", project.Description, project.ID),
			Source:      "Freelancer",
			Category:    "freelance",
			Tags:        tags,
			Salary:      budget,
		})
	}

	if len(items) == 0 {
		return s.fallbackResponse(query)
	}

	return items, nil
}

func (s *FreelancerScraper) fallbackResponse(query string) ([]model.ScrapedItem, error) {
	return []model.ScrapedItem{
		{
			ID:          "freelancer-info",
			Title:       fmt.Sprintf("Search Freelancer.com for: %s", query),
			Description: "Full Freelancer API access requires OAuth authentication. Visit the site directly.",
			URL:         fmt.Sprintf("https://www.freelancer.com/jobs/%s/", strings.ReplaceAll(query, " ", "-")),
			Source:      "Freelancer",
			Category:    "freelance",
		},
	}, nil
}

// UpworkScraper scrapes jobs from Upwork
type UpworkScraper struct {
	client *HTTPClient
}

func NewUpworkScraper(client *HTTPClient) *UpworkScraper {
	return &UpworkScraper{client: client}
}

func (s *UpworkScraper) Name() string {
	return "upwork"
}

func (s *UpworkScraper) Category() string {
	return "freelance"
}

func (s *UpworkScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	// Upwork has RSS feeds for job searches
	searchQuery := strings.ReplaceAll(query, " ", "%20")
	rssURL := fmt.Sprintf("https://www.upwork.com/ab/feed/jobs/rss?q=%s&sort=recency&paging=0%%3B%d", searchQuery, limit)

	data, err := s.client.Get(ctx, rssURL)
	if err != nil {
		return s.fallbackResponse(query)
	}

	rssItems := parseRSSItems(string(data), limit)
	if len(rssItems) == 0 {
		return s.fallbackResponse(query)
	}

	var items []model.ScrapedItem
	for _, item := range rssItems {
		items = append(items, model.ScrapedItem{
			ID:          item.GUID,
			Title:       item.Title,
			Description: truncateText(stripHTML(item.Description), 500),
			URL:         item.Link,
			Source:      "Upwork",
			Category:    "freelance",
			PostedAt:    item.PubDate,
		})
	}

	return items, nil
}

func (s *UpworkScraper) fallbackResponse(query string) ([]model.ScrapedItem, error) {
	return []model.ScrapedItem{
		{
			ID:          "upwork-info",
			Title:       fmt.Sprintf("Search Upwork for: %s", query),
			Description: "Visit Upwork directly to search for freelance opportunities.",
			URL:         fmt.Sprintf("https://www.upwork.com/nx/jobs/search/?q=%s&sort=recency", strings.ReplaceAll(query, " ", "%20")),
			Source:      "Upwork",
			Category:    "freelance",
		},
	}, nil
}
