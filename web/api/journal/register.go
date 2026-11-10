// Package journal is the web API's journal entry endpoints: an account's
// statement, an entry's detail, and its deletion.
package journal

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"journal"}

// Register adds every journal entry endpoint to api.
func Register(api huma.API, c client.Client) {
	registerList(api, c)
	registerRetrieve(api, c)
	registerDelete(api, c)
}
