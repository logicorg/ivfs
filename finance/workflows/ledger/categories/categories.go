// Package categories holds the category workflows, one per activity: a
// category is an entity, with nothing underneath to compose.
package categories

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/categories"
)

// ca is a nil pointer to the category activities, used to name them.
var ca = (*activity.Activities)(nil)
