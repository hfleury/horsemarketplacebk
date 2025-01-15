package config

type Configuration interface {
	LoadConfiguration()
	GetConfig() *AllConfiguration
}
