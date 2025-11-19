package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const (
	unisonDBURL = "http://localhost:8001/api/v1" // UnisonDB HTTP endpoint
	namespace   = "demo"
)

type WriteRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type WriteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Key     string `json:"key"`
}

type UnisonDBPutRequest struct {
	Value string `json:"value"` // base64-encoded
}

// Handler to write data to UnisonDB
func writeHandler(w http.ResponseWriter, r *http.Request) {
	var req WriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Key == "" || req.Value == "" {
		respondError(w, http.StatusBadRequest, "Key and value are required")
		return
	}

	// Write to UnisonDB
	if err := writeToUnisonDB(req.Key, req.Value); err != nil {
		log.Printf("Error writing to UnisonDB: %v", err)
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to write to UnisonDB: %v", err))
		return
	}

	log.Printf("[WRITER] Successfully wrote key=%s, value=%s", req.Key, req.Value)

	respondJSON(w, http.StatusOK, WriteResponse{
		Success: true,
		Message: "Data written to UnisonDB successfully",
		Key:     req.Key,
	})
}

// Write data to UnisonDB via HTTP API
func writeToUnisonDB(key, value string) error {
	// Encode value as base64
	encodedValue := base64.StdEncoding.EncodeToString([]byte(value))

	putReq := UnisonDBPutRequest{
		Value: encodedValue,
	}

	body, err := json.Marshal(putReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s/kv/%s", unisonDBURL, namespace, key)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"service": "writer-api",
	})
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func main() {
	router := mux.NewRouter()

	// Routes
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/write", writeHandler).Methods("POST")

	// CORS middleware for easy testing
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	port := "8080"
	log.Printf("🚀 Writer API starting on port %s", port)
	log.Printf("📝 POST /write - Write data to UnisonDB")
	log.Printf("❤️  GET /health - Health check")
	log.Printf("🔗 UnisonDB URL: %s", unisonDBURL)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
