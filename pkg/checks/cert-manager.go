package checks

import (
	"context"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	certManagerStatuses []CertManagerStatus
)

// CertManagerStatus represents the status of a cert-manager certificate
type CertManagerStatus struct {
	Namespace      string
	Certificate    string
	RenewalFailure string
	Status         string
}

// GetCertManagerStatuses returns the current statuses of cert-manager certificates
func GetCertManagerStatuses() []CertManagerStatus {
	statusLock.Lock()
	defer statusLock.Unlock()
	return append([]CertManagerStatus(nil), certManagerStatuses...) // Return a copy to avoid race conditions
}

// CheckCertManagerCertificates scans for certificates managed by cert-manager and checks their renewal status.
func CheckCertManagerCertificates(ctx context.Context, kubeConfigPath string) error {
	var kubeConfig *rest.Config
	var err error

	// Connect to the cluster
	if kubeConfigPath != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	} else {
		kubeConfig, err = rest.InClusterConfig()
	}

	if err != nil {
		log.Errorf("Failed to configure Kubernetes client: %v", err)
		return err
	}

	// Create a dynamic client
	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		log.Errorf("Failed to create dynamic client: %v", err)
		return err
	}

	// Define the GVR (GroupVersionResource) for Certificates
	certificateGVR := certmanagerv1.SchemeGroupVersion.WithResource("certificates")

	// List all Certificate resources in all namespaces
	certList, err := dynamicClient.Resource(certificateGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Errorf("Failed to list cert-manager Certificates: %v", err)
		return err
	}

	certManagerStatuses = []CertManagerStatus{} // Clear previous statuses

	// Iterate through Certificates and check conditions
	for _, cert := range certList.Items {
		status := "valid"
		renewalFailure := ""

		// Decode Certificate into its structured form
		var certObj certmanagerv1.Certificate
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(cert.UnstructuredContent(), &certObj); err != nil {
			log.Errorf("Failed to convert Certificate: %v", err)
			continue
		}

		for _, condition := range certObj.Status.Conditions {
			if condition.Type == certmanagerv1.CertificateConditionReady &&
				metav1.ConditionStatus(condition.Status) != metav1.ConditionTrue {
				status = "not ready"
				renewalFailure = condition.Reason
			}
		}

		certManagerStatuses = append(certManagerStatuses, CertManagerStatus{
			Namespace:      certObj.Namespace,
			Certificate:    certObj.Name,
			RenewalFailure: renewalFailure,
			Status:         status,
		})
	}

	return nil
}
