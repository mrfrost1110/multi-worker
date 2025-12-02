package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/multi-worker/internal/model"
)

// ==================== Glints Indonesia ====================

// GlintsIndonesiaScraper scrapes tech jobs from Glints.com for Indonesia
type GlintsIndonesiaScraper struct {
	client *HTTPClient
}

func NewGlintsIndonesiaScraper(client *HTTPClient) *GlintsIndonesiaScraper {
	return &GlintsIndonesiaScraper{client: client}
}

func (s *GlintsIndonesiaScraper) Name() string {
	return "glints_indonesia"
}

func (s *GlintsIndonesiaScraper) Category() string {
	return "jobs"
}

type glintsResponse struct {
	Data []struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		CompanyName string `json:"companyName"`
		Location    struct {
			City    string `json:"city"`
			Country string `json:"country"`
		} `json:"location"`
		Salary struct {
			Min      int    `json:"min"`
			Max      int    `json:"max"`
			Currency string `json:"currency"`
		} `json:"salary"`
		Skills      []string `json:"skills"`
		Description string   `json:"description"`
		CreatedAt   string   `json:"createdAt"`
	} `json:"data"`
}

func (s *GlintsIndonesiaScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	// Glints API endpoint for Indonesia jobs
	baseURL := "https://glints.com/api/v2/jobs"
	params := url.Values{}
	params.Set("country", "ID")
	params.Set("limit", fmt.Sprintf("%d", limit))
	if query != "" {
		params.Set("keyword", query)
	}

	apiURL := baseURL + "?" + params.Encode()
	data, err := s.client.GetJSON(ctx, apiURL)
	if err != nil {
		// Fallback to search page if API fails
		return s.scrapeSearchPage(ctx, query, limit)
	}

	var response glintsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return s.scrapeSearchPage(ctx, query, limit)
	}

	var items []model.ScrapedItem
	for _, job := range response.Data {
		salary := ""
		if job.Salary.Min > 0 || job.Salary.Max > 0 {
			salary = fmt.Sprintf("%s %d - %d juta", job.Salary.Currency, job.Salary.Min/1000000, job.Salary.Max/1000000)
		}

		location := job.Location.City
		if location == "" {
			location = "Indonesia"
		}

		items = append(items, model.ScrapedItem{
			ID:          job.ID,
			Title:       job.Title,
			Description: truncateText(job.Description, 500),
			URL:         fmt.Sprintf("https://glints.com/id/opportunities/jobs/%s", job.ID),
			Source:      "Glints Indonesia",
			Category:    "jobs",
			Tags:        job.Skills,
			Salary:      salary,
			Company:     job.CompanyName,
			Location:    location,
			PostedAt:    job.CreatedAt,
		})
	}

	return items, nil
}

func (s *GlintsIndonesiaScraper) scrapeSearchPage(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "software engineer"
	}

	searchURL := fmt.Sprintf("https://glints.com/id/opportunities/jobs/explore?keyword=%s&country=ID&locationName=All+Cities%%2FProvinces",
		url.QueryEscape(searchQuery))

	// Return placeholder with search link
	return []model.ScrapedItem{
		{
			ID:          "glints-search",
			Title:       fmt.Sprintf("🔍 Search: %s jobs on Glints Indonesia", searchQuery),
			Description: "Click to browse jobs on Glints Indonesia",
			URL:         searchURL,
			Source:      "Glints Indonesia",
			Category:    "jobs",
			Location:    "Indonesia",
		},
	}, nil
}

// ==================== Jobstreet Indonesia ====================

// JobstreetIndonesiaScraper scrapes jobs from Jobstreet.co.id
type JobstreetIndonesiaScraper struct {
	client *HTTPClient
}

func NewJobstreetIndonesiaScraper(client *HTTPClient) *JobstreetIndonesiaScraper {
	return &JobstreetIndonesiaScraper{client: client}
}

func (s *JobstreetIndonesiaScraper) Name() string {
	return "jobstreet_indonesia"
}

func (s *JobstreetIndonesiaScraper) Category() string {
	return "jobs"
}

