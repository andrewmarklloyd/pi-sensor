package main

import (
	"fmt"
	"os"

	"github.com/dghubble/sessions"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
)

// sessionStore encodes and decodes session data stored in signed cookies
var sessionStore *sessions.CookieStore

// NewServer returns a new ServeMux with app routes.
func NewServer() {
	// Set the router as the default one shipped with Gin
	router := gin.Default()

	// Serve frontend static files
	router.Use(static.Serve("/", static.LocalFile("./frontend/build", true)))

	// Start and run the server

	router.Run(fmt.Sprintf(":%s", os.Getenv("PORT")))
}
