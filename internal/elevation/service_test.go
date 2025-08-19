package elevation

import (
	"math"
	"sync"
	"testing"
)

func TestNewService(t *testing.T) {
	service, err := NewService("../../data/elevation.bin", "../../data/elevation.bin.header")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	
	if service == nil {
		t.Fatal("Service is nil")
	}
	
	if service.grid == nil {
		t.Fatal("Grid is nil")
	}
}

func TestGetElevation(t *testing.T) {
	service, err := NewService("../../data/elevation.bin", "../../data/elevation.bin.header")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	
	tests := []struct {
		name      string
		lat       float64
		lon       float64
		wantErr   bool
		checkElev bool
		minElev   float64
		maxElev   float64
	}{
		{
			name:      "Tokyo Station",
			lat:       35.6812,
			lon:       139.7671,
			wantErr:   false,
			checkElev: true,
			minElev:   0,
			maxElev:   100,
		},
		{
			name:      "Mt. Fuji",
			lat:       35.3606,
			lon:       138.7274,
			wantErr:   false,
			checkElev: true,
			minElev:   300,
			maxElev:   330,
		},
		{
			name:      "Osaka Castle",
			lat:       34.6873,
			lon:       135.5262,
			wantErr:   false,
			checkElev: true,
			minElev:   10,
			maxElev:   50,
		},
		{
			name:    "Out of bounds - north",
			lat:     50.0,
			lon:     140.0,
			wantErr: true,
		},
		{
			name:    "Out of bounds - south",
			lat:     15.0,
			lon:     140.0,
			wantErr: true,
		},
		{
			name:    "Out of bounds - east",
			lat:     35.0,
			lon:     160.0,
			wantErr: true,
		},
		{
			name:    "Out of bounds - west",
			lat:     35.0,
			lon:     120.0,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elev, err := service.GetElevation(tt.lat, tt.lon)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetElevation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.checkElev {
				if elev < tt.minElev || elev > tt.maxElev {
					t.Errorf("GetElevation() elevation = %v, want between %v and %v",
						elev, tt.minElev, tt.maxElev)
				}
			}
		})
	}
}

func TestGetBatchElevations(t *testing.T) {
	service, err := NewService("../../data/elevation.bin", "../../data/elevation.bin.header")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	
	points := []BatchPoint{
		{Lat: 35.6812, Lon: 139.7671},
		{Lat: 35.3606, Lon: 138.7274},
		{Lat: 34.6873, Lon: 135.5262},
		{Lat: 50.0, Lon: 140.0},
	}
	
	results, err := service.GetBatchElevations(points)
	if err != nil {
		t.Fatalf("GetBatchElevations() error = %v", err)
	}
	
	if len(results) != len(points) {
		t.Errorf("GetBatchElevations() returned %d results, want %d", len(results), len(points))
	}
	
	for i, result := range results {
		if result.Lat != points[i].Lat || result.Lon != points[i].Lon {
			t.Errorf("Result %d: coordinates mismatch", i)
		}
		
		if i == 3 && result.Elevation != -9999 {
			t.Errorf("Result %d: expected -9999 for out of bounds, got %v", i, result.Elevation)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	service, err := NewService("../../data/elevation.bin", "../../data/elevation.bin.header")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	
	var wg sync.WaitGroup
	numGoroutines := 100
	numRequests := 100
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < numRequests; j++ {
				lat := 35.0 + float64(id%10)*0.1
				lon := 139.0 + float64(j%10)*0.1
				
				_, err := service.GetElevation(lat, lon)
				if err != nil {
					t.Errorf("Goroutine %d request %d failed: %v", id, j, err)
				}
			}
		}(i)
	}
	
	wg.Wait()
}

