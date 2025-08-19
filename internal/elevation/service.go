package elevation

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

type Service struct {
	grid      *ElevationGrid
	startTime time.Time
	requests  uint64
	mu        sync.RWMutex
}

func NewService(dataPath, headerPath string) (*Service, error) {
	grid := NewElevationGrid()
	if err := grid.LoadFromFile(dataPath, headerPath); err != nil {
		return nil, fmt.Errorf("failed to load elevation data: %w", err)
	}

	return &Service{
		grid:      grid,
		startTime: time.Now(),
	}, nil
}

func (s *Service) GetElevation(lat, lon float64) (float64, error) {
	s.mu.Lock()
	s.requests++
	s.mu.Unlock()

	return s.grid.GetElevation(lat, lon)
}

type BatchPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type ElevationResult struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Elevation float64 `json:"elevation"`
}

func (s *Service) GetBatchElevations(points []BatchPoint) ([]ElevationResult, error) {
	s.mu.Lock()
	s.requests += uint64(len(points))
	s.mu.Unlock()

	results := make([]ElevationResult, len(points))
	
	convertedPoints := make([]struct{ Lat, Lon float64 }, len(points))
	for i, p := range points {
		convertedPoints[i].Lat = p.Lat
		convertedPoints[i].Lon = p.Lon
	}

	elevations, err := s.grid.GetBatchElevations(convertedPoints)
	if err != nil {
		return nil, err
	}

	for i, p := range points {
		results[i] = ElevationResult{
			Lat:       p.Lat,
			Lon:       p.Lon,
			Elevation: elevations[i],
		}
	}

	return results, nil
}

type HealthStatus struct {
	Status         string  `json:"status"`
	MemoryMB       int     `json:"memory_mb"`
	Goroutines     int     `json:"goroutines"`
	UptimeSeconds  float64 `json:"uptime_seconds"`
	TotalRequests  uint64  `json:"total_requests"`
}

func (s *Service) GetHealth() HealthStatus {
	s.mu.RLock()
	requests := s.requests
	s.mu.RUnlock()

	return HealthStatus{
		Status:         "ok",
		MemoryMB:       getMemoryUsageMB(),
		Goroutines:     getGoroutineCount(),
		UptimeSeconds:  time.Since(s.startTime).Seconds(),
		TotalRequests:  requests,
	}
}

func getMemoryUsageMB() int {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int(m.Alloc / 1024 / 1024)
}

func getGoroutineCount() int {
	return runtime.NumGoroutine()
}