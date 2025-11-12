package llmsearch

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/radutopala/onemcp/internal/tools"
)

// CodexSearchStore uses Codex CLI for semantic search
type CodexSearchStore struct {
	searcher *CodexSearcher
	tools    []*tools.Tool
	schemas  []byte // Cached JSON schemas
	logger   *slog.Logger
}

// NewCodexSearchStore creates a search store that uses Codex CLI
func NewCodexSearchStore(searcher *CodexSearcher, logger *slog.Logger) *CodexSearchStore {
	return &CodexSearchStore{
		searcher: searcher,
		tools:    make([]*tools.Tool, 0),
		logger:   logger,
	}
}

// BuildFromTools caches tool schemas for Codex queries
func (s *CodexSearchStore) BuildFromTools(allTools []*tools.Tool) error {
	s.logger.Info("Building Codex search index", "tool_count", len(allTools))

	s.tools = allTools

	// Build tool metadata with full schemas for Codex
	toolSchemas := make([]tools.ToolMetadata, len(allTools))
	for i, tool := range allTools {
		metadata := tools.ToolMetadata{
			Name:        tool.Name,
			Category:    tool.Category,
			Description: tool.Description,
		}

		// Include full schema
		if tool.InputSchema != nil {
			if schemaMap, ok := tool.InputSchema.(map[string]any); ok {
				metadata.Parameters = schemaMap
			}
		}

		toolSchemas[i] = metadata
	}

	// Marshal to JSON for Codex
	schemas, err := json.Marshal(toolSchemas)
	if err != nil {
		return fmt.Errorf("failed to marshal tool schemas: %w", err)
	}

	s.schemas = schemas

	s.logger.Info("Codex search index built", "tool_count", len(s.tools), "schema_size_kb", len(schemas)/1024)

	return nil
}

// Search uses Codex CLI to find relevant tools
func (s *CodexSearchStore) Search(query string, topK int) ([]*tools.Tool, error) {
	if len(s.tools) == 0 {
		return []*tools.Tool{}, nil
	}

	// Ask Codex to rank tools
	toolNames, err := s.searcher.SearchTools(query, s.schemas, topK)
	if err != nil {
		return nil, fmt.Errorf("codex search failed: %w", err)
	}

	// Map tool names back to tool objects
	toolMap := make(map[string]*tools.Tool)
	for _, tool := range s.tools {
		toolMap[tool.Name] = tool
	}

	results := make([]*tools.Tool, 0, len(toolNames))
	for _, name := range toolNames {
		if tool, ok := toolMap[name]; ok {
			results = append(results, tool)
		}
	}

	s.logger.Debug("Codex search results", "query", query, "requested", topK, "returned", len(results))

	return results, nil
}

// GetToolCount returns the number of tools indexed
func (s *CodexSearchStore) GetToolCount() int {
	return len(s.tools)
}
