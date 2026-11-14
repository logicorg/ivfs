package tagtypes

import "github.com/logicorg/ivfs/pkg/model"

// RetrieveTagParams input to a retrieve tag operation
type RetrieveTagParams struct {
	ID   int64
	Name string
}

// RetrieveTagResult result of a retrieve tag operation
type RetrieveTagResult struct {
	Tag *model.Tag
}

type ListTagsParams struct{}

type ListTagsResult struct {
	Tags []*model.Tag
}
