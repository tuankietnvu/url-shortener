package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"url-shortener/internal/config"
	"url-shortener/internal/database"
	"url-shortener/internal/handler"
	"url-shortener/internal/repository"
)

func main() {
	// Khởi tạo router của Gin
	r := gin.Default()

	cfg := config.LoadConfig()
	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		panic(err)
	}

	db := database.ConnectDB(cfg.DatabaseURL)
	urlRepo := repository.NewURLRepository(db)
	urlHandler := handler.NewURLHandler(urlRepo)
	urlHandler.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// Gin's r.Run expects an address like ":8080".
	if len(port) > 0 && port[0] != ':' {
		port = ":" + port
	}

	// Tạo một API Ping để test server
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
			"status":  "Hệ thống URL Shortener đã sẵn sàng!",
		})
	})

	// Chạy server ở port 8080
	r.Run(port)
}