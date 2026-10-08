package main

import (
	"context"
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
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// TODO: Needs real SignalClient adapter
	bot := rafibot.New(nil, "+12345678901")

	// Configure options
	var cfg Config
	cfgFile, _ := os.ReadFile("config.toml")
	toml.Unmarshal(cfgFile, &cfg)

	// Register plugins
	bot.
		OnMention(fuckyou.Handler(cfg.FuckyouOptions))

	bot.Run(ctx)
}
