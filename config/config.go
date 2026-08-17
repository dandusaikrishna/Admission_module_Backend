package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string

	SMTPHost  string
	SMTPPort  string
	SMTPUser  string
	SMTPPass  string
	EmailFrom string
	// Kafka
	KafkaBrokers  string
	KafkaTopic    string
	KafkaDLQTopic string
}

var AppConfig Config

func LoadConfig() {
	envLocations := []string{
		".env",              
		"config/.env",      
		"../config/.env",   
		"../../config/.env", 
	}

	for _, location := range envLocations {
		if err := godotenv.Load(location); err == nil {
			break
		}
	}

	AppConfig = Config{
		DBHost:     getEnvWithDefault("DB_HOST", "localhost"),
		DBPort:     getEnvWithDefault("DB_PORT", "5432"),
		DBUser:     getEnvWithDefault("DB_USER", "postgres"),
		DBPassword: getEnvWithDefault("DB_PASSWORD", "Sai@6303179072$"),
		DBName:     getEnvWithDefault("DB_NAME", "postgres"),

		RazorpayKeyID:         os.Getenv("RazorpayKeyID"),
		RazorpayKeySecret:     os.Getenv("RazorpayKeySecret"),
		RazorpayWebhookSecret: os.Getenv("RAZORPAY_WEBHOOK_SECRET"),

		SMTPHost:  getEnvWithDefault("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:  getEnvWithDefault("SMTP_PORT", "587"),
		SMTPUser:  os.Getenv("SMTP_USER"),
		SMTPPass:  os.Getenv("SMTP_PASS"),
		EmailFrom: os.Getenv("EMAIL_FROM"),

		KafkaBrokers:  getEnvWithDefault("KAFKA_BROKERS", "127.0.0.1:9092"),
		KafkaTopic:    getEnvWithDefault("KAFKA_TOPIC", "admissions.payments"),
		KafkaDLQTopic: getEnvWithDefault("KAFKA_DLQ_TOPIC", "admissions.payments.dlq"),
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetDBConnString() string {
	return "host=" + AppConfig.DBHost +
		" port=" + AppConfig.DBPort +
		" user=" + AppConfig.DBUser +
		" password=" + AppConfig.DBPassword +
		" dbname=" + AppConfig.DBName +
		" sslmode=require"
}
