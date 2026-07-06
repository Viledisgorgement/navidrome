package nativeapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

func (api *Router) addCommunityRoute(r chi.Router) {
	r.Route("/community", func(r chi.Router) {
		r.Get("/recent", api.getCommunityRecent)
		r.Get("/top", api.getCommunityTop)
		r.Get("/users", api.getCommunityUsers)
		r.Get("/alerts", api.getCommunityAlerts)
		r.Post("/alerts/seen", api.postCommunityAlertsSeen)
	})
}

const alertsSeenAtKey = "communityAlertsSeenAt"

func (api *Router) getCommunityRecent(w http.ResponseWriter, r *http.Request) {
	entries, err := api.community.Recent(r.Context(),
		intParam(r, "count"), intParam(r, "offset"), usersParam(r))
	if err != nil {
		log.Error(r.Context(), "Error retrieving community play feed", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, entries)
}

func (api *Router) getCommunityTop(w http.ResponseWriter, r *http.Request) {
	entries, err := api.community.Top(r.Context(),
		r.URL.Query().Get("type"), r.URL.Query().Get("range"),
		intParam(r, "count"), usersParam(r))
	if err != nil {
		log.Error(r.Context(), "Error retrieving community top played", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, entries)
}

func (api *Router) getCommunityUsers(w http.ResponseWriter, r *http.Request) {
	users, err := api.community.Users(r.Context())
	if err != nil {
		log.Error(r.Context(), "Error retrieving community users", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, users)
}

// getCommunityAlerts returns release alerts newest-first plus the caller's
// unread count, computed against their personal "seen" watermark stored in
// user_props.
func (api *Router) getCommunityAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	count := intParam(r, "count")
	if count <= 0 {
		count = 50
	}
	alerts, err := api.ds.ReleaseAlert(ctx).GetAll(count)
	if err != nil {
		log.Error(ctx, "Error retrieving release alerts", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	unread := int64(0)
	if user, ok := request.UserFrom(ctx); ok {
		watermark := time.Time{}
		if raw, err := api.ds.UserProps(ctx).Get(user.ID, alertsSeenAtKey); err == nil {
			if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
				watermark = time.Unix(seconds, 0)
			}
		}
		unread, err = api.ds.ReleaseAlert(ctx).CountSince(watermark)
		if err != nil {
			log.Error(ctx, "Error counting unread alerts", err)
		}
	}
	if alerts == nil {
		alerts = []model.ReleaseAlert{}
	}
	writeJSON(w, r, map[string]any{"alerts": alerts, "unreadCount": unread})
}

func (api *Router) postCommunityAlertsSeen(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := request.UserFrom(ctx)
	if !ok {
		http.Error(w, "no user", http.StatusUnauthorized)
		return
	}
	err := api.ds.UserProps(ctx).Put(user.ID, alertsSeenAtKey, strconv.FormatInt(time.Now().Unix(), 10))
	if err != nil {
		log.Error(ctx, "Error saving alerts watermark", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, r *http.Request, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error(r.Context(), "Error encoding response", err)
	}
}

func intParam(r *http.Request, name string) int {
	v, _ := strconv.Atoi(r.URL.Query().Get(name))
	return v
}

func usersParam(r *http.Request) []string {
	raw := r.URL.Query().Get("users")
	if raw == "" {
		return nil
	}
	var ids []string
	for _, id := range strings.Split(raw, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
