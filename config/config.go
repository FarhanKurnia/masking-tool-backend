package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Config holds environment configuration
var Config *config

type config struct {
	Port              string
	DBType            string
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	JobsPageSize      int
	EncryptionKeyName string
	EncryptionKey     string
}

func init() {
	_ = godotenv.Load()

	jobsPageSize, err := strconv.Atoi(getEnv("JOBS_PAGE_SIZE", "20"))
	if err != nil {
		logrus.Warn("Invalid JOBS_PAGE_SIZE, using default 20")
		jobsPageSize = 20
	}

	Config = &config{
		Port:              getEnv("PORT", "8080"),
		DBType:            getEnv("DB_TYPE", "postgres"),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", ""),
		DBName:            getEnv("DB_NAME", "masking"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		JobsPageSize:      jobsPageSize,
		EncryptionKeyName: getEnv("ENCRYPTION_KEY_NAME", "default"),
		EncryptionKey:     getEnv("ENCRYPTION_KEY_default", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
