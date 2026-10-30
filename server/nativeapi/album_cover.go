package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/coverart"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
)

// Where a changed cover was saved
const (
	coverSavedToFolder   = "folder"   // cover.<ext> written into the album folder
	coverSavedAsOverride = "override" // folder not writable: stored under the data folder
)

func (api *Router) addAlbumRoute(r chi.Router) {
	constructor := func(ctx context.Context) rest.Repository {
		return api.ds.Resource(ctx, model.Album{})
	}
	r.Route("/album", func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			r.Get("/cover/search", api.searchAlbumCover)
			r.Post("/cover", api.setAlbumCoverFromURL)
			r.Post("/image", api.uploadAlbumImage())
			r.Delete("/image", api.deleteAlbumImage())
		})
	})
}

func (api *Router) searchAlbumCover(w http.ResponseWriter, r *http.Request) {
	if !checkImageUploadPermission(w, r) {
		return
	}
	artist := strings.TrimSpace(r.URL.Query().Get("artist"))
	album := strings.TrimSpace(r.URL.Query().Get("album"))
	if album == "" {
		http.Error(w, "album is required", http.StatusBadRequest)
		return
	}
	res := coverart.Search(r.Context(), artist, album)
	if res == nil {
		res = []coverart.Candidate{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (api *Router) setAlbumCoverFromURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !checkImageUploadPermission(w, r) {
		return
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&body); err != nil || body.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	data, ext, err := coverart.Download(ctx, body.URL, maxImageUploadSize())
	if err != nil {
		log.Warn(ctx, "Could not download cover image", "url", body.URL, err)
		http.Error(w, "could not download image: "+err.Error(), http.StatusBadRequest)
		return
	}
	savedTo, err := api.saveAlbumCover(ctx, chi.URLParamFromCtx(ctx, "id"), data, ext)
	if errors.Is(err, model.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error(ctx, "Error saving album cover", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "savedTo": savedTo})
}

func (api *Router) uploadAlbumImage() http.HandlerFunc {
	return handleImageUpload(func(ctx context.Context, reader io.Reader, ext string) error {
		data, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		_, err = api.saveAlbumCover(ctx, chi.URLParamFromCtx(ctx, "id"), data, coverart.ImageExt(strings.TrimPrefix(ext, ".")))
		return err
	})
}

// deleteAlbumImage removes a cover stored as an override. Covers written to
// the album folder are files in the library and are left for the user to manage.
func (api *Router) deleteAlbumImage() http.HandlerFunc {
	return handleImageDelete(func(ctx context.Context) error {
		al, err := api.ds.Album(ctx).Get(chi.URLParamFromCtx(ctx, "id"))
		if err != nil {
			return err
		}
		if err := api.imgUpload.RemoveImage(ctx, al.UploadedImagePath()); err != nil {
			return err
		}
		return api.ds.Album(ctx).SetCover(al.ID, "")
	})
}

// saveAlbumCover writes the image as cover.<ext> in the album folder, like
// Airsonic, then rescans that folder so Navidrome picks it up. If the folder
// cannot be written (e.g. the library is mounted read-only), the image is
// stored under the data folder as an override instead.
func (api *Router) saveAlbumCover(ctx context.Context, albumID string, data []byte, ext string) (string, error) {
	al, err := api.ds.Album(ctx).Get(albumID)
	if err != nil {
		return "", err
	}
	folder, err := api.albumCoverFolder(ctx, *al)
	if err != nil {
		return "", err
	}

	path, err := coverart.WriteFolderCover(folder.AbsolutePath(), data, ext)
	if err == nil {
		log.Info(ctx, "Saved album cover to album folder", "album", al.Name, "path", path)
		if err := api.imgUpload.RemoveImage(ctx, al.UploadedImagePath()); err != nil {
			log.Warn(ctx, "Could not remove previous cover override", "album", al.Name, err)
		}
		api.rescanCoverFolder(ctx, *folder)
		return coverSavedToFolder, api.ds.Album(ctx).SetCover(al.ID, "")
	}
	if !isNotWritable(err) {
		return "", err
	}

	log.Warn(ctx, "Album folder is not writable, saving cover as an override in the data folder",
		"album", al.Name, "folder", folder.AbsolutePath(), err)
	filename, err := api.imgUpload.SetImage(ctx, consts.EntityAlbum, al.ID, al.Name, al.UploadedImagePath(), bytes.NewReader(data), ext)
	if err != nil {
		return "", err
	}
	return coverSavedAsOverride, api.ds.Album(ctx).SetCover(al.ID, filename)
}

// albumCoverFolder picks the folder the cover belongs in: the album's folder,
// or for albums split across disc subfolders, their shared parent folder.
func (api *Router) albumCoverFolder(ctx context.Context, al model.Album) (*model.Folder, error) {
	folders, err := api.ds.Folder(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"folder.id": al.FolderIDs, "missing": false},
	})
	if err != nil {
		return nil, err
	}
	if len(folders) == 0 {
		return nil, model.ErrNotFound
	}
	if len(folders) > 1 {
		parentID := folders[0].ParentID
		sameParent := parentID != ""
		for _, f := range folders[1:] {
			sameParent = sameParent && f.ParentID == parentID
		}
		if sameParent {
			parent, err := api.ds.Folder(ctx).Get(parentID)
			if err == nil && parent.ParentID != "" { // never the library root
				return parent, nil
			}
		}
	}
	return &folders[0], nil
}

func (api *Router) rescanCoverFolder(ctx context.Context, folder model.Folder) {
	if api.scanner == nil {
		return
	}
	target := model.ScanTarget{LibraryID: folder.LibraryID, FolderPath: filepath.ToSlash(filepath.Join(folder.Path, folder.Name))}
	// Wait for the scan, so the next cover request already sees the new file
	if _, err := api.scanner.ScanFolders(context.WithoutCancel(ctx), false, []model.ScanTarget{target}); err != nil {
		log.Warn(ctx, "Could not rescan album folder after changing its cover; it will be picked up by the next scan",
			"folder", target.String(), err)
	}
}

func isNotWritable(err error) bool {
	return errors.Is(err, syscall.EROFS) || errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM)
}
