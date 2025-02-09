// pkg/config/config.go
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/robfig/cron/v3"
)

// AppConfig structure for environment-based configurations.
type AppConfig struct {
	Debug        bool   `json:"debug"`
	MetricsPort  int    `json:"metricsPort"`
	CronSchedule string `json:"cronSchedule"`
	ClusterName  string `json:"clusterName"`
	KubeConfig   string `json:"kubeConfig"`
}

// CFG is the global configuration object.
var CFG AppConfig

// LoadConfiguration loads configuration from environment variables.
func LoadConfiguration() {
	CFG.Debug = parseEnvBool("DEBUG", false)
	CFG.MetricsPort = parseEnvInt("METRICS_PORT", 9990)
	CFG.CronSchedule = getEnvOrDefault("CRON_SCHEDULE", "0 */12 * * *")
	CFG.ClusterName = getEnvOrDefault("CLUSTER_NAME", "")
	CFG.KubeConfig = getEnvOrDefault("KUBECONFIG", "")

	if CFG.Debug {
		log.Printf("Configuration Loaded: %+v\n", CFG)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Environment variable %s not set. Using default: %s", key, defaultValue)
	return defaultValue
}

func parseEnvInt(key string, defaultValue int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Printf("Environment variable %s not set. Using default: %d", key, defaultValue)
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Error parsing %s as int: %v. Using default value: %d", key, err, defaultValue)
		return defaultValue
	}
	return intValue
}

func parseEnvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Printf("Environment variable %s not set. Using default: %t", key, defaultValue)
		return defaultValue
	}
	value = strings.ToLower(value)

	// Handle additional truthy and falsy values
	switch value {
	case "1", "t", "true", "yes", "on", "enabled":
		return true
	case "0", "f", "false", "no", "off", "disabled":
		return false
	default:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Error parsing %s as bool: %v. Using default value: %t", key, err, defaultValue)
			return defaultValue
		}
		return boolValue
	}
}

// validateCronExpression validates the cron expression format
func validateCronExpression(cronExpr string) error {
	if cronExpr == "" {
		return fmt.Errorf("cron expression cannot be empty")
	}
	
	// Attempt to parse the cron expression
	_, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %v", err)
	}
	
	return nil
}

// ValidateRequiredConfig validates required configuration.
func ValidateRequiredConfig() error {
	if CFG.ClusterName == "" {
		return fmt.Errorf("CLUSTER_NAME is required but not set")
	}

	// Validate cron expression
	if err := validateCronExpression(CFG.CronSchedule); err != nil {
		return fmt.Errorf("CRON_SCHEDULE validation failed: %v", err)
	}

	// Validate metrics port
	if CFG.MetricsPort < 1024 || CFG.MetricsPort > 65535 {
		return fmt.Errorf("METRICS_PORT must be between 1024 and 65535, got %d", CFG.MetricsPort)
	}

	return nil
}
