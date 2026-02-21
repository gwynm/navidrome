package nativeapi

import (
	"context"
	"errors"
	"maps"
	"net/http"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/req"
)

type potentialTrashRepository struct {
	model.ResourceRepository
	mfRepo model.MediaFileRepository
}

func newPotentialTrashRepository(ds model.DataStore) rest.RepositoryConstructor {
	return func(ctx context.Context) rest.Repository {
		return &potentialTrashRepository{
			mfRepo:             ds.MediaFile(ctx),
			ResourceRepository: ds.Resource(ctx, model.MediaFile{}),
		}
	}
}

func (r *potentialTrashRepository) Count(options ...rest.QueryOptions) (int64, error) {
	opt := r.parseOptions(options)
	return r.ResourceRepository.Count(opt)
}

func (r *potentialTrashRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	opt := r.parseOptions(options)
	return r.ResourceRepository.ReadAll(opt)
}

func (r *potentialTrashRepository) parseOptions(options []rest.QueryOptions) rest.QueryOptions {
	var opt rest.QueryOptions
	if len(options) > 0 {
		opt = options[0]
		opt.Filters = maps.Clone(opt.Filters)
	}
	if opt.Filters == nil {
		opt.Filters = make(map[string]any)
	}
	opt.Filters["potential_trash"] = "true"
	return opt
}

func (r *potentialTrashRepository) EntityName() string {
	return "potential_trash"
}

func deletePotentialTrashFiles(maintenance core.Maintenance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		p := req.Params(r)
		ids, _ := p.Strings("id")

		err := maintenance.DeletePotentialTrash(ctx, ids)

		if len(ids) == 1 && errors.Is(err, model.ErrNotFound) {
			log.Warn(ctx, "Potential trash file not found", "id", ids[0])
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Error(ctx, "Failed to delete potential trash files", err)
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		writeDeleteManyResponse(w, r, ids)
	}
}

var _ model.ResourceRepository = &potentialTrashRepository{}
