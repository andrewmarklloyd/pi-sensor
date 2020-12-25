package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
)

// NewServer returns a new ServeMux with app routes.
func NewServer() {
	// Set the router as the default one shipped with Gin
	router := gin.Default()

	// Serve frontend static files
	router.Use(static.Serve("/", static.LocalFile("./frontend/build", true)))

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalln("PORT must be set")
	}

	fmt.Println(fmt.Sprintf("Running web server on port %s", port))
	router.Run(fmt.Sprintf(":%s", port))
}
