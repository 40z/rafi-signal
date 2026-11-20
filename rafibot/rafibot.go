package rafibot

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/robfig/cron/v3"
)

type SignalClient interface {
	Start(chan<- Message)
	Stop()
	SendMessage(string) error
}

type Message struct {
	Sender   string
	Body     string
	Mentions map[string]struct{}
}

type MessageHandler = func(Message, chan<- string)
type ScheduleHandler = func(chan<- string)

type RafiBot struct {
	logger          *slog.Logger
	messageOut      chan string
	messageIn       chan Message
	messageHandlers []MessageHandler
	signalClient    SignalClient
	scheduler       *cron.Cron
	name            string
}

func New(logger *slog.Logger, signal SignalClient, name string) RafiBot {
	return RafiBot{
		logger:       logger,
		messageOut:   make(chan string, 20),
		messageIn:    make(chan Message),
		signalClient: signal,
		scheduler:    cron.New(cron.WithSeconds()),
		name:         name,
	}
}

func (s *RafiBot) Run(ctx context.Context) {
	// Start signal client
	s.signalClient.Start(s.messageIn)
	defer s.signalClient.Stop()

	// Start scheduler
	s.scheduler.Start()
	defer s.scheduler.Stop()

	// Background go routine listens to context cancellation and closes
	// input/ouput channels. Closing the channels will break the loops below and
	// allow this method to return.
	go func() {
		<-ctx.Done()
		close(s.messageIn)
		close(s.messageOut)
	}()

	// Background go routine listens for messages coming from the plugins and
	// sends them through the signal client.
	go func() {
		for sendMessage := range s.messageOut {
			s.signalClient.SendMessage(sendMessage)
		}
	}()

	// Every received message is distributed to each plugin in a background
	// go routine.
	for receiveMsg := range s.messageIn {
		for _, h := range s.messageHandlers {
			go s.runHandler(h, receiveMsg, s.messageOut)
		}
	}
}

func (s *RafiBot) OnMessage(handler MessageHandler) *RafiBot {
	s.messageHandlers = append(s.messageHandlers, handler)
	return s
}

func (s *RafiBot) OnMention(handler MessageHandler) *RafiBot {
	filteredHandler := func(msg Message, sendChan chan<- string) {
		if _, ok := msg.Mentions[s.name]; ok {
			handler(msg, sendChan)
		}
	}
	return s.OnMessage(filteredHandler)
}

func (s *RafiBot) OnSchedule(cronStr string, handler ScheduleHandler) *RafiBot {
	s.scheduler.AddFunc(cronStr, func() {
		handler(s.messageOut)
	})
	return s
}

func (s *RafiBot) runHandler(h MessageHandler, msg Message, sendChan chan<- string) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Handler panic", "err", r)
			debug.PrintStack()
		}
	}()
	h(msg, sendChan)
}
