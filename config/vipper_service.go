package config

import (
	"log"

	"github.com/spf13/viper"
)

type VipperService struct {
	Config *AllConfiguration
}

func NewVipperService() *VipperService {
	viper.AutomaticEnv()
	viper.SetDefault("environment", "development")
	return &VipperService{
		Config: &AllConfiguration{},
	}
}

func (vs *VipperService) LoadConfiguration() {
	// Load each environment variable manually with uppercase names
	vs.Config.Psql.Host = viper.GetString("PSQL_HOST")
	vs.Config.Psql.DdName = viper.GetString("PSQL_DB_NAME")
	vs.Config.Psql.Username = viper.GetString("PSQL_USERNAME")
	vs.Config.Psql.Port = viper.GetString("PSQL_PORT")
	vs.Config.Psql.Password = viper.GetString("PSQL_PASSWORD")
	vs.Config.Psql.SSLMode = viper.GetString("PSQL_SSLMODE")
	vs.Config.PasetoKey = viper.GetString("PASETO_KEY")
	vs.Config.Env = viper.GetString("ENVIRONMENT")

	// SMTP / mail settings (optional)
	vs.Config.SMTP.Host = viper.GetString("SMTP_HOST")
	vs.Config.SMTP.Port = viper.GetString("SMTP_PORT")
	vs.Config.SMTP.Username = viper.GetString("SMTP_USERNAME")
	vs.Config.SMTP.Password = viper.GetString("SMTP_PASSWORD")
	vs.Config.SMTP.From = viper.GetString("MAIL_FROM")

	// AWS / MinIO
	vs.Config.AWS.Endpoint = viper.GetString("AWS_ENDPOINT")
	vs.Config.AWS.PublicEndpoint = viper.GetString("AWS_PUBLIC_ENDPOINT")
	vs.Config.AWS.Region = viper.GetString("AWS_REGION")
	vs.Config.AWS.AccessKeyID = viper.GetString("AWS_ACCESS_KEY_ID")
	vs.Config.AWS.SecretAccessKey = viper.GetString("AWS_SECRET_ACCESS_KEY")
	vs.Config.AWS.BucketName = viper.GetString("AWS_BUCKET_NAME")

	// Log loaded configuration for debugging
	log.Printf("Loaded configuration: %+v", vs.Config)

	// Optionally, you can check if some values are missing or invalid
	if vs.Config.Psql.Host == "" || vs.Config.Psql.DdName == "" {
		log.Fatalf("Missing required database environment variables")
	}
}

func (vs *VipperService) GetConfig() *AllConfiguration {
	return vs.Config
}