func (s *JobstreetIndonesiaScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "developer"
	}

	// Jobstreet Indonesia search URL
	searchURL := fmt.Sprintf("https://www.jobstreet.co.id/id/job-search/%s-jobs/",
		strings.ReplaceAll(strings.ToLower(searchQuery), " ", "-"))

	return []model.ScrapedItem{
		{
			ID:          "jobstreet-search-" + strings.ReplaceAll(searchQuery, " ", "-"),
			Title:       fmt.Sprintf("🇮🇩 %s Jobs on Jobstreet Indonesia", strings.Title(searchQuery)),
			Description: fmt.Sprintf("Find %s jobs in Indonesia. Jobstreet is one of the largest job portals in Southeast Asia.", searchQuery),
			URL:         searchURL,
			Source:      "Jobstreet Indonesia",
			Category:    "jobs",
			Location:    "Indonesia",
			Tags:        []string{"indonesia", searchQuery},
		},
	}, nil
}

// ==================== Kalibrr Indonesia ====================

// KalibrrIndonesiaScraper scrapes tech jobs from Kalibrr
type KalibrrIndonesiaScraper struct {
	client *HTTPClient
}

func NewKalibrrIndonesiaScraper(client *HTTPClient) *KalibrrIndonesiaScraper {
	return &KalibrrIndonesiaScraper{client: client}
}

func (s *KalibrrIndonesiaScraper) Name() string {
	return "kalibrr_indonesia"
}

func (s *KalibrrIndonesiaScraper) Category() string {
	return "jobs"
}

func (s *KalibrrIndonesiaScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "software"
	}

	searchURL := fmt.Sprintf("https://www.kalibrr.com/id-ID/job-board/te/%s/co/Indonesia",
		url.QueryEscape(searchQuery))

	return []model.ScrapedItem{
		{
			ID:          "kalibrr-search-" + strings.ReplaceAll(searchQuery, " ", "-"),
			Title:       fmt.Sprintf("💼 %s Jobs on Kalibrr Indonesia", strings.Title(searchQuery)),
			Description: fmt.Sprintf("Discover %s opportunities in Indonesia. Kalibrr connects you with top tech companies.", searchQuery),
			URL:         searchURL,
			Source:      "Kalibrr Indonesia",
			Category:    "jobs",
			Location:    "Indonesia",
			Tags:        []string{"indonesia", "tech", searchQuery},
		},
	}, nil
}

// ==================== LinkedIn Indonesia ====================

// LinkedInIndonesiaScraper provides LinkedIn job search for Indonesia
type LinkedInIndonesiaScraper struct {
	client *HTTPClient
}

func NewLinkedInIndonesiaScraper(client *HTTPClient) *LinkedInIndonesiaScraper {
	return &LinkedInIndonesiaScraper{client: client}
}

func (s *LinkedInIndonesiaScraper) Name() string {
	return "linkedin_indonesia"
}

func (s *LinkedInIndonesiaScraper) Category() string {
	return "jobs"
}

func (s *LinkedInIndonesiaScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "software engineer"
	}

	// LinkedIn job search URL for Indonesia
	searchURL := fmt.Sprintf("https://www.linkedin.com/jobs/search/?keywords=%s&location=Indonesia&f_TPR=r604800",
		url.QueryEscape(searchQuery))

	return []model.ScrapedItem{
		{
			ID:          "linkedin-id-" + strings.ReplaceAll(searchQuery, " ", "-"),
			Title:       fmt.Sprintf("💼 %s Jobs in Indonesia - LinkedIn", strings.Title(searchQuery)),
			Description: fmt.Sprintf("Browse %s positions in Indonesia on LinkedIn. Filter by experience level, company size, and more.", searchQuery),
			URL:         searchURL,
			Source:      "LinkedIn Indonesia",
			Category:    "jobs",
			Location:    "Indonesia",
			Tags:        []string{"indonesia", "linkedin", searchQuery},
		},
	}, nil
}

// ==================== Indeed Indonesia ====================

// IndeedIndonesiaScraper provides Indeed job search for Indonesia
type IndeedIndonesiaScraper struct {
	client *HTTPClient
}

func NewIndeedIndonesiaScraper(client *HTTPClient) *IndeedIndonesiaScraper {
	return &IndeedIndonesiaScraper{client: client}
}

func (s *IndeedIndonesiaScraper) Name() string {
	return "indeed_indonesia"
}

func (s *IndeedIndonesiaScraper) Category() string {
	return "jobs"
}

