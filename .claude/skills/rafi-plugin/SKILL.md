---
name: rafi-plugin
description: Build, extend, test, or debug plugins for rafibot, the Go bot framework for Signal groups (github.com/40z/rafi-signal). Use when the user wants a new rafibot plugin, a message/mention/scheduled handler, a cron job for the bot, plugin options in config.toml, or help with a plugin that misbehaves (races, panics, silent cron or config failures, duplicate replies).
---

# Writing rafibot plugins

rafibot is a small Go framework (`package rafibot`, module `github.com/40z/rafi-signal`). A **plugin is a Go package
under `plugins/` that exports a handler function**, wired into `main.go` and configured from `config.toml`. There is
no plugin interface or dynamic loader: plugins are compiled in and registered in code.

## Repo layout

```
main.go                    # builds the bot, loads config.toml, registers plugins
config.toml                # one [table] per plugin
rafibot/rafibot.go         # the framework (don't edit for a plugin)
plugins/<name>/<name>.go   # one package per plugin; package name = dir name = file name
```

`plugins/fuckyou/fuckyou.go` is the reference example; read it and `main.go` before writing a new plugin to confirm these conventions still hold.

## The framework API (everything a plugin can touch)

```go
type Message struct {
    Sender   string
    Body     string
    Mentions map[string]struct{} // set of mentioned names
}

type MessageHandler  = func(Message, chan<- string)
type ScheduleHandler = func(chan<- string)

bot := rafibot.New(signalClient, "botname")
bot.OnMessage(h)                    // h runs for every incoming message
bot.OnMention(h)                    // same, but only when Mentions contains the bot's name
bot.OnSchedule("0 30 9 * * 1-5", h) // cron, see "Cron" below
// all three return *RafiBot, so registration chains
```

To reply, send a plain string on the `chan<- string` the handler receives. That is the only output mechanism: no recipient argument, no formatting, no error return.

In `main.go` the bot's name is its Signal phone number (`rafibot.New(nil, "+12345678901")`), so `OnMention` matches when `Mentions` contains that exact string.

## Simple plugins

The simplest plugin can just be a single exported function. The convention is to name the handler `Handler`, but that
is just a convention

## The plugin convention

Every plugin follows a similar shape (see `plugins/fuckyou/fuckyou.go`):

1. `package <name>` in `plugins/<name>/<name>.go`.
2. An exported `Options` struct holding the plugin's settings, with **kebab-case `toml` tags** (`AllCaps bool \`toml:"all-caps"\``).
3. An exported `Plugin` struct with a constructor that accepts the `Options` object. Config and state live in the plugin instance, not in globals.
4. If a plugin has no settings or state, omit `Options` and the `Plugin` structs in favor of a single exported handler function.

## How handlers run (this drives most design decisions)

- **Each handler is invoked in its own goroutine for every message.** Handlers run concurrently with each other and with other invocations of themselves. Any shared state (counters, maps, caches) needs a `sync.Mutex`, `sync.RWMutex`, or atomic.
- **A panic in a handler is not recovered** and takes down the whole bot process. Guard against nil maps, index-out-of-range, and bad type assertions. Consider a `defer recover()` wrapper for risky plugins.
- **The output channel is buffered (20) and closed on shutdown.** Sending after the context is cancelled panics (send on closed channel). Do not spawn goroutines that outlive the handler call and send later; use `OnSchedule` for delayed or periodic output.
- **Handlers see every message** (`OnMention` just wraps the handler in a mention check), so filter early and cheaply and return fast. The example plugin replies to every mention regardless of the body; a plugin that should react only to certain text must check `msg.Body` itself.

## Cron

`OnSchedule` uses `cron.New(cron.WithSeconds())` from `robfig/cron/v3`, so expressions have **six fields, seconds first**:

```
sec min hour dom month dow
"0 0 9 * * 1-5"   // 09:00:00 on weekdays
"0 */15 * * * *"  // every 15 minutes
```

A standard five-field expression is the most common mistake. The framework **ignores the error from `AddFunc`**, so an invalid expression fails silently and the job never runs. If a user reports "my scheduled plugin never fires," check this first. Time zone is the host's local time.

## Config

`main.go` reads `config.toml` into a `Config` struct with one field per plugin:

```go
type Config struct {
    FuckyouOptions fuckyou.Options `toml:"fuckyou"`
}
```

and `config.toml` has a matching table:

```toml
[fuckyou]
all-caps = false
```

**Both the file read and `toml.Unmarshal` ignore their errors.** A missing file, malformed TOML, or a mistyped key does not fail; the plugin just receives zero values. So:

- Design `Options` so the **zero value is a sensible default** (or fill defaults inside `Handler`).
- When a user says "my setting has no effect," check the key spelling and tag (`all-caps`, not `all_caps`), the table name, and that the TOML parses.

## Workflow for a new plugin

