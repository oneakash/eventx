package main

import (
    "os"
	"fmt"
	// "net/http"
    _ "eventx/routers"

    "github.com/beego/beego/v2/server/web"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using real environment variables")
	} else {
		fmt.Println(".env loaded")
	}
    wd, _ := os.Getwd()
    web.BConfig.WebConfig.ViewsPath = wd + "/views"
    web.Run()
}