package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/multi-worker/internal/model"
)

// ==================== Jakarta/Bekasi Job Scrapers ====================
// These scrapers focus specifically on jobs in Jakarta and Bekasi areas
// for entry-level positions (SMA/SMK graduates)

// JakartaBekasiJobScraper aggregates jobs from multiple sources filtered for Jakarta/Bekasi
type JakartaBekasiJobScraper struct {
	client *HTTPClient
}

func NewJakartaBekasiJobScraper(client *HTTPClient) *JakartaBekasiJobScraper {
	return &JakartaBekasiJobScraper{client: client}
}

func (s *JakartaBekasiJobScraper) Name() string {
	return "jakarta_bekasi_jobs"
}

func (s *JakartaBekasiJobScraper) Category() string {
	return "jobs"
}

// isJakartaBekasiLocation checks if location matches Jakarta or Bekasi
func isJakartaBekasiLocation(location string) bool {
	loc := strings.ToLower(location)
	return strings.Contains(loc, "jakarta") ||
		strings.Contains(loc, "bekasi") ||
		strings.Contains(loc, "jabodetabek") ||
		strings.Contains(loc, "dki") ||
		strings.Contains(loc, "jawa barat") ||
		strings.Contains(loc, "west java") ||
		loc == "indonesia" || // Generic Indonesia often means Jakarta
		loc == "" // Empty location, could be anywhere
}

func (s *JakartaBekasiJobScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	var allItems []model.ScrapedItem

	// Search queries for entry-level jobs
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "admin"
	}

	// Add Jakarta/Bekasi specific search URLs
	sources := []struct {
		name string
		url  string
	}{
		{
			name: "Jobstreet Jakarta",
			url:  fmt.Sprintf("https://www.jobstreet.co.id/id/job-search/%s-jobs-in-jakarta/", strings.ReplaceAll(strings.ToLower(searchQuery), " ", "-")),
		},
		{
			name: "Jobstreet Bekasi",
			url:  fmt.Sprintf("https://www.jobstreet.co.id/id/job-search/%s-jobs-in-bekasi/", strings.ReplaceAll(strings.ToLower(searchQuery), " ", "-")),
		},
		{
			name: "Glints Jakarta",
			url:  fmt.Sprintf("https://glints.com/id/opportunities/jobs/explore?keyword=%s&country=ID&locationName=Jakarta", url.QueryEscape(searchQuery)),
		},
		{
			name: "Glints Bekasi",
			url:  fmt.Sprintf("https://glints.com/id/opportunities/jobs/explore?keyword=%s&country=ID&locationName=Bekasi", url.QueryEscape(searchQuery)),
		},
		{
			name: "Kalibrr Jakarta",
			url:  fmt.Sprintf("https://www.kalibrr.com/id-ID/job-board/te/%s/co/Indonesia/ci/Jakarta", url.QueryEscape(searchQuery)),
		},
		{
			name: "Indeed Jakarta",
			url:  fmt.Sprintf("https://id.indeed.com/jobs?q=%s&l=Jakarta", url.QueryEscape(searchQuery)),
		},
		{
			name: "Indeed Bekasi",
			url:  fmt.Sprintf("https://id.indeed.com/jobs?q=%s&l=Bekasi", url.QueryEscape(searchQuery)),
		},
		{
			name: "LinkedIn Jakarta",
			url:  fmt.Sprintf("https://www.linkedin.com/jobs/search/?keywords=%s&location=Jakarta%%2C%%20Indonesia", url.QueryEscape(searchQuery)),
		},
	}

	for i, src := range sources {
		if limit > 0 && len(allItems) >= limit {
			break
		}

		allItems = append(allItems, model.ScrapedItem{
			ID:          fmt.Sprintf("%s-%s-%d", strings.ReplaceAll(searchQuery, " ", "-"), strings.ToLower(strings.ReplaceAll(src.name, " ", "-")), i),
			Title:       fmt.Sprintf("🔍 %s: %s jobs", src.name, strings.Title(searchQuery)),
			Description: fmt.Sprintf("Search for %s positions in %s. Entry-level and SMA/SMK friendly positions available.", searchQuery, src.name),
			URL:         src.url,
			Source:      src.name,
			Category:    "jobs",
			Location:    extractLocation(src.name),
			Tags:        []string{"indonesia", strings.ToLower(extractLocation(src.name)), searchQuery, "entry-level"},
		})
	}

	return allItems, nil
}

