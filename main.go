package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
)

var config Config

func main() {
	// Arguments and parameters
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	cfg, err := ReadConfig(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	config = cfg

	// Service definition
	app := fiber.New()

	app.Use(func(ctx *fiber.Ctx) error {
		ctx.Set("Access-Control-Allow-Origin", "*")
		ctx.Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		return ctx.Next()
	})

	app.Options("*", func(ctx *fiber.Ctx) error {
		ctx.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		return ctx.Next()
	})

	app.Get("/:serviceID", handleGetStatus)
	app.Post("/:serviceID", handleTrigger)
	app.Use(handleNotFound)

	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("[MAIN] Listening on %s\n", addr)

	log.Fatal(app.Listen(addr))
}

// handleGetStatus handles the request to run a certain trigger
func handleGetStatus(ctx *fiber.Ctx) error {
	// serviceID := ctx.Params("serviceID")
	// serviceToken := ctx.Params("serviceToken")
	log.Printf("[GET] Trigger not found: %s\n", ctx.Path())

	// dynamicLink := MakeEntityLink(serviceID, config)
	// ctx.Set("Location", dynamicLink)

	return ctx.SendStatus(fiber.StatusFound) // 302
}

// handleTrigger handles the request to run a certain trigger
func handleTrigger(ctx *fiber.Ctx) error {
	// serviceID := ctx.Params("serviceID")
	// serviceToken := ctx.Params("serviceToken")
	log.Printf("[RUN] Trigger not found: %s\n", ctx.Path())

	// dynamicLink := MakeEntityLink(serviceID, config)
	// ctx.Set("Location", dynamicLink)

	return ctx.SendString("OK")
}

// handleNotFound sends an empty 404 response
func handleNotFound(ctx *fiber.Ctx) error {
	log.Printf("[ERROR] Trigger not found: %s\n", ctx.Path())
	return ctx.SendStatus(fiber.StatusNotFound)
}

func showUsage() {
	fmt.Println("Usage: webtrigger <config-file>")
}
