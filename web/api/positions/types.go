package positions

import "github.com/logicorg/ivfs/web/api/gateway"

// DeletePositionMovementInput is the request of DELETE
// /api/position-movements/{id}.
type DeletePositionMovementInput struct {
	ID int64 `path:"id" minimum:"1"`
}

// DeletePositionMovementOutput is the response of DELETE
// /api/position-movements/{id}.
type DeletePositionMovementOutput struct {
	Body gateway.Deletion
}
