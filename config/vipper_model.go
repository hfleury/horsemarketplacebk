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
}

type SMTPConfig struct {
	Host     string `mapstructure:"smtp_host"`
	Port     string `mapstructure:"smtp_port"`
	Username string `mapstructure:"smtp_username"`
	Password string `mapstructure:"smtp_password"`
	From     string `mapstructure:"mail_from"`
}
