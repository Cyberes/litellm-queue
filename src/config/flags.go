package config

import "flag"

var CliArgs *CliConfig

type CliConfig struct {
	ConfigFile string
	Debug      bool
	Help       bool
	Version    bool
}

func ParseArgs() {
	if CliArgs != nil {
		panic("already defined")
	}
	CliArgs = &CliConfig{}
	flag.StringVar(&CliArgs.ConfigFile, "config", "", "Path to the config file")
	flag.BoolVar(&CliArgs.Debug, "d", false, "Enable debug mode")
	flag.BoolVar(&CliArgs.Debug, "debug", false, "Enable debug mode")
	flag.BoolVar(&CliArgs.Version, "v", false, "Print version and exit")
	flag.Parse()
}
