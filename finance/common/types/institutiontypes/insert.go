package institutiontypes

import (
	"time"

	"cuelang.org/go/cuego"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

type InsertInstitutionParams struct {
	Name     string     `json:"name" cue:"!~\"^[[:space:]]*$\""`
	Kind     string     `json:"kind" cue:"!~\"^[[:space:]]*$\""`
	OpenedAt *time.Time `json:"openedAt,omitempty"`
	ClosedAt *time.Time `json:"closedAt,omitempty"`
}

func (i *InsertInstitutionParams) Validate() error {
	err := cuego.Validate(i)
	if err != nil {
		return errors.ValidationError{Err: err}
	}

	return nil
}

func (i *InsertInstitutionParams) ToModel() *model.Institution {
	return &model.Institution{
		Name:     i.Name,
		Kind:     i.Kind,
		OpenedAt: i.OpenedAt,
		ClosedAt: i.ClosedAt,
	}
}

type InsertInstitutionResult struct {
	ID int64 `json:"id"`
}
