package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

/*
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rqManager.Start(ctx)

	http.HandleFunc("/ws", wsHandler)

	addr := ":8080"
	log.Printf("Starting server on %s", addr)
	srv := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("ListenAndServe error: %v", err)
	}
}
*/

func main() {
	// Step 1: Connect to MongoDB
	connectToMongoDB()
	log.Println("Connected to MongoDB successfully.")

	// Step 2: Create Echo instance
	e := echo.New()

	// Health check endpoint (to ensure server is running)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "Server is running",
		})
	})

	// Step 4: Register route controllers
	UserRouteController(e)
	ChatRouteController(e)

	// Step 5: Start server
	port := "8080"
	log.Printf("Server running on port %s\n", port)
	if err := e.Start(":" + port); err != nil {
		log.Fatal("Error starting server:", err)
	}
}
