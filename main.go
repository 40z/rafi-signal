package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/40z/rafi-signal/plugins/fuckyou"
	"github.com/40z/rafi-signal/rafibot"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Name           string                                   `toml:"name"`
	FuckYouOptions fuckyou.Options                          `toml:"fuckyou"`
	SocketOptions  rafibot.SocketSignalClientAdapterOptions `toml:"socket"`
}

func main() {
	cfgFlag := flag.String("c", "config.toml", "Path to config file")
	vFlag := flag.Bool("v", false, "Verbose logging")
	flag.Parse()

	// Configure logging
	logLevel := slog.LevelInfo
	if *vFlag {
		logLevel = slog.LevelDebug
	}
	logOpts := slog.HandlerOptions{Level: logLevel}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &logOpts))

	// Configure options
	var cfg Config
	cfgFile, _ := os.ReadFile(*cfgFlag)
	toml.Unmarshal(cfgFile, &cfg)

	// Setup context for cancelling on ctrl+c
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// TODO: Needs real SignalClient adapter
	signalClient := rafibot.NewSocketSignalClient(logger, cfg.SocketOptions)
	bot := rafibot.New(&signalClient, cfg.Name)

	// Register plugins
	bot.OnMention(fuckyou.Handler)

	// registering with options might look like this
	// fuckYouPlugin := fuckyou.NewPlugin(cfg.FuckYouOptions)
	// bot.OnMention(fuckYouPlugin.Handler)

	bot.Run(ctx)
}
