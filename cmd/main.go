package main

import (
	"fmt"

	"github.com/hfleury/horsemarketplacebk/config"
)

// enviroment
// database
//

func main() {
	AppConfig := config.NewVipperService()
	AppConfig.GetAllConfiguration()
	fmt.Println(AppConfig)

}
