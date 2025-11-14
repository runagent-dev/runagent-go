package client

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
)

// StreamIterator handles streaming responses
type StreamIterator struct {
	conn       *websocket.Conn
	serializer *CoreSerializer
	finished   bool
	err        error
}

// NewStreamIterator creates a new stream iterator
func NewStreamIterator(conn *websocket.Conn, serializer *CoreSerializer) *StreamIterator {
	return &StreamIterator{
		conn:       conn,
		serializer: serializer,
	}
}

// Next returns the next item from the stream
func (s *StreamIterator) Next(ctx context.Context) (interface{}, bool, error) {
	if s.finished || s.err != nil {
		return nil, false, s.err
	}

	select {
	case <-ctx.Done():
		s.finished = true
		s.conn.Close()
		return nil, false, ctx.Err()
	default:
	}

	_, messageData, err := s.conn.ReadMessage()
	if err != nil {
		s.finished = true
		s.err = fmt.Errorf("failed to read WebSocket message: %w", err)
		return nil, false, s.err
	}

	msg, err := s.serializer.DeserializeMessage(string(messageData))
	if err != nil {
		s.finished = true
		s.err = fmt.Errorf("failed to deserialize message: %w", err)
		return nil, false, s.err
	}

	if msg.Error != "" {
		s.finished = true
		s.err = fmt.Errorf("stream error: %s", msg.Error)
		return nil, false, s.err
	}

	switch msg.Type {
	case "status":
		switch msg.Status {
		case "stream_completed":
			s.finished = true
			return nil, false, nil
		case "stream_started":
			return s.Next(ctx)
		default:
			return s.Next(ctx)
		}
	case "data":
		return msg.Content, true, nil
	case "error":
		s.finished = true
		s.err = fmt.Errorf("agent error: %v", msg.Error)
		return nil, false, s.err
	default:
		return s.Next(ctx)
	}
}

// Close closes the stream iterator
func (s *StreamIterator) Close() error {
	s.finished = true
	return s.conn.Close()
}