1. **Clarify the behavior in one pass:** trigger (every message, mention, or schedule), what it replies, whether it keeps state, and what should be configurable. Ask only for what is missing.
2. **Create `plugins/<name>/<name>.go`** following the convention above (template below).
3. **Wire it into `main.go`:**
   - import `github.com/40z/rafi-signal/plugins/<name>`
   - add `<Name>Options <name>.Options \`toml:"<name>"\`` to `Config` (only if the plugin has options)
   - add it to the chain under `// Register plugins`, e.g. `.OnMention(<name>.Handler)` or `.OnSchedule("0 50 9 * * 1-5", <name>.Reminder)`
4. **Add a `[<name>]` table to `config.toml`** with the defaults.
5. **Write a unit test** `plugins/<name>/<name>_test.go` (handlers are plain functions; see below).
6. **Run `go mod tidy`, `go vet ./...`, `go test -race ./...`.** The race detector matters because of the concurrency model. Run `go mod tidy` whenever a plugin adds an import.

## Template: message plugin with options and state

```go
package counter

import (
    "fmt"
    "strings"
    "sync"

    "github.com/40z/rafi-signal/rafibot"
)

type Options struct {
    Prefix string `toml:"prefix"` // trigger word; empty means "!hi"
}

type Plugin struct {
	Options Options
  mu      sync.Mutex
}

func NewPlugin(options Options) *Plugin {
	return &Plugin{
		Options: options,
	}
}

// Handler counts how many times each sender has used the trigger.
func (s *Plugin) Handler(msg rafibot.Message, out chan<- string) {
    prefix := s.options.Prefix
    if prefix == "" {
        prefix = "!hi" // zero value must be a sane default
    }

    counts := map[string]int{}

    if !strings.HasPrefix(msg.Body, prefix) {
        return
    }
    s.mu.Lock()
    counts[msg.Sender]++
    n := counts[msg.Sender]
    s.mu.Unlock()
    out <- fmt.Sprintf("%s has said hi %d times", msg.Sender, n)
}
```

**Wiring:**

`Config` gets `CounterOptions counter.Options \`toml:"counter"\``.

Register the plugin with:

```go
counterPlugin := counter.NewPlugin(cfg.CounterOptions)
bot.OnMessage(counterPlugin.Handler)
```

Add configuration to `config.toml`:

```toml
[counter]
prefix = "!hi"
```

## Template: simple scheduled plugin

```go
package standup

import "github.com/40z/rafi-signal/rafibot"

func Reminder(out chan<- string) {
    out <- "Standup in 10 minutes."
}
```

Register with `.OnSchedule("0 50 9 * * 1-5", standup.Reminder)`.

## Template: test

```go
package counter

import (
    "testing"

    "github.com/40z/rafi-signal/rafibot"
)

func TestHandler(t *testing.T) {
    h := NewPlugin(Options{}).Handler // zero-value options exercise the defaults
    out := make(chan string, 5) // buffered so the handler never blocks

    h(rafibot.Message{Sender: "alice", Body: "!hi"}, out)
    h(rafibot.Message{Sender: "alice", Body: "!hi"}, out)
    h(rafibot.Message{Sender: "alice", Body: "unrelated"}, out)

    if got := len(out); got != 2 {
        t.Fatalf("expected 2 replies, got %d", got)
    }
    <-out
    if got := <-out; got != "alice has said hi 2 times" {
        t.Fatalf("unexpected reply: %q", got)
    }
}
```

Test mention-gated plugins by passing `Mentions: map[string]struct{}{"+12345678901": {}}` when going through `OnMention`, or call the plugin's handler directly (it has no mention check of its own). To test through the bot itself, construct `rafibot.New(fake, name)` with a fake `SignalClient` (`Start(chan<- Message)`, `Stop()`, `SendMessage(string) error`).

## Pitfalls checklist

- Shared state without a mutex (data race; run with `-race`).
- Five-field cron string (silently never runs).
- Option keys or tags that don't match `config.toml` (silently ignored); zero-value options that break the plugin.
- Forgetting one of the wiring steps (import, `Config` field, registration, `config.toml` table).
- Replying to every message instead of filtering (spam in the group).
- Blocking work (HTTP calls, sleeps) without a timeout. Use `http.Client{Timeout: ...}` or a `context.WithTimeout`, since a stuck goroutine per message accumulates.
- Goroutines that send on `out` after the handler returns (panic on shutdown).
- A reply that triggers the plugin again. Whether the Signal client delivers the bot's own messages depends on its implementation, which does not exist yet, so guard plugins that react to arbitrary text.
- Unrecovered panics (crash the whole bot).

## Current limits of the project

`main.go` still passes `nil` as the `SignalClient` (marked `TODO: Needs real SignalClient adapter`), so `bot.Run` cannot run end to end yet and will panic on the nil client. Verify plugins with unit tests rather than by running the bot, and do not implement the Signal adapter unless the user asks. If the repo has moved on from what this skill describes (a real client, a different plugin layout, new registration helpers), read the current `main.go` and an existing plugin and follow them over this document.
