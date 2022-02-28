package main

// Usage:
//	go run main.go
//	curl

import (
	"fmt"
	"os"
)

var (
	// 应用版本信息
	Version   string
	BuildTime string
	GoVersion string
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("Version:%s\nBuild Time:%s\nGo Version:%s\n", Version, BuildTime, GoVersion)
		return
	}
}