func TestGetHealth(t *testing.T) {
	service, err := NewService("../../data/elevation.bin", "../../data/elevation.bin.header")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	
	health := service.GetHealth()
	
	if health.Status != "ok" {
		t.Errorf("Health status = %v, want ok", health.Status)
	}
	
	if health.MemoryMB < 0 {
		t.Errorf("Memory usage is negative: %d", health.MemoryMB)
	}
	
	if health.Goroutines < 1 {
		t.Errorf("Goroutine count is less than 1: %d", health.Goroutines)
	}
	
	if health.UptimeSeconds < 0 {
		t.Errorf("Uptime is negative: %f", health.UptimeSeconds)
	}
	
	initialRequests := health.TotalRequests
	
	service.GetElevation(35.0, 139.0)
	
	health2 := service.GetHealth()
	if health2.TotalRequests != initialRequests+1 {
		t.Errorf("Request counter not incremented: was %d, now %d, expected %d",
			initialRequests, health2.TotalRequests, initialRequests+1)
	}
}

func BenchmarkGetElevation(b *testing.B) {
	service, err := NewService("../../data/elevation.bin", "../../data/elevation.bin.header")
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		lat := 35.0 + float64(i%100)*0.01
		lon := 139.0 + float64(i%100)*0.01
		service.GetElevation(lat, lon)
	}
}

func BenchmarkGetElevationCached(b *testing.B) {
	service, err := NewService("../../data/elevation.bin", "../../data/elevation.bin.header")
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}
	
	service.GetElevation(35.6812, 139.7671)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		service.GetElevation(35.6812, 139.7671)
	}
}

func BenchmarkGetElevationBatch(b *testing.B) {
	service, err := NewService("../../data/elevation.bin", "../../data/elevation.bin.header")
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}
	
	points := make([]BatchPoint, 100)
	for i := range points {
		points[i] = BatchPoint{
			Lat: 35.0 + float64(i%50)*0.01,
			Lon: 139.0 + float64(i%50)*0.01,
		}
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		service.GetBatchElevations(points)
	}
}

func BenchmarkConcurrentRequests(b *testing.B) {
	service, err := NewService("../../data/elevation.bin", "../../data/elevation.bin.header")
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			lat := 35.0 + float64(i%100)*0.01
			lon := 139.0 + float64(i%100)*0.01
			service.GetElevation(lat, lon)
			i++
		}
	})
}

func TestInterpolation(t *testing.T) {
	grid := NewElevationGrid()
	grid.width = 100
	grid.height = 100
	grid.minLat = 35.0
	grid.maxLat = 36.0
	grid.minLon = 139.0
	grid.maxLon = 140.0
	grid.gridSize = 0.01
	grid.invGridSize = 1.0 / grid.gridSize
	grid.grid = make([]int16, grid.width*grid.height)
	
	for i := range grid.grid {
		grid.grid[i] = int16(i % 1000)
	}
	
	elev, err := grid.GetElevation(35.5, 139.5)
	if err != nil {
		t.Fatalf("GetElevation failed: %v", err)
	}
	
	if math.IsNaN(elev) || math.IsInf(elev, 0) {
		t.Errorf("Invalid elevation value: %v", elev)
	}
}

func TestEdgeCases(t *testing.T) {
	grid := NewElevationGrid()
	grid.width = 100
	grid.height = 100
	grid.minLat = 35.0
	grid.maxLat = 36.0
	grid.minLon = 139.0
	grid.maxLon = 140.0
	grid.gridSize = 0.01
	grid.invGridSize = 1.0 / grid.gridSize
	grid.grid = make([]int16, grid.width*grid.height)
	
	for i := range grid.grid {
		grid.grid[i] = 100
	}
	
	grid.grid[0] = -9999
	
	tests := []struct {
		name string
		lat  float64
		lon  float64
	}{
		{"Min corner", 35.0, 139.0},
		{"Max corner", 36.0, 140.0},
		{"Just inside min", 35.001, 139.001},
		{"Just inside max", 35.999, 139.999},
		{"Center", 35.5, 139.5},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := grid.GetElevation(tt.lat, tt.lon)
			if err != nil {
				t.Errorf("GetElevation(%v, %v) failed: %v", tt.lat, tt.lon, err)
			}
		})
	}
}