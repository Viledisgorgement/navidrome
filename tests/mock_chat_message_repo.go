package tests

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockChatMessageRepo struct {
	Messages []model.ChatMessage
	Err      error
}

func (m *MockChatMessageRepo) Add(msg model.ChatMessage) error {
	if m.Err != nil {
		return m.Err
	}
	m.Messages = append(m.Messages, msg)
	return nil
}

func (m *MockChatMessageRepo) GetRecent(limit int, before time.Time) ([]model.ChatMessage, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var res []model.ChatMessage
	for i := len(m.Messages) - 1; i >= 0 && len(res) < limit; i-- {
		msg := m.Messages[i]
		if !before.IsZero() && !msg.CreatedAt.Before(before) {
			continue
		}
		res = append(res, msg)
	}
	return res, nil
}

func (m *MockChatMessageRepo) Get(id string) (*model.ChatMessage, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	for _, msg := range m.Messages {
		if msg.ID == id {
			return &msg, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *MockChatMessageRepo) Delete(id string) error {
	if m.Err != nil {
		return m.Err
	}
	for i, msg := range m.Messages {
		if msg.ID == id {
			m.Messages = append(m.Messages[:i], m.Messages[i+1:]...)
			return nil
		}
	}
	return model.ErrNotFound
}
