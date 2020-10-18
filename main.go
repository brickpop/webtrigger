package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
)

type jsonResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

var config Config
var activeTriggers map[string]bool

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

	app.Get("/:triggerID", handleGetStatus)
	app.Post("/:triggerID", handleRunTrigger)
	app.Use(handleNotFound)

	addr := fmt.Sprintf(":%d", config.Port)
	log.Fatal(app.Listen(addr))
}

// handleGetStatus handles the request to run a certain trigger
func handleGetStatus(ctx *fiber.Ctx) error {
	triggerID := ctx.Params("triggerID")
	authorizationHeader := ctx.Get("Authorization")

	_, httpStatus, err := findTrigger(triggerID, authorizationHeader)
	if err != nil {
		log.Printf("[GET] Trigger %s: %s\n", triggerID, err)

		ctx.Status(httpStatus)
		return ctx.SendString(fmt.Sprintf("%s", err))
	}
	if httpStatus == fiber.StatusLocked {
		log.Printf("[GET] Trigger %s is already running\n", triggerID)

		response := jsonResponse{ID: triggerID, Status: "running"}
		return ctx.JSON(response)
	}

	log.Printf("[GET] Trigger %s is not running\n", triggerID)

	response := jsonResponse{ID: triggerID, Status: "unstarted"}
	return ctx.JSON(response)
}

// handleRunTrigger handles the request to run a certain trigger
func handleRunTrigger(ctx *fiber.Ctx) error {
	// triggerID := ctx.Params("triggerID")
	// serviceToken := ctx.Params("serviceToken")
	log.Printf("[RUN] Trigger not found: %s\n", ctx.Path())

	// cmd := exec.Command("cat")
	// in, _ := cmd.StdinPipe()
	// out, _ := cmd.StdoutPipe()
	// err, _ := cmd.StderrPipe()

	// cmd := exec.Command("ls")
	// cmd.Stderr = os.Stderr
	// cmd.Stdout = os.Stdout
	// err = cmd.Run()

	return ctx.SendString("OK")
}

// handleNotFound sends an empty 404 response
func handleNotFound(ctx *fiber.Ctx) error {
	log.Printf("[HTTP] Not found %s\n", ctx.Path())
	return ctx.SendStatus(fiber.StatusNotFound)
}

// Helpers

func findTrigger(triggerID, authorizationHeader string) (*Trigger, int, error) {
	authorizationItems := strings.Split(authorizationHeader, " ")
	if len(authorizationItems) != 2 {
		return nil, fiber.StatusNotAcceptable, errors.Errorf("Invalid authorization header")
	}
	if authorizationItems[0] != "Bearer" {
		return nil, fiber.StatusNotAcceptable, errors.Errorf("The authorization header should be in the form \"Bearer <token>\"")
	}

	for _, trigger := range config.Triggers {
		if trigger.ID != triggerID {
			continue
		}
		if trigger.Token != authorizationItems[1] {
			return nil, fiber.StatusUnauthorized, errors.Errorf("Invalid token")
		}
		if activeTriggers[trigger.ID] {
			return &trigger, fiber.StatusLocked, nil
		}

		// OK
		return &trigger, fiber.StatusOK, nil
	}
	return nil, fiber.StatusNotFound, errors.Errorf("Trigger not found: %s", triggerID)
}

func showUsage() {
	log.Println("Usage: webtrigger <config-file>")
}
