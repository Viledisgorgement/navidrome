package nativeapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func (api *Router) getArtistDiscography(w http.ResponseWriter, r *http.Request) {
	artistID := chi.URLParam(r, "id")
	refresh := r.URL.Query().Get("refresh") == "true"
	discography, err := api.community.Discography(r.Context(), artistID, refresh)
	if err == model.ErrNotFound {
		http.Error(w, "artist not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error(r.Context(), "Error retrieving discography", "artistId", artistID, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, discography)
}
