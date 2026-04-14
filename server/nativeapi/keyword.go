package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	taglib "github.com/navidrome/navidrome/adapters/gotaglib"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/server"
)

type renameKeywordRequest struct {
	Name string `json:"name"`
}

func (api *Router) addKeywordRoute(r chi.Router) {
	keywordRepo := persistence.NewKeywordRepository(api.ds)
	if keywordRepo == nil {
		return
	}
	r.Route("/keyword", func(r chi.Router) {
		r.Get("/", rest.GetAll(keywordRepo))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(keywordRepo))
			r.Put("/", renameKeyword(api.ds))
			r.Delete("/", deleteKeyword(api.ds))
		})
	})
}

func renameKeyword(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := chi.URLParam(r, "id")

		var req renameKeywordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		newName := strings.TrimSpace(req.Name)
		if newName == "" {
			http.Error(w, "Name cannot be empty", http.StatusBadRequest)
			return
		}

		tag, err := lookupTag(ds, ctx, id)
		if err != nil {
			writeTagLookupError(w, err)
			return
		}
		oldName := tag.TagValue

		if oldName == newName {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tag)
			return
		}

		mediaFiles, err := ds.MediaFile(ctx).GetAllByTags(model.TagKeyword, []string{oldName})
		if err != nil {
			log.Error(ctx, "Failed to get media files for keyword rename", "keyword", oldName, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		for i := range mediaFiles {
			mf := &mediaFiles[i]
			keywords := mf.Tags.Values(model.TagKeyword)
			var updated []string
			for _, k := range keywords {
				if k == oldName {
					if !slices.Contains(updated, newName) {
						updated = append(updated, newName)
					}
				} else {
					updated = append(updated, k)
				}
			}

			fullPath := filepath.Join(mf.LibraryPath, mf.Path)
			if err := taglib.WriteTag(fullPath, "KEYWORDS", strings.Join(updated, ";")); err != nil {
				log.Error(ctx, "Failed to write keywords tag during rename", "path", fullPath, err)
				continue
			}

			if mf.Tags == nil {
				mf.Tags = make(model.Tags)
			}
			mf.Tags[model.TagKeyword] = updated
			mf.UpdatedAt = time.Now()
			if err := ds.MediaFile(ctx).Put(mf); err != nil {
				log.Error(ctx, "Failed to update media file during keyword rename", "id", mf.ID, err)
			}
		}

		// Register the new tag value for autocomplete
		newTag := model.NewTag(model.TagKeyword, newName)
		if len(mediaFiles) > 0 {
			_ = ds.Tag(ctx).Add(mediaFiles[0].LibraryID, newTag)
		}

		if err := ds.Tag(ctx).UpdateCounts(); err != nil {
			log.Error(ctx, "Failed to update tag counts after keyword rename", err)
		}

		log.Info(ctx, "Renamed keyword", "old", oldName, "new", newName, "tracks", len(mediaFiles))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": newTag.ID, "oldName": oldName, "newName": newName})
	}
}

func deleteKeyword(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := chi.URLParam(r, "id")

		tag, err := lookupTag(ds, ctx, id)
		if err != nil {
			writeTagLookupError(w, err)
			return
		}
		keyword := tag.TagValue

		mediaFiles, err := ds.MediaFile(ctx).GetAllByTags(model.TagKeyword, []string{keyword})
		if err != nil {
			log.Error(ctx, "Failed to get media files for keyword delete", "keyword", keyword, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		for i := range mediaFiles {
			mf := &mediaFiles[i]
			keywords := mf.Tags.Values(model.TagKeyword)
			var updated []string
			for _, k := range keywords {
				if k != keyword {
					updated = append(updated, k)
				}
			}

			fullPath := filepath.Join(mf.LibraryPath, mf.Path)
			if err := taglib.WriteTag(fullPath, "KEYWORDS", strings.Join(updated, ";")); err != nil {
				log.Error(ctx, "Failed to write keywords tag during delete", "path", fullPath, err)
				continue
			}

			if mf.Tags == nil {
				mf.Tags = make(model.Tags)
			}
			if len(updated) == 0 {
				delete(mf.Tags, model.TagKeyword)
			} else {
				mf.Tags[model.TagKeyword] = updated
			}
			mf.UpdatedAt = time.Now()
			if err := ds.MediaFile(ctx).Put(mf); err != nil {
				log.Error(ctx, "Failed to update media file during keyword delete", "id", mf.ID, err)
			}
		}

		if err := ds.Tag(ctx).UpdateCounts(); err != nil {
			log.Error(ctx, "Failed to update tag counts after keyword delete", err)
		}

		log.Info(ctx, "Deleted keyword", "keyword", keyword, "tracks", len(mediaFiles))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id, "keyword": keyword})
	}
}

func lookupTag(ds model.DataStore, ctx context.Context, id string) (*model.Tag, error) {
	repo := ds.Resource(ctx, model.Tag{})
	result, err := repo.Read(id)
	if err != nil {
		return nil, err
	}
	return result.(*model.Tag), nil
}

func writeTagLookupError(w http.ResponseWriter, err error) {
	if errors.Is(err, model.ErrNotFound) {
		http.Error(w, "Keyword not found", http.StatusNotFound)
		return
	}
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}
