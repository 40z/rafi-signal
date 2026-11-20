package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/40z/rafi-signal/plugins/fuckyou"
	"github.com/40z/rafi-signal/rafibot"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	FuckyouOptions fuckyou.Options `toml:"fuckyou"`
}

func main() {
	cfgFlag := flag.String("c", "config.toml", "Path to config file")
	flag.Parse()

	// Configure options
	var cfg Config
	cfgFile, _ := os.ReadFile(*cfgFlag)
	toml.Unmarshal(cfgFile, &cfg)

	// Setup context for cancelling on ctrl+c
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// TODO: Needs real SignalClient adapter
	bot := rafibot.New(nil, "+12345678901")


	// Register plugins
	bot.OnMention(fuckyou.Handler)

	// registering with options might look like this
	// fuckYouPlugin := fuckyou.NewPlugin(cfg.FuckYouOptions)
	// bot.OnMention(fuckYouPlugin.Handler)

	bot.Run(ctx)
}
