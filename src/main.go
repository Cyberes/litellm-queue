package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"server/config"
	"server/handler"
	"server/logging"
	"server/manager"
	"syscall"

	"github.com/sirupsen/logrus"
)

var Version = "development"
var VersionDate = "not set"

func main() {
	config.ParseArgs()
	if config.CliArgs.Help {
		flag.Usage()
		os.Exit(0)
	}
	if config.CliArgs.Version {
		buildInfo, ok := debug.ReadBuildInfo()

		if ok {
			buildSettings := make(map[string]string)
			for i := range buildInfo.Settings {
				buildSettings[buildInfo.Settings[i].Key] = buildInfo.Settings[i].Value
			}
			fmt.Printf("Version: %s\n\n", Version)
			fmt.Printf("Date Compiled: %s\n", VersionDate)
			fmt.Printf("Git Revision: %s\n", buildSettings["vcs.revision"])
			fmt.Printf("Git Revision Date: %s\n", buildSettings["vcs.time"])
			fmt.Printf("Git Modified: %s\n", buildSettings["vcs.modified"])
		} else {
			fmt.Println("Build info not available")
		}
		os.Exit(0)
	}

	if config.CliArgs.Debug {
		logging.InitLogger(logrus.DebugLevel)
	} else {
		logging.InitLogger(logrus.InfoLevel)
	}
	log := logging.GetLogger()

	if config.CliArgs.ConfigFile == "" {
		exePath, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exeDir := filepath.Dir(exePath)

		if _, err := os.Stat(filepath.Join(exeDir, "config.yml")); err == nil {
			if _, err := os.Stat(filepath.Join(exeDir, "config.yaml")); err == nil {
				log.Fatalln("Both config.yml and config.yaml exist in the executable directory. Please specify one with the --config flag.")
			}
			config.CliArgs.ConfigFile = filepath.Join(exeDir, "config.yml")
		} else if _, err := os.Stat(filepath.Join(exeDir, "config.yaml")); err == nil {
			config.CliArgs.ConfigFile = filepath.Join(exeDir, "config.yaml")
		} else {
			log.Fatalln("No config file found in the executable directory. Please place config.yaml next to the executable or set the path to the config with the --config flag.")
		}
	}
	configData, err := config.LoadConfig(config.CliArgs.ConfigFile)
	if err != nil {
		log.Fatalf(`Failed to load config: %s`, err)
	}

	defaultConcurrency := 100
	concurrencyManager := manager.NewConcurrencyManager(configData.Models, defaultConcurrency)

	httpHandler := handler.NewHTTPHandler(concurrencyManager, configData.BackendURL)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		server := &http.Server{
			Addr:    configData.ListenAddress,
			Handler: httpHandler,
		}
		log.Infof("Starting server on %s", configData.ListenAddress)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-quit
	log.Infoln("Shutting down server...")
	concurrencyManager.Shutdown()
}
