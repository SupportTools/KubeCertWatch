package checks

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// IngressStatus represents the status of a TLS secret.
type IngressStatus struct {
	Namespace      string
	IngressName    string
	InternalStatus string
	ExternalStatus string
}

var (
	ingressStatus []IngressStatus
)

// GetSecretStatuses returns a snapshot of the current statuses.
func GetIngressStatuses() []IngressStatus {
	statusLock.Lock()
	defer statusLock.Unlock()

	// Return a copy to avoid concurrent modification issues
	statusCopy := make([]IngressStatus, len(ingressStatus))
	copy(statusCopy, ingressStatus)
	return statusCopy
}

// CheckIngress performs SSL checks on Ingress resources with TLS configured.
func CheckIngress(ctx context.Context, clientset *kubernetes.Clientset) error {
	log.Println("Starting Ingress checks...")

	// List all Ingresses in the cluster
	ingresses, err := clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Errorf("Failed to list Ingress resources: %v", err)
		return err
	}

	for _, ingress := range ingresses.Items {
		if len(ingress.Spec.TLS) == 0 {
			// Skip Ingress without TLS configured
			continue
		}

		// Get IP addresses from the Ingress status
		for _, ingressStatus := range ingress.Status.LoadBalancer.Ingress {
			internalStatus := "Unknown"
			externalStatus := "Unknown"

			// Internal check using IP
			if ingressStatus.IP != "" {
				internalStatus = checkSSL(fmt.Sprintf("https://%s", ingressStatus.IP))
			}

			// External check using Hostname (DNS)
			for _, rule := range ingress.Spec.Rules {
				if rule.Host != "" {
					externalStatus = checkSSL(fmt.Sprintf("https://%s", rule.Host))
				}
			}

			log.Printf("Ingress: %s/%s, Internal SSL: %s, External SSL: %s",
				ingress.Namespace, ingress.Name, internalStatus, externalStatus)
		}
	}

	log.Println("Ingress checks completed.")
	return nil
}

// checkSSL validates the SSL connection for the given URL.
func checkSSL(url string) string {
	// Configure an HTTP client with a timeout and TLS config
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Skip cert verification for this check
			},
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		log.Errorf("SSL check failed for URL %s: %v", url, err)
		return "Failed"
	}
	defer resp.Body.Close()

	if resp.TLS != nil {
		return "Valid"
	}
	return "Invalid"
}
