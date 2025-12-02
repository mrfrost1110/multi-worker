package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/multi-worker/internal/model"
)

// ==================== Glints Real Job Scraper ====================

// GlintsRealJobScraper scrapes actual job listings from Glints API
type GlintsRealJobScraper struct {
	client *HTTPClient
}

func NewGlintsRealJobScraper(client *HTTPClient) *GlintsRealJobScraper {
	return &GlintsRealJobScraper{client: client}
}

func (s *GlintsRealJobScraper) Name() string {
	return "glints_jobs"
}

func (s *GlintsRealJobScraper) Category() string {
	return "jobs"
}

// Glints GraphQL API response structure
type glintsGraphQLResponse struct {
	Data struct {
		Jobs struct {
			Data []struct {
				ID         string `json:"id"`
				Title      string `json:"title"`
				CreatedAt  string `json:"createdAt"`
				CityName   string `json:"cityName"`
				IsRemote   bool   `json:"isRemote"`
				SalaryEstimate struct {
					MinAmount int    `json:"minAmount"`
					MaxAmount int    `json:"maxAmount"`
					Currency  string `json:"currency"`
				} `json:"salaryEstimate"`
				Company struct {
					Name string `json:"name"`
					Logo string `json:"logo"`
				} `json:"company"`
				Skills []struct {
					Name string `json:"name"`
				} `json:"skills"`
				MinYearsOfExperience int    `json:"minYearsOfExperience"`
				MaxYearsOfExperience int    `json:"maxYearsOfExperience"`
				EducationLevel       string `json:"educationLevel"`
				JobDescription       string `json:"jobDescription"`
			} `json:"data"`
		} `json:"jobs"`
	} `json:"data"`
}

func (s *GlintsRealJobScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "admin"
	}

	// Glints GraphQL endpoint
	graphqlURL := "https://glints.com/api/graphql"

	// GraphQL query for job search
	graphqlQuery := `{
		"operationName": "searchJobs",
		"variables": {
			"data": {
				"SearchTerm": "%s",
				"CountryCode": "ID",
				"CityName": ["Jakarta", "Bekasi"],
				"limit": %d,
				"offset": 0
			}
		},
		"query": "query searchJobs($data: JobSearchConditionInput!) { jobs(data: $data) { data { id title createdAt cityName isRemote salaryEstimate { minAmount maxAmount currency } company { name logo } skills { name } minYearsOfExperience maxYearsOfExperience educationLevel jobDescription } } }"
	}`

	queryBody := fmt.Sprintf(graphqlQuery, searchQuery, limit)

	data, err := s.client.PostJSON(ctx, graphqlURL, bytes.NewBufferString(queryBody))
	if err != nil {
		// Fallback to simple API
		return s.scrapeSimpleAPI(ctx, searchQuery, limit)
	}

	var response glintsGraphQLResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return s.scrapeSimpleAPI(ctx, searchQuery, limit)
	}

	var items []model.ScrapedItem
	for _, job := range response.Data.Jobs.Data {
		salary := ""
		if job.SalaryEstimate.MinAmount > 0 || job.SalaryEstimate.MaxAmount > 0 {
			salary = fmt.Sprintf("%s %s - %s juta/bulan",
				job.SalaryEstimate.Currency,
				formatSalary(job.SalaryEstimate.MinAmount),
				formatSalary(job.SalaryEstimate.MaxAmount))
		}

		location := job.CityName
		if job.IsRemote {
			location = "Remote / " + location
		}

		var skills []string
		for _, skill := range job.Skills {
			skills = append(skills, skill.Name)
		}

		requirements := ""
		if job.EducationLevel != "" {
			requirements = fmt.Sprintf("Pendidikan: %s", job.EducationLevel)
		}
		if job.MinYearsOfExperience > 0 {
			requirements += fmt.Sprintf(", Pengalaman: %d-%d tahun", job.MinYearsOfExperience, job.MaxYearsOfExperience)
		}

		description := truncateText(cleanHTML(job.JobDescription), 500)
		if requirements != "" {
			description = requirements + "\n\n" + description
		}

		items = append(items, model.ScrapedItem{
			ID:          job.ID,
			Title:       job.Title,
			Description: description,
			URL:         fmt.Sprintf("https://glints.com/id/opportunities/jobs/%s", job.ID),
			Source:      "Glints",
			Category:    "jobs",
			Tags:        skills,
			Salary:      salary,
			Company:     job.Company.Name,
			Location:    location,
			PostedAt:    job.CreatedAt,
		})
	}

	return items, nil
}

