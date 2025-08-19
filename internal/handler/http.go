package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/yakkun/gsi-elevation-api/internal/elevation"
)

type Handler struct {
	service *elevation.Service
}

func NewHandler(service *elevation.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/elevation", h.handleElevation)
	mux.HandleFunc("/elevation/batch", h.handleBatchElevation)
	mux.HandleFunc("/health", h.handleHealth)
}

func (h *Handler) handleElevation(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")

	if latStr == "" || lonStr == "" {
		http.Error(w, "Missing lat or lon parameter", http.StatusBadRequest)
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		http.Error(w, "Invalid lat parameter", http.StatusBadRequest)
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		http.Error(w, "Invalid lon parameter", http.StatusBadRequest)
		return
	}

	elev, err := h.service.GetElevation(lat, lon)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}

	response := elevation.ElevationResult{
		Lat:       lat,
		Lon:       lon,
		Elevation: elev,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("GET /elevation lat=%.4f lon=%.4f elevation=%.1f time=%v",
		lat, lon, elev, time.Since(start))
}

func (h *Handler) handleBatchElevation(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Points []elevation.BatchPoint `json:"points"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(request.Points) == 0 {
		http.Error(w, "No points provided", http.StatusBadRequest)
		return
	}

	if len(request.Points) > 1000 {
		http.Error(w, "Too many points (max 1000)", http.StatusBadRequest)
		return
	}

	results, err := h.service.GetBatchElevations(request.Points)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		Results []elevation.ElevationResult `json:"results"`
	}{
		Results: results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("POST /elevation/batch points=%d time=%v",
		len(request.Points), time.Since(start))
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := h.service.GetHealth()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}