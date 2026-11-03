package asset

import "github.com/logicorg/ivfs/internal/pkg/tenancy"

// AssetActivity holds the activities to deal wth assets
type AssetActivity struct { //nolint:revive
	Pools tenancy.Pools
}
