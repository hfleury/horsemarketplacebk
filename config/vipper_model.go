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
	Psql PostgresConfig `mapstructure:"psql"`
	Env  string         `mapstructure:"environment"`
}
