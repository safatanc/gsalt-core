package main

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/safatanc/gsalt-core/injector"
	"github.com/safatanc/gsalt-core/internal/infrastructures"
	"github.com/sirupsen/logrus"
)

func main() {
	infrastructures.LoadConfig()

	app, err := injector.InitializeApplication()
	if err != nil {
		logrus.Fatalf("Failed to initialize application: %v", err)
	}

	// Fiber configuration
	config := fiber.Config{
		ReadTimeout:  time.Second * 60,
		WriteTimeout: time.Second * 60,
		IdleTimeout:  time.Second * 60,
	}

	router := fiber.New(config)

	// Add CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:  "*",
		AllowHeaders:  "Origin, Content-Type, Accept, Authorization",
		AllowMethods:  "GET, POST, PUT, DELETE, OPTIONS",
		ExposeHeaders: "Content-Length",
		MaxAge:        300,
	}))

	app.RegisterRoutes(router)

	logrus.Fatal(router.Listen(":8080"))
}
