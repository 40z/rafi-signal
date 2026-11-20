package rafibot

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
)

type SocketSignalClientAdapterOptions struct {
	Socket string
}

type SocketSignalClientAdapter struct {
	logger      *slog.Logger
	options     SocketSignalClientAdapterOptions
	conn        net.Conn
	sendEncoder *json.Encoder
}

func NewSocketSignalClient(logger *slog.Logger, options SocketSignalClientAdapterOptions) SocketSignalClientAdapter {
	return SocketSignalClientAdapter{
		logger:  logger,
		options: options,
	}
}

func (s *SocketSignalClientAdapter) Start(msgChan chan<- Message) {
	s.logger.Debug("Socket signal client starting")

	conn, err := net.Dial("unix", s.options.Socket)
	if err != nil {
		s.logger.Error("Socket dial failed", "err", err)
		panic(err)
	}
	s.conn = conn
	s.sendEncoder = json.NewEncoder(conn)

	s.logger.Info("Connected to local socket. Listening for incoming messages...")

	go func() {
		scanner := bufio.NewScanner(conn)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}

			var msg struct {
				Sender   string
				Body     string
				Mentions []string
			}

			if err := json.Unmarshal(line, &msg); err != nil {
				s.logger.Debug("Received buffer line could not be deserialized as structured message; treating as string")
				msg.Body = string(line)
			}

			s.logger.Debug("Received message", "msg", msg)

			message := Message{
				Sender:   msg.Sender,
				Body:     msg.Body,
				Mentions: map[string]struct{}{},
			}

			for _, e := range msg.Mentions {
				message.Mentions[e] = struct{}{}
			}

			msgChan <- message
		}

		err := scanner.Err()
		if err == nil {
			s.logger.Warn("Socket connection closed cleanly; the bot may still be running but won't receive any messages")
		} else if errors.Is(err, net.ErrClosed) {
			s.logger.Info("Socket connection closed")
		} else {
			s.logger.Error("Socket read error", "err", err)
			panic(err)
		}
	}()
}

func (s *SocketSignalClientAdapter) Stop() {
	s.logger.Debug("Socket signal client stopping")
	s.conn.Close()
}

func (s *SocketSignalClientAdapter) SendMessage(msg string) error {
	s.logger.Debug("Sending message", "msg", msg)
	return s.sendEncoder.Encode(msg)
}
