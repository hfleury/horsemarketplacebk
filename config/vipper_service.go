package config

import (
	"log"

	"github.com/spf13/viper"
)

type VipperService struct {
	Config *Configuration
}

func NewVipperService() *VipperService {
	viper.AutomaticEnv()
	viper.SetDefault("environment", "development")
	return &VipperService{
		Config: &Configuration{},
	}
}

func (vs *VipperService) GetAllConfiguration() {
	if err := viper.Unmarshal(vs.Config); err != nil {
		log.Fatalf("Error unmarshalling env variable: %v", err)
	}
}
