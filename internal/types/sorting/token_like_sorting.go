package sorting

import "artion-api-graphql/internal/types"

type TokenLikeSorting int8

const (
	TokenLikeSortingCreated TokenLikeSorting = iota
)

func (ts TokenLikeSorting) SortedFieldBson() string {
	return "created"
}

func (ts TokenLikeSorting) OrdinalFieldBson() string {
	return "_id"
}

func (ts TokenLikeSorting) GetCursor(like *types.TokenLike) (types.Cursor, error) {
	params := make(map[string]interface{})
	params["_id"] = like.ID()
	params["created"] = like.Created
	return CursorFromParams(params)
}
