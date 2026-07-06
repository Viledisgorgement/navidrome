package nativeapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
)

func (api *Router) addCommunityRoute(r chi.Router) {
	r.Route("/community", func(r chi.Router) {
		r.Get("/recent", api.getCommunityRecent)
		r.Get("/top", api.getCommunityTop)
		r.Get("/users", api.getCommunityUsers)
		// Alerts are implemented with the release-alerts feature; these
		// stubs keep the UI contract stable in the meantime.
		r.Get("/alerts", getCommunityAlertsStub)
		r.Post("/alerts/seen", postCommunityAlertsSeenStub)
	})
}

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

func getCommunityAlertsStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, map[string]any{"alerts": []any{}, "unreadCount": 0})
}

func postCommunityAlertsSeenStub(w http.ResponseWriter, _ *http.Request) {
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
