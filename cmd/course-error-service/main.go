package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"example.com/course-delivery-errors/internal/edtech"
	"example.com/course-delivery-errors/internal/infrai"
)

type captureClient interface {
	Capture(context.Context, edtech.Capture, string) error
}

type server struct {
	captures captureClient
	now      func() time.Time
}

func (s server) deliveryError(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var failure edtech.Failure
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&failure); err != nil {
		http.Error(w, "invalid delivery failure", http.StatusBadRequest)
		return
	}
	capture, decision, err := edtech.Classify(s.now().UTC(), failure)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeKey := fmt.Sprintf("delivery:%s:%s:%s", failure.CourseID, failure.LearnerID, failure.DeliveryStage)
	if err := s.captures.Capture(r.Context(), capture, writeKey); err != nil {
		var apiErr *infrai.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
			http.Error(w, apiErr.Error(), apiErr.HTTPStatus)
			return
		}
		http.Error(w, "capture unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(decision)
}

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/delivery-errors", server{captures: infrai.NewClient(apiKey), now: time.Now}.deliveryError)
	log.Println("course error service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
