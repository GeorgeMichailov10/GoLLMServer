package main

import (
	"context"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	// Connect DB
	connectToMongoDB()

	e := echo.New()

	// Check run
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "Server is running",
		})
	})

	// Socket + Model Service related
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rqManager.Start(ctx)
	e.GET("/ws", func(c echo.Context) error {
		wsHandler(c.Response(), c.Request())
		return nil
	}, JWTMiddleware)

	/*
		// Socket + Simulated Model Service
		e.GET("/ws", func(c echo.Context) error {
			SimulationWsHandler(c.Response(), c.Request())
			return nil
		}, JWTMiddleware)
	*/

	// Controllers
	UserRouteController(e)
	ChatRouteController(e)

	// Server itself
	port := "8080"
	log.Printf("Server running on port %s\n", port)
	if err := e.Start(":" + port); err != nil {
		log.Fatal("Error starting server:", err)
	}
}
