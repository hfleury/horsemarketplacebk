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
	if err := viper.Unmarshal(vs.Config); err != nil {
		log.Fatalf("Error unmarshalling env variable: %v", err)
	}
}

func (vs *VipperService) GetConfig() *AllConfiguration {
	return vs.Config
}