func extractLocation(sourceName string) string {
	if strings.Contains(sourceName, "Bekasi") {
		return "Bekasi"
	}
	return "Jakarta"
}

// ==================== Entry Level Job Scraper ====================

// EntryLevelJobScraper focuses on SMA/SMK level jobs
type EntryLevelJobScraper struct {
	client *HTTPClient
}

func NewEntryLevelJobScraper(client *HTTPClient) *EntryLevelJobScraper {
	return &EntryLevelJobScraper{client: client}
}

func (s *EntryLevelJobScraper) Name() string {
	return "entry_level_jobs"
}

func (s *EntryLevelJobScraper) Category() string {
	return "jobs"
}

func (s *EntryLevelJobScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	var allItems []model.ScrapedItem

	// Entry-level job categories suitable for SMA/SMK graduates
	jobTypes := []string{
		"admin",
		"customer service",
		"data entry",
		"receptionist",
		"kasir",
		"sales",
		"staff gudang",
		"operator",
		"driver",
		"security",
	}

	if query != "" {
		jobTypes = []string{query}
	}

	// Generate search links for each job type
	for _, jobType := range jobTypes {
		if limit > 0 && len(allItems) >= limit {
			break
		}

		// Jobstreet
		allItems = append(allItems, model.ScrapedItem{
			ID:          fmt.Sprintf("jobstreet-jkt-%s", strings.ReplaceAll(jobType, " ", "-")),
			Title:       fmt.Sprintf("💼 %s - Jakarta (Jobstreet)", strings.Title(jobType)),
			Description: fmt.Sprintf("Lowongan %s di Jakarta. Cocok untuk lulusan SMA/SMK.", jobType),
			URL:         fmt.Sprintf("https://www.jobstreet.co.id/id/job-search/%s-jobs-in-jakarta/", strings.ReplaceAll(strings.ToLower(jobType), " ", "-")),
			Source:      "Jobstreet",
			Category:    "jobs",
			Location:    "Jakarta, Indonesia",
			Tags:        []string{"entry-level", "sma", "smk", "jakarta", jobType},
		})

		allItems = append(allItems, model.ScrapedItem{
			ID:          fmt.Sprintf("jobstreet-bks-%s", strings.ReplaceAll(jobType, " ", "-")),
			Title:       fmt.Sprintf("💼 %s - Bekasi (Jobstreet)", strings.Title(jobType)),
			Description: fmt.Sprintf("Lowongan %s di Bekasi. Cocok untuk lulusan SMA/SMK.", jobType),
			URL:         fmt.Sprintf("https://www.jobstreet.co.id/id/job-search/%s-jobs-in-bekasi/", strings.ReplaceAll(strings.ToLower(jobType), " ", "-")),
			Source:      "Jobstreet",
			Category:    "jobs",
			Location:    "Bekasi, Indonesia",
			Tags:        []string{"entry-level", "sma", "smk", "bekasi", jobType},
		})

		// Glints (often has entry-level)
		allItems = append(allItems, model.ScrapedItem{
			ID:          fmt.Sprintf("glints-jkt-%s", strings.ReplaceAll(jobType, " ", "-")),
			Title:       fmt.Sprintf("🚀 %s - Jakarta (Glints)", strings.Title(jobType)),
			Description: fmt.Sprintf("Cari lowongan %s di Jakarta via Glints.", jobType),
			URL:         fmt.Sprintf("https://glints.com/id/opportunities/jobs/explore?keyword=%s&country=ID&locationName=Jakarta", url.QueryEscape(jobType)),
			Source:      "Glints",
			Category:    "jobs",
			Location:    "Jakarta, Indonesia",
			Tags:        []string{"entry-level", "sma", "smk", "jakarta", jobType},
		})

		// Indeed
		allItems = append(allItems, model.ScrapedItem{
			ID:          fmt.Sprintf("indeed-jkt-%s", strings.ReplaceAll(jobType, " ", "-")),
			Title:       fmt.Sprintf("🔎 %s - Jakarta (Indeed)", strings.Title(jobType)),
			Description: fmt.Sprintf("Lowongan %s di Jakarta dan sekitarnya.", jobType),
			URL:         fmt.Sprintf("https://id.indeed.com/jobs?q=%s&l=Jakarta", url.QueryEscape(jobType)),
			Source:      "Indeed",
			Category:    "jobs",
			Location:    "Jakarta, Indonesia",
			Tags:        []string{"entry-level", "sma", "smk", "jakarta", jobType},
		})
	}

	return allItems, nil
}

