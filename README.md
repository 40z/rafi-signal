# rafi-signal

A simple bot framework for Signal groups, written in Go. The bot listens to a group, hands every message to a set of plugins, and sends back whatever the plugins reply with. Plugins are small Go packages that you compile into the bot and configure from a TOML file.

## How it works

- **`rafibot`** is the framework. It connects to Signal through a `SignalClient` interface, distributes each incoming message to every registered handler, and sends the handlers' replies back to the group.
- **Plugins** are handler functions that react to messages, to mentions of the bot, or to a cron schedule. A plugin replies by sending a string on the channel it is given.
- **`main.go`** builds the bot, loads `config.toml`, and registers the plugins.

Handlers run concurrently, each in its own goroutine, so any state a plugin shares between calls must be protected with a mutex.

## Project layout

```
main.go                    # builds the bot, loads config, registers plugins
config.toml                # plugin settings, one [table] per plugin
rafibot/                   # the bot framework
plugins/<name>/<name>.go   # one package per plugin
```

## Setup

Requirements: Go 1.25.5 or newer.

1. Clone the repo and review `config.toml`.
2. In `main.go`, set the bot's name to your bot's Signal phone number. Mentions are matched against this value.
3. Start the bot:

   ```sh
   go run .
   ```

> **Status:** the Signal connection is not implemented yet. `main.go` currently passes `nil` as the `SignalClient` (marked with a TODO), so the bot cannot run end to end until a real adapter is written. Plugins can still be developed and unit tested on their own.

## Configuration

`config.toml` holds one table per plugin, and each table maps to that plugin's `Options` struct:

```toml
[fuckyou]
all-caps = false
```

Missing files, malformed TOML, and unknown keys are currently ignored without an error, and the plugin simply receives its default (zero) values. If a setting seems to have no effect, check the key spelling first.

## Creating a plugin

1. **Create the package.** Add `plugins/<name>/<name>.go` with `package <name>`.
2. **Define options (optional).** Export an `Options` struct for anything configurable, using kebab-case `toml` tags. Make the zero value a sensible default.
3. **Export a `Handler` constructor.** It takes `Options` and returns a `rafibot.MessageHandler` (or a `rafibot.ScheduleHandler` for scheduled plugins). Keep any state inside the returned closure.
4. **Register it in `main.go`.** Import the package, add its options to the `Config` struct, and add it to the registration chain with one of:
   - `OnMessage` to see every message
   - `OnMention` to react only when the bot is mentioned
   - `OnSchedule` to run on a cron schedule
5. **Add its settings to `config.toml`** as a `[<name>]` table.
6. **Test it.** Handlers are plain functions, so you can call them directly in a unit test with a `rafibot.Message` and a buffered channel, without a Signal connection.

`plugins/fuckyou` is a minimal working example to copy from.

### Things to know

- Schedules use six-field cron expressions with seconds first (for example `0 30 9 * * 1-5` for 09:30:00 on weekdays). An invalid expression is silently ignored, so the job will just never run.
- A panic inside a handler is not recovered and will stop the whole bot.
- Don't send on the reply channel from a goroutine that outlives the handler call, since the channel is closed on shutdown.
