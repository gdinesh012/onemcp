package vectorstore

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// GloVeEmbedder implements embeddings using pre-trained GloVe vectors
type GloVeEmbedder struct {
	vectors map[string][]float32
	dim     int
	logger  *slog.Logger
}

// GloVe model configurations
var gloveModels = map[string]struct {
	url      string
	filename string
	dim      int
}{
	"6B.50d":  {"http://nlp.stanford.edu/data/glove.6B.zip", "glove.6B.50d.txt", 50},
	"6B.100d": {"http://nlp.stanford.edu/data/glove.6B.zip", "glove.6B.100d.txt", 100},
	"6B.200d": {"http://nlp.stanford.edu/data/glove.6B.zip", "glove.6B.200d.txt", 200},
	"6B.300d": {"http://nlp.stanford.edu/data/glove.6B.zip", "glove.6B.300d.txt", 300},
}

// NewGloVeEmbedder creates a new GloVe embedder
// modelName: "6B.50d", "6B.100d", "6B.200d", or "6B.300d"
// cacheDir: directory to store downloaded models (e.g., "/tmp/glove")
func NewGloVeEmbedder(modelName string, cacheDir string, logger *slog.Logger) (*GloVeEmbedder, error) {
	modelConfig, ok := gloveModels[modelName]
	if !ok {
		return nil, fmt.Errorf("unknown GloVe model: %s (available: 6B.50d, 6B.100d, 6B.200d, 6B.300d)", modelName)
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Check if model file already exists
	modelPath := filepath.Join(cacheDir, modelConfig.filename)

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		logger.Info("GloVe model not found, downloading...", "model", modelName, "path", modelPath)

		// Download and extract
		if err := downloadAndExtractGloVe(modelConfig.url, modelConfig.filename, cacheDir, logger); err != nil {
			return nil, fmt.Errorf("failed to download GloVe model: %w", err)
		}

		logger.Info("GloVe model downloaded successfully", "model", modelName)
	} else {
		logger.Info("Using cached GloVe model", "model", modelName, "path", modelPath)
	}

	// Load vectors from file
	vectors, err := loadGloVeVectors(modelPath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load GloVe vectors: %w", err)
	}

	logger.Info("GloVe embedder ready", "model", modelName, "vocabulary_size", len(vectors), "dimension", modelConfig.dim)

	return &GloVeEmbedder{
		vectors: vectors,
		dim:     modelConfig.dim,
		logger:  logger,
	}, nil
}

// downloadAndExtractGloVe downloads and extracts the GloVe model
func downloadAndExtractGloVe(url, targetFile, cacheDir string, logger *slog.Logger) error {
	// Download zip file
	zipPath := filepath.Join(cacheDir, "glove.zip")

	logger.Info("Downloading GloVe model...", "url", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create zip file
	out, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy with progress (simple version)
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	logger.Info("Download complete", "size_mb", written/(1024*1024))

	// Extract the specific file we need
	logger.Info("Extracting model file...", "file", targetFile)

	if err := extractFileFromZip(zipPath, targetFile, cacheDir); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	// Clean up zip file
	os.Remove(zipPath)

	logger.Info("Extraction complete")

	return nil
}

// extractFileFromZip extracts a specific file from a zip archive
func extractFileFromZip(zipPath, targetFile, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == targetFile {
			// Open file in zip
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			// Create destination file
			destPath := filepath.Join(destDir, f.Name)
			outFile, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			// Copy content
			_, err = io.Copy(outFile, rc)
			return err
		}
	}

	return fmt.Errorf("file %s not found in zip", targetFile)
}

// loadGloVeVectors loads GloVe vectors from a text file
func loadGloVeVectors(path string, logger *slog.Logger) (map[string][]float32, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	vectors := make(map[string][]float32)
	scanner := bufio.NewScanner(file)

	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount%10000 == 0 {
			logger.Info("Loading GloVe vectors...", "loaded", lineCount)
		}

		line := scanner.Text()
		parts := strings.Fields(line)

		if len(parts) < 2 {
			continue // Skip malformed lines
		}

		word := parts[0]
		vec := make([]float32, len(parts)-1)

		for i, s := range parts[1:] {
			val, err := strconv.ParseFloat(s, 32)
			if err != nil {
				continue // Skip if parsing fails
			}
			vec[i] = float32(val)
		}

		vectors[word] = vec
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	logger.Info("GloVe vectors loaded", "total_words", len(vectors))

	return vectors, nil
}

// Generate creates an embedding by averaging word vectors
func (e *GloVeEmbedder) Generate(text string) ([]float32, error) {
	words := e.tokenize(text)
	if len(words) == 0 {
		return make([]float32, e.dim), nil
	}

	// Average word vectors
	embedding := make([]float32, e.dim)
	count := 0

	for _, word := range words {
		if vec, ok := e.vectors[word]; ok {
			for i := 0; i < e.dim; i++ {
				embedding[i] += vec[i]
			}
			count++
		}
	}

	// Average
	if count > 0 {
		for i := range embedding {
			embedding[i] /= float32(count)
		}
	}

	return normalize(embedding), nil
}

// tokenize splits text into lowercase words
func (e *GloVeEmbedder) tokenize(text string) []string {
	text = strings.ToLower(text)
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})

	// Filter out very short words and stop words
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true,
		"at": true, "be": true, "by": true, "for": true, "from": true,
		"has": true, "he": true, "in": true, "is": true, "it": true,
		"its": true, "of": true, "on": true, "that": true, "the": true,
		"this": true, "to": true, "was": true, "will": true, "with": true,
	}

	filtered := make([]string, 0, len(words))
	for _, word := range words {
		if len(word) > 1 && !stopWords[word] {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

// Dimension returns the embedding dimension
func (e *GloVeEmbedder) Dimension() int {
	return e.dim
}

// GetVocabularySize returns the number of words in the vocabulary
func (e *GloVeEmbedder) GetVocabularySize() int {
	return len(e.vectors)
}