func (s *IndeedIndonesiaScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "developer"
	}

	searchURL := fmt.Sprintf("https://id.indeed.com/jobs?q=%s&l=Indonesia",
		url.QueryEscape(searchQuery))

	return []model.ScrapedItem{
		{
			ID:          "indeed-id-" + strings.ReplaceAll(searchQuery, " ", "-"),
			Title:       fmt.Sprintf("🔎 %s Jobs - Indeed Indonesia", strings.Title(searchQuery)),
			Description: fmt.Sprintf("Search %s jobs across Indonesia on Indeed. New jobs added daily.", searchQuery),
			URL:         searchURL,
			Source:      "Indeed Indonesia",
			Category:    "jobs",
			Location:    "Indonesia",
			Tags:        []string{"indonesia", "indeed", searchQuery},
		},
	}, nil
}

// ==================== Techinasia Jobs ====================

// TechInAsiaJobsScraper scrapes startup/tech jobs from Tech in Asia
type TechInAsiaJobsScraper struct {
	client *HTTPClient
}

func NewTechInAsiaJobsScraper(client *HTTPClient) *TechInAsiaJobsScraper {
	return &TechInAsiaJobsScraper{client: client}
}

func (s *TechInAsiaJobsScraper) Name() string {
	return "techinasia_jobs"
}

func (s *TechInAsiaJobsScraper) Category() string {
	return "jobs"
}

func (s *TechInAsiaJobsScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "engineering"
	}

	searchURL := fmt.Sprintf("https://www.techinasia.com/jobs/search?country_name[]=Indonesia&query=%s",
		url.QueryEscape(searchQuery))

	return []model.ScrapedItem{
		{
			ID:          "techinasia-" + strings.ReplaceAll(searchQuery, " ", "-"),
			Title:       fmt.Sprintf("🚀 %s Jobs at Indonesian Startups - Tech in Asia", strings.Title(searchQuery)),
			Description: fmt.Sprintf("Find %s roles at top startups and tech companies in Indonesia.", searchQuery),
			URL:         searchURL,
			Source:      "Tech in Asia",
			Category:    "jobs",
			Location:    "Indonesia",
			Tags:        []string{"indonesia", "startup", "tech", searchQuery},
		},
	}, nil
}

// ==================== Remote OK Indonesia Filter ====================

// RemoteOKIndonesiaScraper filters RemoteOK jobs for Indonesia-friendly positions
type RemoteOKIndonesiaScraper struct {
	client *HTTPClient
}

func NewRemoteOKIndonesiaScraper(client *HTTPClient) *RemoteOKIndonesiaScraper {
	return &RemoteOKIndonesiaScraper{client: client}
}

func (s *RemoteOKIndonesiaScraper) Name() string {
	return "remoteok_indonesia"
}

func (s *RemoteOKIndonesiaScraper) Category() string {
	return "jobs"
}

func (s *RemoteOKIndonesiaScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	// RemoteOK with worldwide filter (includes Indonesia)
	apiURL := "https://remoteok.com/api"
	if query != "" {
		apiURL = fmt.Sprintf("https://remoteok.com/api?tag=%s", strings.ReplaceAll(query, " ", "+"))
	}

	data, err := s.client.GetJSON(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RemoteOK: %w", err)
	}

	var jobs []remoteOKJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse RemoteOK response: %w", err)
	}

	// Skip first item (metadata)
	if len(jobs) > 0 {
		jobs = jobs[1:]
	}

	// Filter for worldwide/Asia-friendly jobs
	var items []model.ScrapedItem
	count := 0
	for _, job := range jobs {
		// Include if location is worldwide, Asia, or not US/EU restricted
		location := strings.ToLower(job.Location)
		if location == "" ||
		   strings.Contains(location, "worldwide") ||
		   strings.Contains(location, "asia") ||
		   strings.Contains(location, "remote") ||
		   strings.Contains(location, "anywhere") {

			salary := ""
			if job.SalaryMin > 0 || job.SalaryMax > 0 {
				salary = fmt.Sprintf("$%d - $%d/year", job.SalaryMin, job.SalaryMax)
			}

			items = append(items, model.ScrapedItem{
				ID:          job.ID,
				Title:       job.Position,
				Description: truncateText(job.Description, 500),
				URL:         fmt.Sprintf("https://remoteok.com/remote-jobs/%s", job.Slug),
				Source:      "RemoteOK (ID-friendly)",
				Category:    "jobs",
				Tags:        append(job.Tags, "remote", "indonesia-friendly"),
				Salary:      salary,
				Company:     job.Company,
				Location:    "🌏 Remote (Indonesia OK)",
				PostedAt:    job.Date,
			})

			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}

	return items, nil
}
