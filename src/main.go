package main

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"server/handler"
	"server/logging"
	"server/queue"
)

func main() {
	//if config.CliArgs.Debug {
	logging.InitLogger(logrus.DebugLevel)
	//} else {
	//	logging.InitLogger(logrus.InfoLevel)
	//}
	log := logging.GetLogger()

	// Initialize the request queue
	reqQueue := queue.NewRequestQueue("https://google.com")

	// Initialize the HTTP handler with the request queue and backend client
	httpHandler := handler.NewHTTPHandler(reqQueue)

	// Define the server
	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: httpHandler,
	}

	log.Infoln("Starting server on 0.0.0.0:8080")
	// Start listening and serving
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
