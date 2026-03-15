package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Khởi tạo router của Gin
	r := gin.Default()

	// Tạo một API Ping để test server
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
			"status":  "Hệ thống URL Shortener đã sẵn sàng!",
		})
	})

	// Chạy server ở port 8080
	r.Run(":8080")
}