package k8s

import (
	"github.com/supporttools/KubeCertWatch/pkg/config"
	"github.com/supporttools/KubeCertWatch/pkg/logging"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var log = logging.SetupLogging()

// ConnectToK8s connects to a Kubernetes cluster by checking the environment and configuration settings.
func ConnectToK8s() (*kubernetes.Clientset, error) {
	var kubeConfig *rest.Config
	var err error

	log.Debug("Attempting to connect using in-cluster configuration...")
	// Attempt to connect using in-cluster configuration
	kubeConfig, err = rest.InClusterConfig()
	if err == nil {
		log.Info("Successfully obtained in-cluster configuration.")
		clientset, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			log.Errorf("Failed to create Kubernetes client from in-cluster config: %v", err)
			return nil, err
		}
		log.Debug("Successfully created Kubernetes client using in-cluster configuration.")
		return clientset, nil
	}
	log.Warnf("In-cluster configuration failed: %v. Attempting to use KUBECONFIG.", err)

	// Attempt to connect using KUBECONFIG environment variable
	cfgKubeConfig := config.CFG.KubeConfig
	if cfgKubeConfig != "" {
		log.Debugf("KUBECONFIG environment variable is set: %s", cfgKubeConfig)
		log.Debug("Attempting to build configuration from KUBECONFIG...")
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", cfgKubeConfig)
		if err == nil {
			log.Info("Successfully loaded configuration from KUBECONFIG.")
			clientset, err := kubernetes.NewForConfig(kubeConfig)
			if err != nil {
				log.Errorf("Failed to create Kubernetes client from KUBECONFIG: %v", err)
				return nil, err
			}
			log.Debug("Successfully created Kubernetes client using KUBECONFIG.")
			return clientset, nil
		}
		log.Errorf("Failed to load configuration from KUBECONFIG (%s): %v", cfgKubeConfig, err)
	} else {
		log.Debug("KUBECONFIG environment variable is not set.")
	}

	// All connection attempts failed
	log.Error("All attempts to configure Kubernetes client failed. Ensure the environment or KUBECONFIG is set correctly.")
	return nil, err
}
