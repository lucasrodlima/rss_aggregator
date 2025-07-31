package main

import (
	"fmt"
	"github.com/lucasrodlima/rss_aggregator/internal/config"
)

func main() {
	sysConfig, err := config.Read()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = sysConfig.SetUser("lucasrodlima")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Config file contents:\n\nDbUrl: %s\nUsername: %s\n",
		sysConfig.DbUrl,
		sysConfig.CurrentUserName)
}
