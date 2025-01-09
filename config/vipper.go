package config

type PostgresConfig struct {
	Host     string `mapstructure:"psql_host"`
	DdName   string `mapstructure:"psql_db_name"`
	Username string `mapstructure:"psql_username"`
	Port     string `mapstructure:"psql_port"`
}

type Configuration struct {
	Psql PostgresConfig
	Env  string `mapstructure:"environment"`
}
