package scraper

import (
	"context"
	"fmt"

	"github.com/multi-worker/internal/config"
	"github.com/multi-worker/internal/model"
)

// Source interface for all job/content scrapers
type Source interface {
	Name() string
	Category() string // "jobs", "freelance", "news"
	Scrape(ctx context.Context, query string, limit int) ([]model.ScrapedItem, error)
}

// Registry manages all scraper sources
type Registry struct {
	sources map[string]Source
	client  *HTTPClient
}

// NewRegistry creates a new scraper registry
func NewRegistry(cfg config.ScraperConfig) *Registry {
	client := NewHTTPClient(cfg)
	registry := &Registry{
		sources: make(map[string]Source),
		client:  client,
	}

	// Register job board scrapers
	registry.sources["remoteok"] = NewRemoteOKScraper(client)
	registry.sources["hackernews_jobs"] = NewHackerNewsJobsScraper(client)
	registry.sources["github_jobs"] = NewGitHubJobsScraper(client)
	registry.sources["stackoverflow_jobs"] = NewStackOverflowJobsScraper(client)
	registry.sources["weworkremotely"] = NewWeWorkRemotelyScraper(client)

	// Register freelance scrapers
	registry.sources["freelancer"] = NewFreelancerScraper(client)
	registry.sources["upwork"] = NewUpworkScraper(client)

	// Register tech news scrapers
	registry.sources["hackernews"] = NewHackerNewsScraper(client)
	registry.sources["devto"] = NewDevToScraper(client)
	registry.sources["producthunt"] = NewProductHuntScraper(client)

	return registry
}

// Get returns a source by name
func (r *Registry) Get(name string) (Source, error) {
	source, ok := r.sources[name]
	if !ok {
		return nil, fmt.Errorf("scraper source '%s' not found", name)
	}
	return source, nil
}

// GetByCategory returns all sources in a category
func (r *Registry) GetByCategory(category string) []Source {
	var sources []Source
	for _, source := range r.sources {
		if source.Category() == category {
			sources = append(sources, source)
		}
	}
	return sources
}

// Available returns list of available source names
func (r *Registry) Available() []string {
	names := make([]string, 0, len(r.sources))
	for name := range r.sources {
		names = append(names, name)
	}
	return names
}

// AvailableByCategory returns list of sources grouped by category
func (r *Registry) AvailableByCategory() map[string][]string {
	result := make(map[string][]string)
	for name, source := range r.sources {
		category := source.Category()
		result[category] = append(result[category], name)
	}
	return result
}
