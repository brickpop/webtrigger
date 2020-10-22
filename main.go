package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kballard/go-shellquote"
	"github.com/pkg/errors"
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
	fmt.Printf("[MAIN] Using %d triggers\n", len(config.Triggers))

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
	app.Post("/:triggerID", handlespawnTriggerCommand)
	app.Use(handleNotFound)

	if config.TLS.Certificate != "" && config.TLS.Key != "" {
		// Read TLS certificate
		cer, err := tls.LoadX509KeyPair(config.TLS.Certificate, config.TLS.Key)
		if err != nil {
			log.Fatal(err)
		}

		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cer}}
		addr := fmt.Sprintf(":%d", config.Port)

		// Create custom listener
		ln, err := tls.Listen("tcp", addr, tlsConfig)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("[MAIN] Listening TLS on %s\n", addr)
		log.Fatal(app.Listener(ln))
	} else {
		addr := fmt.Sprintf(":%d", config.Port)
		fmt.Printf("[MAIN] Listening HTTP on %s\n", addr)
		log.Fatal(app.Listen(addr))
	}
}

// handleGetStatus handles the request to run a certain trigger
func handleGetStatus(ctx *fiber.Ctx) error {
	triggerID := ctx.Params("triggerID")
	authorizationHeader := ctx.Get("Authorization")

	trigger, httpStatus, err := findTrigger(triggerID, authorizationHeader)
	if err != nil {
		log.Printf("[GET] %s: %s\n", triggerID, err)

		ctx.Status(httpStatus)
		return ctx.SendString(fmt.Sprintf("%s", err))
	}

	// FOUND
	log.Printf("[GET] %s status\n", triggerID)

	switch trigger.Status {
	case StatusUnstarted:
	case StatusRunning:
	case StatusDone:
	case StatusFailed:

	default:
		ctx.Status(fiber.StatusInternalServerError)
		return ctx.SendString(fmt.Sprintf("[GET] Internal server error: Unknown trigger status %d", trigger.Status))
	}

	type JSONResponse struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	response := JSONResponse{ID: triggerID, Status: trigger.Status.String()}

	return ctx.JSON(response)
}

// handlespawnTriggerCommand handles the request to run a certain trigger
func handlespawnTriggerCommand(ctx *fiber.Ctx) error {
	triggerID := ctx.Params("triggerID")
	authorizationHeader := ctx.Get("Authorization")

	trigger, httpStatus, err := findTrigger(triggerID, authorizationHeader)
	if err != nil {
		log.Printf("[POST] %s: %s\n", triggerID, err)

		ctx.Status(httpStatus)
		return ctx.SendString(fmt.Sprintf("%s", err))
	}

	// FOUND
	log.Printf("[POST] %s requested\n", triggerID)

	err = spawnTriggerCommand(trigger)
	if err != nil {
		ctx.Status(fiber.StatusInternalServerError)
		return ctx.SendString(fmt.Sprintf("[POST] %s", err))
	}

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
		return nil, fiber.StatusNotAcceptable, errors.New("Invalid authorization header")
	}
	if authorizationItems[0] != "Bearer" {
		return nil, fiber.StatusNotAcceptable, errors.New("The authorization header should be in the form \"Bearer <token>\"")
	}

	for idx := range config.Triggers {
		trigger := &config.Triggers[idx]
		if trigger.ID != triggerID {
			continue
		}
		if trigger.Token != authorizationItems[1] {
			return nil, fiber.StatusUnauthorized, errors.New("Invalid token")
		}

		// OK
		return trigger, fiber.StatusOK, nil
	}
	return nil, fiber.StatusNotFound, errors.Errorf("Trigger not found: %s", triggerID)
}

func spawnTriggerCommand(trigger *Trigger) error {
	// Check status
	if trigger.Status == StatusRunning {
		if trigger.WaitGroup == nil {
			log.Println("[RUNNER] Error: The trigger is running but has a nil waitgroup")
			return errors.New("Internal error")
		}

		// Wait for ongoing processes
		log.Printf("[RUNNER] %s waiting for an ongoing execution", trigger.ID)
		trigger.WaitGroup.Wait()
	}

	trigger.WaitGroup = &sync.WaitGroup{}
	trigger.WaitGroup.Add(1)

	// Detect binary and split from the argument list
	commandItems, err := shellquote.Split(trigger.Command)
	if err != nil {
		return err
	}
	executableFile := commandItems[0]
	args := commandItems[1:]

	// Handle timeout (if any)
	var cmd *exec.Cmd
	var cancel context.CancelFunc

	if trigger.Timeout > 0 {
		var ctx context.Context
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(trigger.Timeout)*time.Second)
		cmd = exec.CommandContext(ctx, executableFile, args...)
	} else {
		cmd = exec.Command(executableFile, args...)
	}

	// Pipe output
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// Spawn command
	log.Printf("[RUNNER] Starting %s ", trigger.ID)
	trigger.Status = StatusRunning

	err = cmd.Start()
	go func() {
		if cancel != nil {
			defer cancel()
		}
		err := cmd.Wait()
		if err != nil {
			log.Printf("[RUNNER] Failed: %s > %s", trigger.ID, err)
			trigger.Status = StatusFailed
		} else {
			log.Printf("[RUNNER] Done: %s", trigger.ID)
			trigger.Status = StatusDone
		}
		trigger.WaitGroup.Done()
	}()

	return err
}

func showUsage() {
	log.Println("Usage: webtrigger <config-file>")
}
