package adminServer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/supporttools/KubeCertWatch/pkg/checks"
	"github.com/supporttools/KubeCertWatch/pkg/config"
	"github.com/supporttools/KubeCertWatch/pkg/logging"
	"github.com/supporttools/KubeCertWatch/pkg/pages"
	"github.com/supporttools/KubeCertWatch/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
)

var (
	log           = logging.SetupLogging()
	taskLock      sync.Mutex
	isTaskRunning bool
)

// StartHTTPServer starts an HTTP server for metrics and admin endpoints
func StartHTTPServer(clientset *kubernetes.Clientset) *http.Server {
	log.Println("Setting up HTTP server...")
	mux := http.NewServeMux()

	// Register routes
	registerRoutes(mux, clientset)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.CFG.MetricsPort),
		Handler:      logRequestMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		log.Printf("HTTP server running on port %d", config.CFG.MetricsPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()
	return server
}

// registerRoutes registers all HTTP routes
func registerRoutes(mux *http.ServeMux, clientset *kubernetes.Clientset) {
	mux.HandleFunc("/", pages.DefaultPage)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", healthCheck)
	mux.HandleFunc("/version", versionInfo)

	// TLS Secret Check Handler
	mux.HandleFunc("/check/secrets", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("HTTP request to /check/secrets from %s", r.RemoteAddr)
		if !triggerTask("secrets", clientset) {
			http.Error(w, "Task already running", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, "TLS secrets check initiated.")
	})

	// Cert-Manager Check Handler
	mux.HandleFunc("/check/cert-manager", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("HTTP request to /check/cert-manager from %s", r.RemoteAddr)
		if !triggerTask("cert-manager", clientset) {
			http.Error(w, "Task already running", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, "cert-manager check initiated.")
	})

	// Ingress Check Handler
	mux.HandleFunc("/check/ingress", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("HTTP request to /check/ingress from %s", r.RemoteAddr)
		if !triggerTask("ingress", clientset) {
			http.Error(w, "Task already running", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, "Ingress check initiated.")
	})

	// Status Pages
	mux.HandleFunc("/status/secrets", pages.SecretsStatusPage)
	mux.HandleFunc("/status/cert-manager", pages.CertManagerStatusPage)
	mux.HandleFunc("/status/ingress", pages.IngressStatusPage)
}

// healthCheck returns a JSON response indicating system health
func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	if err != nil {
		log.Printf("Failed to write health check response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// versionInfo returns the application's version information
func versionInfo(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]string{
		"version":   version.Version,
		"gitCommit": version.GitCommit,
		"buildTime": version.BuildTime,
	})
	if err != nil {
		log.Printf("Failed to encode version info: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// logRequestMiddleware logs incoming HTTP requests
func logRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Incoming request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// triggerTask ensures only one task runs at a time
func triggerTask(task string, clientset *kubernetes.Clientset) bool {
	taskLock.Lock()
	defer taskLock.Unlock()

	if isTaskRunning {
		log.Printf("Task already running; skipping %s request.", task)
		return false
	}

	isTaskRunning = true
	go func() {
		defer func() {
			taskLock.Lock()
			isTaskRunning = false
			taskLock.Unlock()
			log.Printf("Task %s completed.", task)
		}()
		runTask(task, clientset)
	}()

	return true
}

// runTask executes the specified task
func runTask(task string, clientset *kubernetes.Clientset) {
	switch task {
	case "secrets":
		log.Println("Starting TLS secret check...")
		err := checks.CheckTLSSecrets(context.Background(), clientset)
		if err != nil {
			log.Errorf("Error during TLS secret check: %v", err)
		}
	case "cert-manager":
		log.Println("Starting cert-manager certificate check...")
		err := checks.CheckCertManagerCertificates(context.Background(), config.CFG.KubeConfig)
		if err != nil {
			log.Errorf("Error during cert-manager certificate check: %v", err)
		}
	case "ingress":
		log.Println("Starting Ingress check...")
		err := checks.CheckIngress(context.Background(), clientset)
		if err != nil {
			log.Errorf("Error during Ingress check: %v", err)
		}
	default:
		log.Printf("Invalid task specified: %s", task)
	}
}
