package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
)

type keywordTagRepository struct {
	sqlRepository
}

func NewKeywordRepository(ds model.DataStore) rest.RepositoryConstructor {
	store, ok := ds.(*SQLStore)
	if !ok {
		return nil
	}
	return func(ctx context.Context) rest.Repository {
		r := &keywordTagRepository{}
		r.ctx = ctx
		r.db = store.getDBXBuilder()
		r.tableName = "tag"
		r.registerModel(&model.Tag{}, map[string]filterFunc{
			"name": containsFilter("tag_value"),
		})
		r.setSortMappings(map[string]string{
			"name": "tag_value",
		})
		return r
	}
}

func (r *keywordTagRepository) newKeywordSelect(options ...model.QueryOptions) SelectBuilder {
	sq := r.sqlRepository.newSelect(options...).
		Where(Eq{"tag.tag_name": "keyword"}).
		Columns(
			"tag.id",
			"tag.tag_name",
			"tag.tag_value",
			"0 as album_count",
			`(SELECT COUNT(DISTINCT mf.id) FROM media_file mf
			  WHERE EXISTS (
			    SELECT 1 FROM json_tree(mf.tags, '$.keyword') jt
			    WHERE jt.key = 'value' AND jt.value = tag.tag_value
			  )) as song_count`,
		)

	return sq
}

func (r *keywordTagRepository) Count(options ...rest.QueryOptions) (int64, error) {
	sq := Select("COUNT(*)").From("tag").Where(Eq{"tag.tag_name": "keyword"})
	return r.count(sq, r.parseRestOptions(r.ctx, options...))
}

func (r *keywordTagRepository) Read(id string) (any, error) {
	query := r.newKeywordSelect().Where(Eq{"tag.id": id})
	var res model.Tag
	err := r.queryOne(query, &res)
	return &res, err
}

func (r *keywordTagRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	query := r.newKeywordSelect(r.parseRestOptions(r.ctx, options...))
	var res model.TagList
	err := r.queryAll(query, &res)
	return res, err
}

func (r *keywordTagRepository) EntityName() string {
	return "keyword"
}

func (r *keywordTagRepository) NewInstance() any {
	return model.Tag{}
}

var _ model.ResourceRepository = (*keywordTagRepository)(nil)
