package main

import (
	"errors"
	"net/http"
	"os"
	"os/signal"
	"server/handler"
	"server/logging"
	"server/queue"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize logger.
	logging.InitLogger(logrus.DebugLevel)
	log := logging.GetLogger()

	// Define the model configuration.
	modelConfig := map[string]int{
		"claude-3-sonnet": 2,
		// Add other models here.
	}

	// Initialize the Queue Manager with the model configuration.
	backendURL := "https://llm.evulid.cc" // Replace with your actual backend URL.
	reqQueueManager := queue.NewQueueManager(modelConfig, backendURL)

	// Initialize the HTTP handler with the Queue Manager.
	httpHandler := handler.NewHTTPHandler(reqQueueManager)

	// Define the server.
	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: httpHandler,
	}

	// Channel to listen for OS signals for graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Infoln("Starting server on 0.0.0.0:8080")
		// Start listening and serving.
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Block until a signal is received.
	<-quit
	log.Infoln("Shutting down server...")

	// Shutdown the Queue Manager.
	reqQueueManager.Shutdown()
}
