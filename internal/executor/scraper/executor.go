package scraper

import (
	"context"
	"fmt"
	"strings"

	"github.com/multi-worker/internal/model"
	"github.com/multi-worker/internal/storage"
)

// Executor handles web scraping in pipelines
type Executor struct {
	registry *Registry
	cache    *storage.CacheRepository
}

// NewExecutor creates a new scraper executor
func NewExecutor(registry *Registry, cache *storage.CacheRepository) *Executor {
	return &Executor{
		registry: registry,
		cache:    cache,
	}
}

func (e *Executor) Type() string {
	return "scraper"
}

func (e *Executor) Validate(config map[string]interface{}) error {
	if _, ok := config["source"]; !ok {
		// If no specific source, check for sources array
		if _, ok := config["sources"]; !ok {
			return fmt.Errorf("scraper requires 'source' or 'sources' in config")
		}
	}
	return nil
}

func (e *Executor) Execute(ctx context.Context, input *model.ExecutorResult, config map[string]interface{}) (*model.ExecutorResult, error) {
	// Get query/keywords
	query, _ := config["query"].(string)
	keywords, _ := config["keywords"].([]interface{})

	// Build query from keywords if provided
	if len(keywords) > 0 && query == "" {
		var keywordStrs []string
		for _, k := range keywords {
			if s, ok := k.(string); ok {
				keywordStrs = append(keywordStrs, s)
			}
		}
		query = strings.Join(keywordStrs, " ")
	}

	// Get limit
	limit := 10
	if l, ok := config["limit"].(float64); ok {
		limit = int(l)
	}

	// Get task ID for caching
	taskID, _ := config["task_id"].(string)

	// Determine sources to scrape
	var sources []string
	if source, ok := config["source"].(string); ok {
		sources = append(sources, source)
	}
	if sourcesArr, ok := config["sources"].([]interface{}); ok {
		for _, s := range sourcesArr {
			if str, ok := s.(string); ok {
				sources = append(sources, str)
			}
		}
	}

	// If category specified, get all sources in that category
	if category, ok := config["category"].(string); ok {
		categorySources := e.registry.GetByCategory(category)
		for _, s := range categorySources {
			sources = append(sources, s.Name())
		}
	}

	// Scrape from all sources
	var allItems []model.ScrapedItem
	var errors []string

	for _, sourceName := range sources {
		source, err := e.registry.Get(sourceName)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", sourceName, err))
			continue
		}

		items, err := source.Scrape(ctx, query, limit)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", sourceName, err))
			continue
		}

		// Deduplicate using cache
		if e.cache != nil && taskID != "" {
			items = e.filterNewItems(ctx, items, taskID)
		}

		allItems = append(allItems, items...)
	}

	// Build metadata
	metadata := map[string]interface{}{
		"sources":      sources,
		"query":        query,
		"total_items":  len(allItems),
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

// filterNewItems removes items that have been seen before
func (e *Executor) filterNewItems(ctx context.Context, items []model.ScrapedItem, taskID string) []model.ScrapedItem {
	var newItems []model.ScrapedItem
	var newHashes []string

	for _, item := range items {
		// Create unique hash from URL or ID + source
		content := item.URL
		if content == "" {
			content = item.ID + item.Source
		}
		hash := e.cache.HashContent(content)

		exists, err := e.cache.ExistsForTask(ctx, hash, taskID)
		if err != nil || exists {
			continue
		}

		newItems = append(newItems, item)
		newHashes = append(newHashes, hash)
	}

	// Add new hashes to cache
	if len(newHashes) > 0 {
		e.cache.AddBatch(ctx, newHashes, "scraper", taskID)
	}

	return newItems
}

// ScrapeMultiple scrapes from multiple sources concurrently
func (e *Executor) ScrapeMultiple(ctx context.Context, sources []string, query string, limit int) ([]model.ScrapedItem, error) {
	type result struct {
		items []model.ScrapedItem
		err   error
	}

	results := make(chan result, len(sources))

	for _, sourceName := range sources {
		go func(name string) {
			source, err := e.registry.Get(name)
			if err != nil {
				results <- result{err: err}
				return
			}

			items, err := source.Scrape(ctx, query, limit)
			results <- result{items: items, err: err}
		}(sourceName)
	}

	var allItems []model.ScrapedItem
	var errs []error

	for range sources {
		r := <-results
		if r.err != nil {
			errs = append(errs, r.err)
		} else {
			allItems = append(allItems, r.items...)
		}
	}

	if len(allItems) == 0 && len(errs) > 0 {
		return nil, errs[0]
	}

	return allItems, nil
}