func (s *GlintsRealJobScraper) scrapeSimpleAPI(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	// Fallback to simple search API
	apiURL := fmt.Sprintf("https://glints.com/api/v2/jobs/search?country=ID&keyword=%s&city=Jakarta,Bekasi&limit=%d",
		url.QueryEscape(query), limit)

	data, err := s.client.GetJSON(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("glints API error: %w", err)
	}

	var response struct {
		Data []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			CompanyName string `json:"companyName"`
			Location    string `json:"location"`
			Salary      string `json:"salary"`
			Description string `json:"description"`
			CreatedAt   string `json:"createdAt"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	var items []model.ScrapedItem
	for _, job := range response.Data {
		items = append(items, model.ScrapedItem{
			ID:          job.ID,
			Title:       job.Title,
			Description: truncateText(job.Description, 500),
			URL:         fmt.Sprintf("https://glints.com/id/opportunities/jobs/%s", job.ID),
			Source:      "Glints",
			Category:    "jobs",
			Company:     job.CompanyName,
			Location:    job.Location,
			Salary:      job.Salary,
			PostedAt:    job.CreatedAt,
		})
	}

	return items, nil
}

// ==================== Jobstreet Real Job Scraper ====================

// JobstreetRealScraper scrapes actual job listings from Jobstreet
type JobstreetRealScraper struct {
	client *HTTPClient
}

func NewJobstreetRealScraper(client *HTTPClient) *JobstreetRealScraper {
	return &JobstreetRealScraper{client: client}
}

func (s *JobstreetRealScraper) Name() string {
	return "jobstreet_jobs"
}

func (s *JobstreetRealScraper) Category() string {
	return "jobs"
}

type jobstreetAPIResponse struct {
	Data struct {
		Jobs struct {
			Jobs []struct {
				ID           string `json:"id"`
				Title        string `json:"title"`
				JobURL       string `json:"jobUrl"`
				Company      struct {
					Name string `json:"name"`
				} `json:"company"`
				Location struct {
					Label string `json:"label"`
				} `json:"location"`
				Salary struct {
					Label string `json:"label"`
				} `json:"salary"`
				ListingDate  string   `json:"listingDate"`
				WorkTypes    []string `json:"workTypes"`
				Teaser       string   `json:"teaser"`
				Requirements string   `json:"requirements"`
			} `json:"jobs"`
		} `json:"jobs"`
	} `json:"data"`
}

func (s *JobstreetRealScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "admin"
	}

	// Jobstreet GraphQL API
	graphqlURL := "https://xapi.supercharge-srp.co/job-search/graphql"

	graphqlQuery := `{
		"operationName": "GetJobList",
		"variables": {
			"country": "id",
			"locale": "id",
			"keyword": "%s",
			"locationId": ["jakarta", "bekasi"],
			"pageSize": %d,
			"page": 1
		},
		"query": "query GetJobList($country: String!, $locale: String!, $keyword: String, $locationId: [String], $pageSize: Int, $page: Int) { jobs(country: $country, locale: $locale, keyword: $keyword, locationId: $locationId, pageSize: $pageSize, page: $page) { jobs { id title jobUrl company { name } location { label } salary { label } listingDate workTypes teaser } } }"
	}`

	queryBody := fmt.Sprintf(graphqlQuery, searchQuery, limit)

	data, err := s.client.PostJSON(ctx, graphqlURL, bytes.NewBufferString(queryBody))
	if err != nil {
		// Fallback to HTML scraping
		return s.scrapeHTML(ctx, searchQuery, limit)
	}

	var response jobstreetAPIResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return s.scrapeHTML(ctx, searchQuery, limit)
	}

	var items []model.ScrapedItem
	for _, job := range response.Data.Jobs.Jobs {
		workType := ""
		if len(job.WorkTypes) > 0 {
			workType = strings.Join(job.WorkTypes, ", ")
		}

		description := job.Teaser
		if workType != "" {
			description = fmt.Sprintf("Tipe: %s\n\n%s", workType, description)
		}

		items = append(items, model.ScrapedItem{
			ID:          job.ID,
			Title:       job.Title,
			Description: truncateText(description, 500),
			URL:         job.JobURL,
			Source:      "Jobstreet",
			Category:    "jobs",
			Tags:        job.WorkTypes,
			Salary:      job.Salary.Label,
			Company:     job.Company.Name,
			Location:    job.Location.Label,
			PostedAt:    job.ListingDate,
		})
	}

	return items, nil
}

func (s *JobstreetRealScraper) scrapeHTML(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchURL := fmt.Sprintf("https://www.jobstreet.co.id/id/%s-jobs-in-jakarta",
		strings.ReplaceAll(strings.ToLower(query), " ", "-"))

	htmlData, err := s.client.Get(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	return parseJobstreetHTML(string(htmlData), limit)
}

func parseJobstreetHTML(html string, limit int) ([]model.ScrapedItem, error) {
	var items []model.ScrapedItem

	// Extract job cards using regex
	jobPattern := regexp.MustCompile(`<article[^>]*data-job-id="([^"]+)"[^>]*>[\s\S]*?<h1[^>]*>[\s\S]*?<a[^>]*href="([^"]+)"[^>]*>([^<]+)</a>[\s\S]*?<span[^>]*data-automation="jobCardCompanyLink"[^>]*>([^<]+)</span>[\s\S]*?<span[^>]*data-automation="jobCardLocation"[^>]*>([^<]+)</span>`)

	matches := jobPattern.FindAllStringSubmatch(html, limit)

	for _, match := range matches {
		if len(match) >= 6 {
			items = append(items, model.ScrapedItem{
				ID:       match[1],
				Title:    strings.TrimSpace(match[3]),
				URL:      "https://www.jobstreet.co.id" + match[2],
				Source:   "Jobstreet",
				Category: "jobs",
				Company:  strings.TrimSpace(match[4]),
				Location: strings.TrimSpace(match[5]),
			})
		}
	}

	return items, nil
}

// ==================== Kalibrr Real Job Scraper ====================

// KalibrrRealScraper scrapes actual job listings from Kalibrr
type KalibrrRealScraper struct {
	client *HTTPClient
}

func NewKalibrrRealScraper(client *HTTPClient) *KalibrrRealScraper {
	return &KalibrrRealScraper{client: client}
}

func (s *KalibrrRealScraper) Name() string {
	return "kalibrr_jobs"
}

func (s *KalibrrRealScraper) Category() string {
	return "jobs"
}

type kalibrrResponse struct {
	Jobs []struct {
		ID              int    `json:"id"`
		Name            string `json:"name"`
		Slug            string `json:"slug"`
		SalaryRangeFrom int    `json:"salary_range_from"`
		SalaryRangeTo   int    `json:"salary_range_to"`
		Location        string `json:"google_location"`
		Company         struct {
			Name string `json:"name"`
			Logo string `json:"logo"`
		} `json:"company"`
		FunctionNames           []string `json:"function_names"`
		Qualifications          string   `json:"qualifications"`
		Responsibilities        string   `json:"responsibilities"`
		EducationLevelName      string   `json:"education_level_name"`
		YearsOfExperienceMin    int      `json:"years_of_experience_min"`
		YearsOfExperienceMax    int      `json:"years_of_experience_max"`
		ApplicationInstructions string   `json:"application_instructions"`
		CreatedAt               string   `json:"created_at"`
		WorkSetupName           string   `json:"work_setup_name"`
	} `json:"jobs"`
}

func (s *KalibrrRealScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "admin"
	}

	// Kalibrr API endpoint
	apiURL := fmt.Sprintf("https://www.kalibrr.com/kjs/job_board/search?keywords=%s&country=Indonesia&city=Jakarta,Bekasi&sort=Freshness&limit=%d",
		url.QueryEscape(searchQuery), limit)

	data, err := s.client.GetJSON(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("kalibrr API error: %w", err)
	}

	var response kalibrrResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	var items []model.ScrapedItem
	for _, job := range response.Jobs {
		salary := ""
		if job.SalaryRangeFrom > 0 || job.SalaryRangeTo > 0 {
			salary = fmt.Sprintf("Rp %s - %s / bulan",
				formatSalaryIDR(job.SalaryRangeFrom),
				formatSalaryIDR(job.SalaryRangeTo))
		}

		requirements := ""
		if job.EducationLevelName != "" {
			requirements = fmt.Sprintf("Pendidikan: %s", job.EducationLevelName)
		}
		if job.YearsOfExperienceMin > 0 || job.YearsOfExperienceMax > 0 {
			requirements += fmt.Sprintf("\nPengalaman: %d-%d tahun", job.YearsOfExperienceMin, job.YearsOfExperienceMax)
		}

		description := cleanHTML(job.Qualifications)
		if requirements != "" {
			description = requirements + "\n\n" + description
		}
		if job.WorkSetupName != "" {
			description = fmt.Sprintf("Tipe Kerja: %s\n\n%s", job.WorkSetupName, description)
		}

		items = append(items, model.ScrapedItem{
			ID:          fmt.Sprintf("%d", job.ID),
			Title:       job.Name,
			Description: truncateText(description, 500),
			URL:         fmt.Sprintf("https://www.kalibrr.com/c/%s/jobs/%d/%s", job.Company.Name, job.ID, job.Slug),
			Source:      "Kalibrr",
			Category:    "jobs",
			Tags:        job.FunctionNames,
			Salary:      salary,
			Company:     job.Company.Name,
			Location:    job.Location,
			PostedAt:    job.CreatedAt,
		})
	}

	return items, nil
}

// ==================== Indeed Indonesia Real Scraper ====================

// IndeedRealScraper scrapes actual job listings from Indeed Indonesia
type IndeedRealScraper struct {
	client *HTTPClient
}

func NewIndeedRealScraper(client *HTTPClient) *IndeedRealScraper {
	return &IndeedRealScraper{client: client}
}

func (s *IndeedRealScraper) Name() string {
	return "indeed_jobs"
}

func (s *IndeedRealScraper) Category() string {
	return "jobs"
}

func (s *IndeedRealScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "admin"
	}

	// Indeed search URL
	searchURL := fmt.Sprintf("https://id.indeed.com/jobs?q=%s&l=Jakarta&limit=%d&fromage=7",
		url.QueryEscape(searchQuery), limit)

	htmlData, err := s.client.Get(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	return parseIndeedHTML(string(htmlData), limit)
}

func parseIndeedHTML(html string, limit int) ([]model.ScrapedItem, error) {
	var items []model.ScrapedItem

	// Extract job cards using regex patterns
	titlePattern := regexp.MustCompile(`<h2[^>]*class="[^"]*jobTitle[^"]*"[^>]*>[\s\S]*?<span[^>]*>([^<]+)</span>`)
	companyPattern := regexp.MustCompile(`data-testid="company-name"[^>]*>([^<]+)<`)
	locationPattern := regexp.MustCompile(`data-testid="text-location"[^>]*>([^<]+)<`)
	salaryPattern := regexp.MustCompile(`<div[^>]*class="[^"]*salary-snippet[^"]*"[^>]*>([^<]+)<`)

	// Split by job cards
	jobCards := strings.Split(html, `data-jk="`)

	count := 0
	for i, card := range jobCards {
		if i == 0 || count >= limit {
			continue
		}

		// Extract job key
		jkMatch := regexp.MustCompile(`^([^"]+)`).FindStringSubmatch(card)
		if len(jkMatch) < 2 {
			continue
		}
		jobKey := jkMatch[1]

		// Extract title
		title := ""
		titleMatch := titlePattern.FindStringSubmatch(card)
		if len(titleMatch) >= 2 {
			title = strings.TrimSpace(titleMatch[1])
		}

		// Extract company
		company := ""
		companyMatch := companyPattern.FindStringSubmatch(card)
		if len(companyMatch) >= 2 {
			company = strings.TrimSpace(companyMatch[1])
		}

		// Extract location
		location := ""
		locationMatch := locationPattern.FindStringSubmatch(card)
		if len(locationMatch) >= 2 {
			location = strings.TrimSpace(locationMatch[1])
		}

		// Extract salary
		salary := ""
		salaryMatch := salaryPattern.FindStringSubmatch(card)
		if len(salaryMatch) >= 2 {
			salary = strings.TrimSpace(salaryMatch[1])
		}

		if title != "" {
			items = append(items, model.ScrapedItem{
				ID:       jobKey,
				Title:    title,
				URL:      fmt.Sprintf("https://id.indeed.com/viewjob?jk=%s", jobKey),
				Source:   "Indeed Indonesia",
				Category: "jobs",
				Company:  company,
				Location: location,
				Salary:   salary,
			})
			count++
		}
	}

	return items, nil
}

// ==================== Combined Jakarta/Bekasi Real Job Scraper ====================

// JakartaBekasiRealScraper combines multiple sources for Jakarta/Bekasi jobs
type JakartaBekasiRealScraper struct {
	client *HTTPClient
}

func NewJakartaBekasiRealScraper(client *HTTPClient) *JakartaBekasiRealScraper {
	return &JakartaBekasiRealScraper{client: client}
}

func (s *JakartaBekasiRealScraper) Name() string {
	return "jakarta_bekasi_jobs"
}

func (s *JakartaBekasiRealScraper) Category() string {
	return "jobs"
}

func (s *JakartaBekasiRealScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	var allItems []model.ScrapedItem

	// Try Glints first (most reliable API)
	glintsScraper := NewGlintsRealJobScraper(s.client)
	glintsItems, err := glintsScraper.Scrape(ctx, query, limit/3+1)
	if err == nil {
		allItems = append(allItems, glintsItems...)
	}

	// Try Kalibrr
	kalibrrScraper := NewKalibrrRealScraper(s.client)
	kalibrrItems, err := kalibrrScraper.Scrape(ctx, query, limit/3+1)
	if err == nil {
		allItems = append(allItems, kalibrrItems...)
	}

	// Try Indeed
	indeedScraper := NewIndeedRealScraper(s.client)
	indeedItems, err := indeedScraper.Scrape(ctx, query, limit/3+1)
	if err == nil {
		allItems = append(allItems, indeedItems...)
	}

	// Limit results
	if limit > 0 && len(allItems) > limit {
		allItems = allItems[:limit]
	}

	return allItems, nil
}

// ==================== Entry Level Real Job Scraper ====================

// EntryLevelRealScraper scrapes entry-level jobs suitable for SMA/SMK graduates
type EntryLevelRealScraper struct {
	client *HTTPClient
}

func NewEntryLevelRealScraper(client *HTTPClient) *EntryLevelRealScraper {
	return &EntryLevelRealScraper{client: client}
}

func (s *EntryLevelRealScraper) Name() string {
	return "entry_level_jobs"
}

func (s *EntryLevelRealScraper) Category() string {
	return "jobs"
}

func (s *EntryLevelRealScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "admin"
	}

	// Use combined scraper
	combinedScraper := NewJakartaBekasiRealScraper(s.client)
	return combinedScraper.Scrape(ctx, searchQuery, limit)
}

