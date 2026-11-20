package fuckyou

import "github.com/40z/rafi-signal/rafibot"

type Options struct {
	AllCaps bool `toml:"all-caps"`
}

type Plugin struct {
	Options Options
}

func NewPlugin(options Options) *Plugin {
	return &Plugin{
		Options: options,
	}
}

func (s *Plugin) Handler(msg rafibot.Message, out chan<- string) {
	if s.Options.AllCaps {
		out <- "FUCK YOU!"
	} else {
		out <- "fuck you"
	}
}

func Handler(msg rafibot.Message, out chan<- string) {
	NewPlugin(Options{}).Handler(msg, out)
}