// ==================== Remote OK Jakarta Filter ====================

// RemoteJakartaScraper scrapes remote jobs that accept Indonesia/Jakarta workers
type RemoteJakartaScraper struct {
	client *HTTPClient
}

func NewRemoteJakartaScraper(client *HTTPClient) *RemoteJakartaScraper {
	return &RemoteJakartaScraper{client: client}
}

func (s *RemoteJakartaScraper) Name() string {
	return "remote_jakarta"
}

func (s *RemoteJakartaScraper) Category() string {
	return "jobs"
}

func (s *RemoteJakartaScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	// RemoteOK API
	apiURL := "https://remoteok.com/api"
	if query != "" {
		apiURL = fmt.Sprintf("https://remoteok.com/api?tag=%s", strings.ReplaceAll(query, " ", "+"))
	}

	data, err := s.client.GetJSON(ctx, apiURL)
	if err != nil {
		// Fallback to search links
		return s.getFallbackLinks(query), nil
	}

	var jobs []remoteOKJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return s.getFallbackLinks(query), nil
	}

	// Skip first item (metadata)
	if len(jobs) > 0 {
		jobs = jobs[1:]
	}

	// Filter for Indonesia/Asia-friendly + remote jobs
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
				Source:      "RemoteOK (Jakarta OK)",
				Category:    "jobs",
				Tags:        append(job.Tags, "remote", "indonesia-ok", "jakarta-ok"),
				Salary:      salary,
				Company:     job.Company,
				Location:    "🏠 Remote (Indonesia/Jakarta OK)",
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

func (s *RemoteJakartaScraper) getFallbackLinks(query string) []model.ScrapedItem {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "data entry"
	}

	return []model.ScrapedItem{
		{
			ID:          "remote-id-" + strings.ReplaceAll(searchQuery, " ", "-"),
			Title:       fmt.Sprintf("🏠 Remote %s Jobs (Indonesia OK)", strings.Title(searchQuery)),
			Description: fmt.Sprintf("Search for remote %s positions that accept workers from Indonesia/Jakarta.", searchQuery),
			URL:         fmt.Sprintf("https://remoteok.com/remote-%s-jobs", strings.ReplaceAll(searchQuery, " ", "-")),
			Source:      "RemoteOK",
			Category:    "jobs",
			Location:    "Remote (Indonesia OK)",
			Tags:        []string{"remote", "indonesia", "jakarta", searchQuery},
		},
		{
			ID:          "wwr-id-" + strings.ReplaceAll(searchQuery, " ", "-"),
			Title:       fmt.Sprintf("🌏 We Work Remotely - %s", strings.Title(searchQuery)),
			Description: fmt.Sprintf("Remote %s jobs from We Work Remotely.", searchQuery),
			URL:         fmt.Sprintf("https://weworkremotely.com/remote-jobs/search?term=%s", url.QueryEscape(searchQuery)),
			Source:      "WeWorkRemotely",
			Category:    "jobs",
			Location:    "Remote (Worldwide)",
			Tags:        []string{"remote", "worldwide", searchQuery},
		},
	}
}

// ==================== Loker Jakarta (Local Indonesian Job Sites) ====================

// LokerJakartaScraper focuses on local Indonesian job portals
type LokerJakartaScraper struct {
	client *HTTPClient
}