// ==================== Remote Jakarta Real Scraper ====================

// RemoteJakartaRealScraper scrapes remote jobs that accept Indonesia workers
type RemoteJakartaRealScraper struct {
	client *HTTPClient
}

func NewRemoteJakartaRealScraper(client *HTTPClient) *RemoteJakartaRealScraper {
	return &RemoteJakartaRealScraper{client: client}
}

func (s *RemoteJakartaRealScraper) Name() string {
	return "remote_jakarta"
}

func (s *RemoteJakartaRealScraper) Category() string {
	return "jobs"
}

func (s *RemoteJakartaRealScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	// Use RemoteOK API and filter for Indonesia-friendly jobs
	apiURL := "https://remoteok.com/api"
	if query != "" {
		apiURL = fmt.Sprintf("https://remoteok.com/api?tag=%s", strings.ReplaceAll(query, " ", "+"))
	}

	data, err := s.client.GetJSON(ctx, apiURL)
	if err != nil {
		return nil, err
	}

	var jobs []remoteOKJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, err
	}

	// Skip first item (metadata)
	if len(jobs) > 0 {
		jobs = jobs[1:]
	}

	// Filter for worldwide/Asia-friendly jobs
	var items []model.ScrapedItem
	count := 0
	for _, job := range jobs {
		location := strings.ToLower(job.Location)

		// Include worldwide remote or Asia-based
		if location == "" ||
			strings.Contains(location, "worldwide") ||
			strings.Contains(location, "asia") ||
			strings.Contains(location, "remote") ||
			strings.Contains(location, "anywhere") ||
			strings.Contains(location, "indonesia") {

			salary := ""
			if job.SalaryMin > 0 || job.SalaryMax > 0 {
				salary = fmt.Sprintf("$%d - $%d/year", job.SalaryMin, job.SalaryMax)
			}

			items = append(items, model.ScrapedItem{
				ID:          job.ID,
				Title:       job.Position,
				Description: truncateText(job.Description, 500),
				URL:         fmt.Sprintf("https://remoteok.com/remote-jobs/%s", job.Slug),
				Source:      "RemoteOK (Indonesia OK)",
				Category:    "jobs",
				Tags:        append(job.Tags, "remote", "indonesia-ok"),
				Salary:      salary,
				Company:     job.Company,
				Location:    "🏠 Remote (Indonesia OK)",
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

// ==================== Loker Jakarta Real Scraper ====================

// LokerJakartaRealScraper scrapes jobs from multiple local Indonesian portals
type LokerJakartaRealScraper struct {
	client *HTTPClient
}

func NewLokerJakartaRealScraper(client *HTTPClient) *LokerJakartaRealScraper {
	return &LokerJakartaRealScraper{client: client}
}

func (s *LokerJakartaRealScraper) Name() string {
	return "loker_jakarta"
}

func (s *LokerJakartaRealScraper) Category() string {
	return "jobs"
}

func (s *LokerJakartaRealScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	// Use combined scraper
	combinedScraper := NewJakartaBekasiRealScraper(s.client)
	return combinedScraper.Scrape(ctx, query, limit)
}

// ==================== Helper Functions ====================

func formatSalary(amount int) string {
	if amount >= 1000000 {
		return fmt.Sprintf("%.1f", float64(amount)/1000000)
	}
	return fmt.Sprintf("%d", amount)
}

func formatSalaryIDR(amount int) string {
	if amount >= 1000000 {
		return fmt.Sprintf("%.0fjt", float64(amount)/1000000)
	} else if amount >= 1000 {
		return fmt.Sprintf("%.0frb", float64(amount)/1000)
	}
	return fmt.Sprintf("%d", amount)
}

func cleanHTML(html string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, " ")

	// Clean up whitespace
	text = strings.Join(strings.Fields(text), " ")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	return strings.TrimSpace(text)
}
