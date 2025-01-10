package config

type PostgresConfig struct {
	Host     *string `mapstructure:"psql_host"`
	DdName   *string `mapstructure:"psql_db_name"`
	Username *string `mapstructure:"psql_username"`
	Port     *string `mapstructure:"psql_port"`
	Password *string `mapstructure:"psql_password"`
}

type AllConfiguration struct {
	Psql *PostgresConfig
	Env  *string `mapstructure:"environment"`
}
