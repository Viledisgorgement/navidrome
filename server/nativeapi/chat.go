package nativeapi

import (
	"encoding/json"
	"errors"
	"image"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
)

const (
	maxChatMessageLen  = 4000
	defaultChatPage    = 50
	maxChatPage        = 200
	chatImageCacheCtrl = "private, max-age=31536000, immutable"
)

func (api *Router) addChatRoutes(r chi.Router) {
	r.Get("/chat", api.getChatMessages)
	r.Post("/chat", api.postChatMessage)
	r.Get("/chat/image/{id}", api.getChatImage)
	r.Delete("/chat/{id}", api.deleteChatMessage)
}

// getChatMessages returns messages newest-first. Pass before=<unix millis>
// to page backwards in time.
func (api *Router) getChatMessages(w http.ResponseWriter, r *http.Request) {
	count := intParam(r, "count")
	if count <= 0 {
		count = defaultChatPage
	}
	count = min(count, maxChatPage)
	before := time.Time{}
	if ms, err := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64); err == nil && ms > 0 {
		before = time.UnixMilli(ms)
	}
	msgs, err := api.ds.ChatMessage(r.Context()).GetRecent(count, before)
	if err != nil {
		log.Error(r.Context(), "Error retrieving chat messages", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if msgs == nil {
		msgs = []model.ChatMessage{}
	}
	writeJSON(w, r, msgs)
}

// postChatMessage accepts either a JSON body {"message": "..."} or a
// multipart form with a "message" field and an optional "image" file.
func (api *Router) postChatMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := request.UserFrom(ctx)
	if !ok {
		http.Error(w, "no user", http.StatusUnauthorized)
		return
	}

	msg := model.ChatMessage{
		ID:        id.NewRandom(),
		UserID:    user.ID,
		UserName:  user.UserName,
		CreatedAt: time.Now(),
	}

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/") {
		maxSize := maxImageUploadSize()
		r.Body = http.MaxBytesReader(w, r.Body, maxSize+64*1024)
		if err := r.ParseMultipartForm(min(maxSize, 10<<20)); err != nil {
			log.Error(ctx, "Error parsing chat multipart form", err)
			http.Error(w, "image too large or invalid form", http.StatusBadRequest)
			return
		}
		defer func() {
			if r.MultipartForm != nil {
				_ = r.MultipartForm.RemoveAll()
			}
		}()
		msg.Message = r.FormValue("message")
		file, header, err := r.FormFile("image")
		switch {
		case err == nil:
			defer file.Close()
			filename, imgErr := api.saveChatImage(w, r, msg.ID, file, header.Filename)
			if imgErr != nil {
				return // response already written
			}
			msg.Image = filename
		case errors.Is(err, http.ErrMissingFile):
			// text-only message
		default:
			log.Error(ctx, "Error reading chat image", err)
			http.Error(w, "invalid image upload", http.StatusBadRequest)
			return
		}
	} else {
		var body struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		msg.Message = body.Message
	}

	msg.Message = strings.TrimSpace(msg.Message)
	if msg.Message == "" && msg.Image == "" {
		http.Error(w, "empty message", http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(msg.Message) > maxChatMessageLen {
		http.Error(w, "message too long", http.StatusBadRequest)
		return
	}

	if err := api.ds.ChatMessage(ctx).Add(msg); err != nil {
		log.Error(ctx, "Error saving chat message", err)
		if msg.Image != "" {
			_ = api.imgUpload.RemoveImage(ctx, model.UploadedImagePath(consts.EntityChat, msg.Image))
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	msg.HasImage = msg.Image != ""

	api.broker.SendBroadcastMessage(ctx, &events.ChatMessage{
		ID:        msg.ID,
		UserID:    msg.UserID,
		UserName:  msg.UserName,
		Message:   msg.Message,
		HasImage:  msg.HasImage,
		CreatedAt: msg.CreatedAt,
	})
	log.Debug(ctx, "Chat message posted", "user", msg.UserName, "hasImage", msg.HasImage)
	writeJSON(w, r, msg)
}

// saveChatImage validates the uploaded file is a real image and stores it
// under the chat images folder. On failure it writes the error response and
// returns a non-nil error.
func (api *Router) saveChatImage(w http.ResponseWriter, r *http.Request, msgID string, file io.ReadSeeker, originalName string) (string, error) {
	ctx := r.Context()
	_, format, err := image.DecodeConfig(file)
	if err != nil {
		log.Error(ctx, "Chat upload is not a valid image", err)
		http.Error(w, "invalid image file", http.StatusBadRequest)
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		log.Error(ctx, "Error seeking chat image", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return "", err
	}
	ext := "." + format
	if ext == "." {
		ext = strings.ToLower(filepath.Ext(originalName))
	}
	if ext == "" || ext == "." {
		http.Error(w, "could not determine image type", http.StatusBadRequest)
		return "", errors.New("unknown image type")
	}
	filename, err := api.imgUpload.SetImage(ctx, consts.EntityChat, msgID, "", "", file, ext)
	if err != nil {
		log.Error(ctx, "Error saving chat image", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return "", err
	}
	return filename, nil
}

func (api *Router) getChatImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	msg, err := api.ds.ChatMessage(ctx).Get(chi.URLParam(r, "id"))
	if err != nil || msg.Image == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Cache-Control", chatImageCacheCtrl)
	http.ServeFile(w, r, model.UploadedImagePath(consts.EntityChat, msg.Image))
}

// deleteChatMessage removes a message (and its image). Users can delete
// their own messages; admins can delete any.
func (api *Router) deleteChatMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := request.UserFrom(ctx)
	if !ok {
		http.Error(w, "no user", http.StatusUnauthorized)
		return
	}
	repo := api.ds.ChatMessage(ctx)
	msg, err := repo.Get(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if msg.UserID != user.ID && !user.IsAdmin {
		http.Error(w, "not authorized", http.StatusForbidden)
		return
	}
	if err := repo.Delete(msg.ID); err != nil {
		log.Error(ctx, "Error deleting chat message", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if msg.Image != "" {
		if err := api.imgUpload.RemoveImage(ctx, model.UploadedImagePath(consts.EntityChat, msg.Image)); err != nil {
			log.Warn(ctx, "Error removing chat image file", "id", msg.ID, err)
		}
	}
	api.broker.SendBroadcastMessage(ctx, &events.ChatMessage{ID: msg.ID, Deleted: true})
	w.WriteHeader(http.StatusNoContent)
}