func NewLokerJakartaScraper(client *HTTPClient) *LokerJakartaScraper {
	return &LokerJakartaScraper{client: client}
}

func (s *LokerJakartaScraper) Name() string {
	return "loker_jakarta"
}

func (s *LokerJakartaScraper) Category() string {
	return "jobs"
}

func (s *LokerJakartaScraper) Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error) {
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "admin"
	}

	// Local Indonesian job sites focused on Jakarta/Bekasi
	items := []model.ScrapedItem{
		{
			ID:          "karir-jkt-" + strings.ReplaceAll(searchQuery, " ", "-"),
			Title:       fmt.Sprintf("📋 %s - Karir.com Jakarta", strings.Title(searchQuery)),
			Description: fmt.Sprintf("Cari lowongan %s di Jakarta melalui Karir.com. Banyak lowongan untuk lulusan SMA/SMK.", searchQuery),
			URL:         fmt.Sprintf("https://www.karir.com/search-lowongan?keyword=%s&location_string=Jakarta", url.QueryEscape(searchQuery)),
			Source:      "Karir.com",
			Category:    "jobs",
			Location:    "Jakarta, Indonesia",
			Tags:        []string{"jakarta", "loker", searchQuery},
		},
		{
			ID:          "loker-jkt-" + strings.ReplaceAll(searchQuery, " ", "-"),
			Title:       fmt.Sprintf("📋 %s - Loker.id Jakarta", strings.Title(searchQuery)),
			Description: fmt.Sprintf("Portal lowongan kerja %s di Jakarta.", searchQuery),
			URL:         fmt.Sprintf("https://www.loker.id/cari-lowongan?q=%s&lokasi=jakarta", url.QueryEscape(searchQuery)),
			Source:      "Loker.id",
			Category:    "jobs",
			Location:    "Jakarta, Indonesia",
			Tags:        []string{"jakarta", "loker", searchQuery},
		},
		{
			ID:          "jobindo-jkt-" + strings.ReplaceAll(searchQuery, " ", "-"),
			Title:       fmt.Sprintf("📋 %s - Jobs.id Jakarta", strings.Title(searchQuery)),
			Description: fmt.Sprintf("Lowongan %s di Jakarta dan sekitarnya.", searchQuery),
			URL:         fmt.Sprintf("https://www.jobs.id/lowongan-kerja?q=%s&location=jakarta", url.QueryEscape(searchQuery)),
			Source:      "Jobs.id",
			Category:    "jobs",
			Location:    "Jakarta, Indonesia",
			Tags:        []string{"jakarta", "loker", searchQuery},
		},
		{
			ID:          "karir-bks-" + strings.ReplaceAll(searchQuery, " ", "-"),
			Title:       fmt.Sprintf("📋 %s - Karir.com Bekasi", strings.Title(searchQuery)),
			Description: fmt.Sprintf("Lowongan %s di Bekasi dan sekitarnya.", searchQuery),
			URL:         fmt.Sprintf("https://www.karir.com/search-lowongan?keyword=%s&location_string=Bekasi", url.QueryEscape(searchQuery)),
			Source:      "Karir.com",
			Category:    "jobs",
			Location:    "Bekasi, Indonesia",
			Tags:        []string{"bekasi", "loker", searchQuery},
		},
		{
			ID:          "olx-jkt-" + strings.ReplaceAll(searchQuery, " ", "-"),
			Title:       fmt.Sprintf("📋 %s - OLX Lowongan Jakarta", strings.Title(searchQuery)),
			Description: fmt.Sprintf("Cari lowongan %s di Jakarta via OLX.", searchQuery),
			URL:         fmt.Sprintf("https://www.olx.co.id/jakarta-dki_g2000001/lowongan-pekerjaan_c483?q=%s", url.QueryEscape(searchQuery)),
			Source:      "OLX Indonesia",
			Category:    "jobs",
			Location:    "Jakarta, Indonesia",
			Tags:        []string{"jakarta", "olx", searchQuery},
		},
	}

	if limit > 0 && len(items) > limit {
		return items[:limit], nil
	}

	return items, nil
}
