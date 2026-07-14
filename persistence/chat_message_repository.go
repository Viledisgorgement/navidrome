package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type chatMessageRepository struct {
	sqlRepository
}

func NewChatMessageRepository(ctx context.Context, db dbx.Builder) model.ChatMessageRepository {
	r := &chatMessageRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "chat_message"
	return r
}

func (r *chatMessageRepository) Add(msg model.ChatMessage) error {
	insert := Insert(r.tableName).SetMap(map[string]any{
		"id":         msg.ID,
		"user_id":    msg.UserID,
		"message":    msg.Message,
		"image":      msg.Image,
		"created_at": msg.CreatedAt.UnixMilli(),
	})
	_, err := r.executeSQL(insert)
	return err
}

type dbChatMessage struct {
	ID        string `db:"id"`
	UserID    string `db:"user_id"`
	UserName  string `db:"user_name"`
	Message   string `db:"message"`
	Image     string `db:"image"`
	CreatedAt int64  `db:"created_at"`
}

func (m dbChatMessage) toModel() model.ChatMessage {
	return model.ChatMessage{
		ID:        m.ID,
		UserID:    m.UserID,
		UserName:  m.UserName,
		Message:   m.Message,
		Image:     m.Image,
		HasImage:  m.Image != "",
		CreatedAt: time.UnixMilli(m.CreatedAt),
	}
}

func (r *chatMessageRepository) GetRecent(limit int, before time.Time) ([]model.ChatMessage, error) {
	sq := Select("c.id", "c.user_id", "u.user_name", "c.message", "c.image", "c.created_at").
		From(r.tableName + " c").
		Join("user u on u.id = c.user_id").
		OrderBy("c.created_at desc, c.id desc").
		Limit(uint64(limit)) //nolint:gosec
	if !before.IsZero() {
		sq = sq.Where(Lt{"c.created_at": before.UnixMilli()})
	}
	var rows []dbChatMessage
	if err := r.queryAll(sq, &rows); err != nil {
		return nil, err
	}
	msgs := make([]model.ChatMessage, len(rows))
	for i, row := range rows {
		msgs[i] = row.toModel()
	}
	return msgs, nil
}

func (r *chatMessageRepository) Get(id string) (*model.ChatMessage, error) {
	sq := Select("c.id", "c.user_id", "u.user_name", "c.message", "c.image", "c.created_at").
		From(r.tableName + " c").
		Join("user u on u.id = c.user_id").
		Where(Eq{"c.id": id})
	var rows []dbChatMessage
	if err := r.queryAll(sq, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, model.ErrNotFound
	}
	msg := rows[0].toModel()
	return &msg, nil
}

func (r *chatMessageRepository) Delete(id string) error {
	affected, err := r.executeSQL(Delete(r.tableName).Where(Eq{"id": id}))
	if err != nil {
		return err
	}
	if affected == 0 {
		return model.ErrNotFound
	}
	return nil
}
