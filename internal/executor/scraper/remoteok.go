package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/multi-worker/internal/model"
)

// RemoteOKScraper scrapes jobs from RemoteOK.com
type RemoteOKScraper struct {
	client *HTTPClient
}

func NewRemoteOKScraper(client *HTTPClient) *RemoteOKScraper {
	return &RemoteOKScraper{client: client}
}

func (s *RemoteOKScraper) Name() string {
	return "remoteok"
}

func (s *RemoteOKScraper) Category() string {
	return "jobs"
}

type remoteOKJob struct {
	Slug        string   `json:"slug"`
	ID          string   `json:"id"`
	Epoch       int64    `json:"epoch"`
	Date        string   `json:"date"`
	Company     string   `json:"company"`
	Position    string   `json:"position"`
	Tags        []string `json:"tags"`
	Logo        string   `json:"logo"`
	Description string   `json:"description"`
	Location    string   `json:"location"`
	SalaryMin   int      `json:"salary_min"`
	SalaryMax   int      `json:"salary_max"`
	URL         string   `json:"url"`
}

func (s *RemoteOKScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	url := "https://remoteok.com/api"
	if query != "" {
		url = fmt.Sprintf("https://remoteok.com/api?tag=%s", strings.ReplaceAll(query, " ", "+"))
	}

	data, err := s.client.GetJSON(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RemoteOK: %w", err)
	}

	var jobs []remoteOKJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse RemoteOK response: %w", err)
	}

	// First item is metadata, skip it
	if len(jobs) > 0 {
		jobs = jobs[1:]
	}

	if limit > 0 && len(jobs) > limit {
		jobs = jobs[:limit]
	}

	var items []model.ScrapedItem
	for _, job := range jobs {
		salary := ""
		if job.SalaryMin > 0 || job.SalaryMax > 0 {
			salary = fmt.Sprintf("$%d - $%d", job.SalaryMin, job.SalaryMax)
		}

		items = append(items, model.ScrapedItem{
			ID:          job.ID,
			Title:       job.Position,
			Description: truncateText(job.Description, 500),
			URL:         fmt.Sprintf("https://remoteok.com/remote-jobs/%s", job.Slug),
			Source:      "RemoteOK",
			Category:    "jobs",
			Tags:        job.Tags,
			Salary:      salary,
			Company:     job.Company,
			Location:    job.Location,
			PostedAt:    job.Date,
		})
	}

	return items, nil
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
