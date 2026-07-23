package config

type PostgresConfig struct {
	Host     string `mapstructure:"psql_host"`
	DdName   string `mapstructure:"psql_db_name"`
	Username string `mapstructure:"psql_username"`
	Port     string `mapstructure:"psql_port"`
	Password string `mapstructure:"psql_password"`
	SSLMode  string `mapstructure:"psql_sslmode"`
}

type AllConfiguration struct {
	Psql      PostgresConfig `mapstructure:"psql"`
	PasetoKey string         `mapstructure:"paseto_key"`
	Env       string         `mapstructure:"environment"`
	SMTP      SMTPConfig     `mapstructure:"smtp"`
	Storage   StorageConfig  `mapstructure:"storage"`
}

type StorageConfig struct {
	Endpoint        string `mapstructure:"storage_endpoint"`
	PublicEndpoint  string `mapstructure:"storage_public_endpoint"`
	Region          string `mapstructure:"storage_region"`
	AccessKeyID     string `mapstructure:"storage_access_key_id"`
	SecretAccessKey string `mapstructure:"storage_secret_access_key"`
	BucketName      string `mapstructure:"storage_bucket_name"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"smtp_host"`
	Port     string `mapstructure:"smtp_port"`
	Username string `mapstructure:"smtp_username"`
	Password string `mapstructure:"smtp_password"`
	From     string `mapstructure:"mail_from"`
}
