package gateway

import (
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/require"
)

// TestPlainIntegers checks that the int64 format is removed wherever an
// integer appears, nested or shared, and that other formats are kept.
func TestPlainIntegers(t *testing.T) {
	int64Schema := func() *huma.Schema { return &huma.Schema{Type: huma.TypeInteger, Format: "int64"} }
	shared := int64Schema()
	registry := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	registry.Map()["Thing"] = &huma.Schema{
		Type: huma.TypeObject,
		Properties: map[string]*huma.Schema{
			"id":        int64Schema(),
			"small":     {Type: huma.TypeInteger, Format: "int32"},
			"at":        {Type: huma.TypeString, Format: "date-time"},
			"ids":       {Type: huma.TypeArray, Items: int64Schema()},
			"either":    {OneOf: []*huma.Schema{shared, {Type: huma.TypeString}}},
			"sharedToo": shared,
		},
	}
	pathParam := int64Schema()
	body := int64Schema()
	response := int64Schema()
	doc := &huma.OpenAPI{
		Components: &huma.Components{Schemas: registry},
		Paths: map[string]*huma.PathItem{"/things/{id}": {Patch: &huma.Operation{
			Parameters:  []*huma.Param{{Name: "id", In: "path", Schema: pathParam}},
			RequestBody: &huma.RequestBody{Content: map[string]*huma.MediaType{"application/json": {Schema: body}}},
			Responses: map[string]*huma.Response{"200": {
				Content: map[string]*huma.MediaType{"application/json": {Schema: response}},
			}},
		}}},
	}

	PlainIntegers(doc)

	thing := registry.Map()["Thing"].Properties
	tests := map[string]struct {
		schema     *huma.Schema
		wantFormat string
	}{
		"a property":        {schema: thing["id"], wantFormat: ""},
		"an array's items":  {schema: thing["ids"].Items, wantFormat: ""},
		"a shared schema":   {schema: shared, wantFormat: ""},
		"a path parameter":  {schema: pathParam, wantFormat: ""},
		"a request body":    {schema: body, wantFormat: ""},
		"a response body":   {schema: response, wantFormat: ""},
		"a 32-bit integer":  {schema: thing["small"], wantFormat: "int32"},
		"a string's format": {schema: thing["at"], wantFormat: "date-time"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.wantFormat, tc.schema.Format)
		})
	}
}
