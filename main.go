package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
)

const (
	maxFileSize = 10 << 20 // 10 MB
)

type WordFrequency struct {
	Word      string `json:"word"`
	Frequency int    `json:"frequency"`
}

type Analysis struct {
	ID          string          `json:"id"`
	Frequencies []WordFrequency `json:"frequencies"`
}

type Store struct {
	mu        sync.RWMutex
	analyses  map[string]Analysis
}

func NewStore() *Store {
	return &Store{
		analyses: make(map[string]Analysis),
	}
}

var (
	store = NewStore()
	wordRegex = regexp.MustCompile(`\b[\p{L}]+\b`)
)

func main() {
	mux := http.NewServeMux()

	// Register routes with the new ServeMux pattern matching
	mux.HandleFunc("POST /api/upload", handleUpload)
	mux.HandleFunc("GET /api/analysis/{id}", handleGetAnalysis)

	log.Printf("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form with size limit
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	if !isValidContentType(contentType) {
		http.Error(w, "Invalid file type. Only text/plain and application/pdf are supported", http.StatusBadRequest)
		return
	}

	// Read and process file content
	var text string
	if contentType == "application/pdf" {
		text, err = extractPDFText(file)
	} else {
		content, err := io.ReadAll(file)
		if err == nil {
			text = string(content)
		}
	}

	if err != nil {
		http.Error(w, "Error reading file content", http.StatusInternalServerError)
		return
	}

	// Process text and count words
	frequencies := processText(text)

	// Generate UUID and store analysis
	analysisID := uuid.New().String()
	analysis := Analysis{
		ID:          analysisID,
		Frequencies: frequencies,
	}

	store.mu.Lock()
	store.analyses[analysisID] = analysis
	store.mu.Unlock()

	// Return analysis ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id": analysisID,
	})
}

func handleGetAnalysis(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	store.mu.RLock()
	analysis, exists := store.analyses[id]
	store.mu.RUnlock()

	if !exists {
		http.Error(w, "Analysis not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

func isValidContentType(contentType string) bool {
	return contentType == "text/plain" || contentType == "application/pdf"
}

func extractPDFText(file io.Reader) (string, error) {
	// Create temporary file to store PDF content
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("error reading PDF file: %v", err)
	}

	// Read PDF content
	reader, err := pdf.NewReader(content)
	if err != nil {
		return "", fmt.Errorf("error creating PDF reader: %v", err)
	}

	var text strings.Builder
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		pageText, err := page.GetPlainText()
		if err != nil {
			continue
		}
		text.WriteString(pageText)
	}

	return text.String(), nil
}

func processText(text string) []WordFrequency {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Extract words using regex
	words := wordRegex.FindAllString(text, -1)

	// Count frequencies
	frequencies := make(map[string]int)
	for _, word := range words {
		frequencies[word]++
	}

	// Convert to slice and sort
	var result []WordFrequency
	for word, freq := range frequencies {
		result = append(result, WordFrequency{
			Word:      word,
			Frequency: freq,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Frequency > result[j].Frequency
	})

	return result
}