package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/yakkun/gsi-elevation-api/internal/elevation"
	"github.com/yakkun/gsi-elevation-api/internal/handler"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port            string        `yaml:"port"`
		ReadTimeout     time.Duration `yaml:"read_timeout"`
		WriteTimeout    time.Duration `yaml:"write_timeout"`
		MaxHeaderBytes  int           `yaml:"max_header_bytes"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	} `yaml:"server"`
	Data struct {
		DataPath   string `yaml:"data_path"`
		HeaderPath string `yaml:"header_path"`
	} `yaml:"data"`
	Performance struct {
		GOMAXPROCS int `yaml:"gomaxprocs"`
	} `yaml:"performance"`
}

func loadConfig(path string) (*Config, error) {
	config := &Config{}
	
	config.Server.Port = "8080"
	config.Server.ReadTimeout = 10 * time.Second
	config.Server.WriteTimeout = 10 * time.Second
	config.Server.MaxHeaderBytes = 1 << 20
	config.Server.ShutdownTimeout = 30 * time.Second
	config.Data.DataPath = "data/elevation.bin"
	config.Data.HeaderPath = "data/elevation.bin.header"
	config.Performance.GOMAXPROCS = runtime.NumCPU()

	if port := os.Getenv("PORT"); port != "" {
		config.Server.Port = port
	}

	if path == "" {
		return config, nil
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}

	if port := os.Getenv("PORT"); port != "" {
		config.Server.Port = port
	}

	return config, nil
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config/config.yaml", "path to config file")
	flag.Parse()

	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	runtime.GOMAXPROCS(config.Performance.GOMAXPROCS)

	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	baseDir := filepath.Dir(execPath)
	
	dataPath := config.Data.DataPath
	if !filepath.IsAbs(dataPath) {
		dataPath = filepath.Join(baseDir, dataPath)
	}
	
	headerPath := config.Data.HeaderPath
	if !filepath.IsAbs(headerPath) {
		headerPath = filepath.Join(baseDir, headerPath)
	}

	log.Printf("Starting elevation API server...")
	log.Printf("Loading elevation data from %s", dataPath)
	
	service, err := elevation.NewService(dataPath, headerPath)
	if err != nil {
		log.Fatalf("Failed to initialize elevation service: %v", err)
	}

	h := handler.NewHandler(service)
	
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	loggingMux := loggingMiddleware(mux)

	server := &http.Server{
		Addr:           ":" + config.Server.Port,
		Handler:        loggingMux,
		ReadTimeout:    config.Server.ReadTimeout,
		WriteTimeout:   config.Server.WriteTimeout,
		MaxHeaderBytes: config.Server.MaxHeaderBytes,
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			health := service.GetHealth()
			log.Printf("METRICS: status=%s memory_mb=%d goroutines=%d uptime_s=%.0f requests=%d",
				health.Status, health.MemoryMB, health.Goroutines, 
				health.UptimeSeconds, health.TotalRequests)
		}
	}()

	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), config.Server.ShutdownTimeout)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v", err)
		}
		close(done)
	}()

	log.Printf("Server is ready to handle requests at :%s", config.Server.Port)
	log.Printf("GOMAXPROCS set to %d", runtime.GOMAXPROCS(0))
	
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on :%s: %v", config.Server.Port, err)
	}

	<-done
	log.Println("Server stopped")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		lw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(lw, r)
		
		duration := time.Since(start)
		
		log.Printf(`{"method":"%s","path":"%s","status":%d,"duration_ms":%.2f,"remote_addr":"%s"}`,
			r.Method, r.URL.Path, lw.statusCode, float64(duration.Microseconds())/1000.0, r.RemoteAddr)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.ResponseWriter.WriteHeader(code)
}