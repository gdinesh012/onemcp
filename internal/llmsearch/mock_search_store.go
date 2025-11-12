package llmsearch

import (
	"log/slog"
	"strings"

	"github.com/radutopala/onemcp/internal/tools"
)

// MockSearchStore is a simple in-memory search store for testing
// It does keyword matching without calling external LLMs
type MockSearchStore struct {
	tools  []*tools.Tool
	logger *slog.Logger
}

// NewMockSearchStore creates a mock search store for testing
func NewMockSearchStore(logger *slog.Logger) *MockSearchStore {
	return &MockSearchStore{
		tools:  make([]*tools.Tool, 0),
		logger: logger,
	}
}

// BuildFromTools stores the tools for searching
func (s *MockSearchStore) BuildFromTools(allTools []*tools.Tool) error {
	s.tools = allTools
	s.logger.Info("Built mock search store", "tool_count", len(allTools))
	return nil
}

// Search performs simple keyword matching for testing
func (s *MockSearchStore) Search(query string, topK int) ([]*tools.Tool, error) {
	if len(s.tools) == 0 {
		return []*tools.Tool{}, nil
	}

	// Simple keyword matching - check if query words appear in tool name or description
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	type scoredTool struct {
		tool  *tools.Tool
		score int
	}

	scored := make([]scoredTool, 0)

	for _, tool := range s.tools {
		score := 0
		nameLower := strings.ToLower(tool.Name)
		descLower := strings.ToLower(tool.Description)
		categoryLower := strings.ToLower(tool.Category)

		// Score based on keyword matches
		for _, word := range queryWords {
			if strings.Contains(nameLower, word) {
				score += 3 // Name match is worth more
			}
			if strings.Contains(descLower, word) {
				score += 2
			}
			if strings.Contains(categoryLower, word) {
				score += 1
			}
		}

		if score > 0 || query == "" {
			scored = append(scored, scoredTool{tool: tool, score: score})
		}
	}

	// Sort by score (simple bubble sort for small test data)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Return top K results
	results := make([]*tools.Tool, 0, topK)
	for i := 0; i < len(scored) && i < topK; i++ {
		results = append(results, scored[i].tool)
	}

	s.logger.Debug("Mock search completed", "query", query, "found", len(results))

	return results, nil
}

// GetToolCount returns the number of tools indexed
func (s *MockSearchStore) GetToolCount() int {
	return len(s.tools)
}
