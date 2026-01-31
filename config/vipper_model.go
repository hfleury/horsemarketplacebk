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
	AWS       AWSConfig      `mapstructure:"aws"`
}

type AWSConfig struct {
	Endpoint        string `mapstructure:"aws_endpoint"`
	PublicEndpoint  string `mapstructure:"aws_public_endpoint"`
	Region          string `mapstructure:"aws_region"`
	AccessKeyID     string `mapstructure:"aws_access_key_id"`
	SecretAccessKey string `mapstructure:"aws_secret_access_key"`
	BucketName      string `mapstructure:"aws_bucket_name"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"smtp_host"`
	Port     string `mapstructure:"smtp_port"`
	Username string `mapstructure:"smtp_username"`
	Password string `mapstructure:"smtp_password"`
	From     string `mapstructure:"mail_from"`
}
