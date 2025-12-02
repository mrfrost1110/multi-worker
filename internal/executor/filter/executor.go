package filter

import (
	"context"
	"fmt"
	"strings"

	"github.com/multi-worker/internal/model"
	"github.com/multi-worker/internal/storage"
)

// Executor handles filtering and deduplication in pipelines
type Executor struct {
	cache *storage.CacheRepository
}

// NewExecutor creates a new filter executor
func NewExecutor(cache *storage.CacheRepository) *Executor {
	return &Executor{cache: cache}
}

func (e *Executor) Type() string {
	return "filter"
}

func (e *Executor) Validate(config map[string]interface{}) error {
	return nil // All configs are optional
}

func (e *Executor) Execute(ctx context.Context, input *model.ExecutorResult, config map[string]interface{}) (*model.ExecutorResult, error) {
	if input == nil || input.Data == nil {
		return input, nil
	}

	// Get configuration
	includeKeywords := getStringSlice(config, "include_keywords")
	excludeKeywords := getStringSlice(config, "exclude_keywords")
	dedupe, _ := config["deduplicate"].(bool)
	taskID, _ := config["task_id"].(string)
	limit := 0
	if l, ok := config["limit"].(float64); ok {
		limit = int(l)
	}

	var filtered interface{}
	var count int

	switch v := input.Data.(type) {
	case []model.ScrapedItem:
		items := filterScrapedItems(v, includeKeywords, excludeKeywords)
		if dedupe && e.cache != nil && taskID != "" {
			items = e.dedupeScrapedItems(ctx, items, taskID)
		}
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}
		filtered = items
		count = len(items)

	case []model.RSSItem:
		items := filterRSSItems(v, includeKeywords, excludeKeywords)
		if dedupe && e.cache != nil && taskID != "" {
			items = e.dedupeRSSItems(ctx, items, taskID)
		}
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}
		filtered = items
		count = len(items)

	default:
		return input, nil
	}

	return &model.ExecutorResult{
		Data:      filtered,
		Metadata:  input.Metadata,
		ItemCount: count,
	}, nil
}

func filterScrapedItems(items []model.ScrapedItem, include, exclude []string) []model.ScrapedItem {
	if len(include) == 0 && len(exclude) == 0 {
		return items
	}

	var filtered []model.ScrapedItem
	for _, item := range items {
		text := strings.ToLower(item.Title + " " + item.Description + " " + strings.Join(item.Tags, " "))

		// Check exclude first
		excluded := false
		for _, keyword := range exclude {
			if strings.Contains(text, strings.ToLower(keyword)) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Check include (if specified, at least one must match)
		if len(include) > 0 {
			included := false
			for _, keyword := range include {
				if strings.Contains(text, strings.ToLower(keyword)) {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}

		filtered = append(filtered, item)
	}

	return filtered
}

func filterRSSItems(items []model.RSSItem, include, exclude []string) []model.RSSItem {
	if len(include) == 0 && len(exclude) == 0 {
		return items
	}

	var filtered []model.RSSItem
	for _, item := range items {
		text := strings.ToLower(item.Title + " " + item.Description)

		excluded := false
		for _, keyword := range exclude {
			if strings.Contains(text, strings.ToLower(keyword)) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		if len(include) > 0 {
			included := false
			for _, keyword := range include {
				if strings.Contains(text, strings.ToLower(keyword)) {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}

		filtered = append(filtered, item)
	}

	return filtered
}

func (e *Executor) dedupeScrapedItems(ctx context.Context, items []model.ScrapedItem, taskID string) []model.ScrapedItem {
	var unique []model.ScrapedItem
	var hashes []string

	for _, item := range items {
		content := item.URL
		if content == "" {
			content = item.ID + item.Source
		}
		hash := e.cache.HashContent(content)

		exists, _ := e.cache.ExistsForTask(ctx, hash, taskID)
		if exists {
			continue
		}

		unique = append(unique, item)
		hashes = append(hashes, hash)
	}

	if len(hashes) > 0 {
		e.cache.AddBatch(ctx, hashes, "filter", taskID)
	}

	return unique
}

func (e *Executor) dedupeRSSItems(ctx context.Context, items []model.RSSItem, taskID string) []model.RSSItem {
	var unique []model.RSSItem
	var hashes []string

	for _, item := range items {
		content := item.Link
		if content == "" {
			content = item.ID
		}
		hash := e.cache.HashContent(content)

		exists, _ := e.cache.ExistsForTask(ctx, hash, taskID)
		if exists {
			continue
		}

		unique = append(unique, item)
		hashes = append(hashes, hash)
	}

	if len(hashes) > 0 {
		e.cache.AddBatch(ctx, hashes, "filter", taskID)
	}

	return unique
}

func getStringSlice(config map[string]interface{}, key string) []string {
	var result []string

	if arr, ok := config[key].([]interface{}); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
	}

	if s, ok := config[key].(string); ok && s != "" {
		result = append(result, s)
	}

	return result
}

// Helper function to create a filter step config
func NewFilterConfig(includeKeywords, excludeKeywords []string, dedupe bool) map[string]interface{} {
	return map[string]interface{}{
		"include_keywords": includeKeywords,
		"exclude_keywords": excludeKeywords,
		"deduplicate":      dedupe,
	}
}

// SkipEmpty checks if results should be skipped (empty)
func SkipEmpty(result *model.ExecutorResult) bool {
	if result == nil || result.Data == nil {
		return true
	}

	switch v := result.Data.(type) {
	case []model.ScrapedItem:
		return len(v) == 0
	case []model.RSSItem:
		return len(v) == 0
	case string:
		return v == ""
	default:
		return false
	}
}

// Error types for pipeline control
type SkipPipelineError struct {
	Reason string
}

func (e SkipPipelineError) Error() string {
	return fmt.Sprintf("pipeline skipped: %s", e.Reason)
}

func NewSkipPipelineError(reason string) SkipPipelineError {
	return SkipPipelineError{Reason: reason}
}
