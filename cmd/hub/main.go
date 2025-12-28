package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
)

func main() {
	addr := getenv("HUB_LISTEN_ADDR", ":8080")

	app := fiber.New()

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true})
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString("xVault Hub API")
	})

	log.Printf("hub listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("hub server error: %v", err)
	}
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
