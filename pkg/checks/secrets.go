package checks

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"sync"
	"time"

	"github.com/supporttools/KubeCertWatch/pkg/logging"
	"github.com/supporttools/KubeCertWatch/pkg/metrics"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var log = logging.SetupLogging()

// SecretStatus represents the status of a TLS secret.
type SecretStatus struct {
	Namespace      string
	SecretName     string
	ExpirationDate string
	DaysUntil      int
	Status         string
}

var (
	secretStatuses []SecretStatus
	statusLock     sync.Mutex
)

// GetSecretStatuses returns a snapshot of the current statuses.
func GetSecretStatuses() []SecretStatus {
	statusLock.Lock()
	defer statusLock.Unlock()

	// Return a copy to avoid concurrent modification issues
	statusCopy := make([]SecretStatus, len(secretStatuses))
	copy(statusCopy, secretStatuses)
	return statusCopy
}

// CheckTLSSecrets scans all secrets in the cluster for TLS secrets and checks their expiration dates.
func CheckTLSSecrets(ctx context.Context, clientset *kubernetes.Clientset) error {
	log.Debug("Acquiring lock for secretStatuses")
	statusLock.Lock()
	defer statusLock.Unlock()

	log.Debug("Clearing previous secret statuses")
	secretStatuses = []SecretStatus{} // Clear previous statuses

	log.Debug("Listing all secrets in the cluster")
	secrets, err := clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Errorf("Failed to list secrets: %v", err)
		return err
	}
	log.Debugf("Found %d secrets in the cluster", len(secrets.Items))

	for _, secret := range secrets.Items {
		log.Debugf("Processing secret: %s/%s", secret.Namespace, secret.Name)
		if secret.Type == v1.SecretTypeTLS {
			log.Debugf("Secret %s/%s is of type TLS", secret.Namespace, secret.Name)

			expirationDate := "unknown"
			daysUntil := 0
			status := "valid"

			certPEM, ok := secret.Data["tls.crt"]
			if ok {
				log.Debugf("Found tls.crt in secret %s/%s. Parsing certificate...", secret.Namespace, secret.Name)
				expiration, err := getCertificateExpiration(certPEM)
				if err != nil {
					log.Errorf("Failed to parse certificate in secret %s/%s: %v", secret.Namespace, secret.Name, err)
					status = "error parsing cert"
				} else {
					expirationDate = expiration.Format("2006-01-02")
					daysUntil = int(time.Until(expiration).Hours() / 24)
					log.Debugf("Certificate in secret %s/%s expires on %s (in %d days)", secret.Namespace, secret.Name, expirationDate, daysUntil)

					if time.Now().After(expiration) {
						status = "expired"
						daysUntil = int(time.Since(expiration).Hours()/24) * -1 // Negative days since expiration
						log.Warnf("Certificate in secret %s/%s is expired by %d days", secret.Namespace, secret.Name, -daysUntil)
					} else if daysUntil < 7 {
						status = "expiring soon"
						log.Warnf("Certificate in secret %s/%s is expiring soon", secret.Namespace, secret.Name)
					}
				}
			} else {
				log.Warnf("Secret %s/%s is missing tls.crt", secret.Namespace, secret.Name)
				status = "missing cert"
			}

			// Update certificate expiry metrics
			if status == "valid" || status == "expiring soon" {
				metrics.CertificateExpiryDays.WithLabelValues(secret.Namespace, secret.Name).Set(float64(daysUntil))
			}

			// Add to status list
			secretStatuses = append(secretStatuses, SecretStatus{
				Namespace:      secret.Namespace,
				SecretName:     secret.Name,
				ExpirationDate: expirationDate,
				DaysUntil:      daysUntil,
				Status:         status,
			})
			log.Debugf("Added status for secret %s/%s: %v", secret.Namespace, secret.Name, status)
		} else {
			log.Debugf("Secret %s/%s is not of type TLS. Skipping.", secret.Namespace, secret.Name)
		}
	}

	log.Debug("Completed processing all secrets")
	return nil
}

// getCertificateExpiration parses the PEM-encoded certificate and returns the expiration date.
func getCertificateExpiration(certPEM []byte) (time.Time, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return time.Time{}, errors.New("failed to decode PEM block containing certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, err
	}

	return cert.NotAfter, nil
}
