package tags

import (
	"errors"

	"github.com/logicorg/ivfs/finance/activities/tag"
	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// InsertTag is a Temporal workflow entrypoint that orchestrates creating a new tag.
// It validates inputs, sets activity options (including retries), and invokes the
// "tags.InsertTag" activity to perform the actual persistence.
//
// Parameters:
//   - ctx: workflow-scoped context provided by the SDK.
//   - params: inputs required to insert a tag.
//
// Returns:
//   - (*InsertTagResult, error): the inserted tag ID on success or an error.
func InsertTag(ctx workflow.Context, params tt.InsertTagParams) (*tt.InsertTagResult, error) {
	// Fail fast on invalid input to avoid unnecessary activity scheduling.
	if params.Name == "" {
		return nil, errors.New("invalid InsertTagParams: Name must not be empty")
	}

	// Configure conservative activity options and retries.
	ao := workflows.DefaultActivityOptions
	ctx = workflow.WithActivityOptions(ctx, ao)

	tagsActivity := tag.Tag{}
	var out tt.InsertTagResult
	// Execute the activity by name to avoid tight coupling to the activity symbol.
	// Ensure your worker registers an activity with this name.
	if err := workflow.ExecuteActivity(ctx, tagsActivity.InsertTag, params).Get(ctx, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
