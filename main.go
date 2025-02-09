package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/supporttools/KubeCertWatch/pkg/adminServer"
	"github.com/supporttools/KubeCertWatch/pkg/checks"
	"github.com/supporttools/KubeCertWatch/pkg/config"
	"github.com/supporttools/KubeCertWatch/pkg/k8s"
	"github.com/supporttools/KubeCertWatch/pkg/logging"
	"github.com/supporttools/KubeCertWatch/pkg/metrics"
	"github.com/robfig/cron/v3"
	"k8s.io/client-go/kubernetes"
)

var (
	logger            = logging.SetupLogging()
	taskLock          sync.Mutex
	isTaskRunning     bool
	lastSuccessfulRun time.Time
	clientset         *kubernetes.Clientset
)

// withRetry implements retry logic for transient failures
func withRetry(ctx context.Context, fn func() error) error {
	backoff := time.Second
	for attempts := 0; attempts < 3; attempts++ {
		if err := fn(); err != nil {
			if attempts == 2 {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
				continue
			}
		}
		return nil
	}
	return nil
}

// isHealthy returns the health status of the service
func isHealthy() bool {
	taskLock.Lock()
	defer taskLock.Unlock()
	return !isTaskRunning || time.Since(lastSuccessfulRun) < time.Hour
}

func main() {
	logger.Println("Starting KubeCertWatch...")

	// Load and validate configuration
	logger.Println("Loading configuration...")
	config.LoadConfiguration()
	if err := config.ValidateRequiredConfig(); err != nil {
		logger.Fatalf("Configuration validation failed: %v", err)
	}
	logger.Println("Configuration loaded and validated successfully.")

	// Connect to Kubernetes
	logger.Println("Connecting to Kubernetes...")
	var err error
	clientset, err = k8s.ConnectToK8s()
	if err != nil {
		logger.Fatalf("Failed to connect to Kubernetes: %v", err)
	}
	logger.Println("Connected to Kubernetes successfully.")

	// Start HTTP server
	logger.Println("Starting HTTP server...")
	server := adminServer.StartHTTPServer(clientset)

	// Setup Cron Scheduler
	logger.Println("Setting up cron scheduler...")
	c := cron.New()
	_, err = c.AddFunc(config.CFG.CronSchedule, func() {
		logger.Println("Running scheduled tasks...")
		runChecks(context.Background())
	})
	if err != nil {
		logger.Fatalf("Failed to schedule cron job: %v", err)
	}
	c.Start()

	// Graceful Shutdown
	logger.Println("Setting up signal handling for graceful shutdown...")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	logger.Println("Received shutdown signal. Shutting down gracefully...")
	c.Stop()
	if err := server.Shutdown(context.Background()); err != nil {
		logger.Printf("Error during server shutdown: %v", err)
	}
	logger.Println("Shutdown complete.")
}

// runChecks performs all scheduled checks
func runChecks(ctx context.Context) {
	taskLock.Lock()
	if isTaskRunning {
		logger.Println("Checks already running. Skipping this cycle.")
		taskLock.Unlock()
		return
	}
	isTaskRunning = true
	taskLock.Unlock()

	defer func() {
		taskLock.Lock()
		isTaskRunning = false
		taskLock.Unlock()
	}()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Run checks in parallel
	wg.Add(2)
	
	// TLS secret checks
	go func() {
		defer wg.Done()
		logger.Println("Running TLS secret checks...")
		err := withRetry(ctx, func() error {
			return checks.CheckTLSSecrets(ctx, clientset)
		})
		if err != nil {
			logger.Errorf("Failed to check TLS secrets: %v", err)
			metrics.ErrorCounter.WithLabelValues("tls-secrets", "check_error").Inc()
			errChan <- fmt.Errorf("TLS secrets check: %w", err)
		}
		metrics.LastCheckTime.WithLabelValues("tls-secret-check").SetToCurrentTime()
	}()

	// cert-manager checks
	go func() {
		defer wg.Done()
		logger.Println("Running cert-manager certificate checks...")
		err := withRetry(ctx, func() error {
			return checks.CheckCertManagerCertificates(ctx, config.CFG.KubeConfig)
		})
		if err != nil {
			logger.Errorf("Failed to check cert-manager certificates: %v", err)
			metrics.ErrorCounter.WithLabelValues("cert-manager", "check_error").Inc()
			errChan <- fmt.Errorf("cert-manager check: %w", err)
		}
		metrics.LastCheckTime.WithLabelValues("cert-manager-check").SetToCurrentTime()
	}()

	// Wait for all checks to complete
	wg.Wait()
	close(errChan)

	// Process any errors
	errCount := 0
	for err := range errChan {
		errCount++
		logger.Error(err)
	}

	if errCount == 0 {
		taskLock.Lock()
		lastSuccessfulRun = time.Now()
		taskLock.Unlock()
		logger.Println("Certificate checks completed successfully.")
	} else {
		logger.Printf("Certificate checks completed with %d errors.", errCount)
	}
}
