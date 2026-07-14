package events

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"time"
	"unicode"
)

type eventCtxKey string

const broadcastToAllKey eventCtxKey = "broadcastToAll"

// broadcastToAll is a context key that can be used to broadcast an event to all clients
func broadcastToAll(ctx context.Context) context.Context {
	return context.WithValue(ctx, broadcastToAllKey, true)
}

type Event interface {
	Name(Event) string
	Data(Event) string
}

type baseEvent struct{}

func (e *baseEvent) Name(evt Event) string {
	str := strings.TrimPrefix(reflect.TypeOf(evt).String(), "*events.")
	return str[:0] + string(unicode.ToLower(rune(str[0]))) + str[1:]
}

func (e *baseEvent) Data(evt Event) string {
	data, _ := json.Marshal(evt)
	return string(data)
}

type ScanStatus struct {
	baseEvent
	Scanning    bool          `json:"scanning"`
	Count       int64         `json:"count"`
	FolderCount int64         `json:"folderCount"`
	Error       string        `json:"error"`
	ScanType    string        `json:"scanType"`
	ElapsedTime time.Duration `json:"elapsedTime"`
}

type KeepAlive struct {
	baseEvent
	TS int64 `json:"ts"`
}

type ServerStart struct {
	baseEvent
	StartTime time.Time `json:"startTime"`
	Version   string    `json:"version"`
}

const Any = "*"

type RefreshResource struct {
	baseEvent
	resources map[string][]string
}

type NowPlayingCount struct {
	baseEvent
	Count int `json:"count"`
}

type PlayEvent struct {
	baseEvent
	UserName  string    `json:"userName"`
	SongID    string    `json:"songId"`
	Title     string    `json:"title"`
	Artist    string    `json:"artist"`
	AlbumID   string    `json:"albumId"`
	Timestamp time.Time `json:"timestamp"`
}

type ReleaseAlert struct {
	baseEvent
	Count int `json:"count"`
}

// ChatMessage announces a new (or deleted) community chat message to all
// connected clients.
type ChatMessage struct {
	baseEvent
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	UserName  string    `json:"userName"`
	Message   string    `json:"message"`
	HasImage  bool      `json:"hasImage"`
	CreatedAt time.Time `json:"createdAt"`
	Deleted   bool      `json:"deleted,omitempty"`
}

func (rr *RefreshResource) With(resource string, ids ...string) *RefreshResource {
	if rr.resources == nil {
		rr.resources = make(map[string][]string)
	}
	if len(ids) == 0 {
		rr.resources[resource] = append(rr.resources[resource], Any)
	}
	rr.resources[resource] = append(rr.resources[resource], ids...)
	return rr
}

func (rr *RefreshResource) Data(evt Event) string {
	if rr.resources == nil {
		return `{"*":"*"}`
	}
	r := evt.(*RefreshResource)
	data, _ := json.Marshal(r.resources)
	return string(data)
}
