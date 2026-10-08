package fuckyou

import "github.com/40z/rafi-signal/rafibot"

type Options struct {
	AllCaps bool `toml:"all-caps"`
}

func Handler(options Options) rafibot.MessageHandler {
	return func(msg rafibot.Message, out chan<- string) {
		if options.AllCaps {
			out <- "FUCK YOU!"
		} else {
			out <- "fuck you"
		}
	}
}
