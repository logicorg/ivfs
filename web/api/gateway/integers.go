package gateway

import "github.com/danielgtaylor/huma/v2"

// PlainIntegers removes the int64 format Huma gives every Go int and int64
// in the document. The generated zod schemas would otherwise parse those
// integers as BigInt, which the TypeScript types do not expect and JSON
// cannot encode. A JavaScript number holds them exactly up to 2^53, which is
// enough for ids and precisions (ADR 0007).
func PlainIntegers(doc *huma.OpenAPI) {
	seen := map[*huma.Schema]bool{}
	var walk func(schema *huma.Schema)
	walk = func(schema *huma.Schema) {
		if schema == nil || seen[schema] {
			return
		}
		seen[schema] = true
		if schema.Type == huma.TypeInteger && schema.Format == "int64" {
			schema.Format = ""
		}
		walk(schema.Items)
		walk(schema.Not)
		if additional, ok := schema.AdditionalProperties.(*huma.Schema); ok {
			walk(additional)
		}
		for _, property := range schema.Properties {
			walk(property)
		}
		for _, list := range [][]*huma.Schema{schema.OneOf, schema.AnyOf, schema.AllOf} {
			for _, item := range list {
				walk(item)
			}
		}
	}

	if doc.Components != nil && doc.Components.Schemas != nil {
		for _, schema := range doc.Components.Schemas.Map() {
			walk(schema)
		}
	}
	for _, path := range doc.Paths {
		for _, operation := range []*huma.Operation{
			path.Get, path.Put, path.Post, path.Delete, path.Options, path.Head, path.Patch, path.Trace,
		} {
			if operation == nil {
				continue
			}
			for _, param := range operation.Parameters {
				walk(param.Schema)
			}
			if operation.RequestBody != nil {
				for _, media := range operation.RequestBody.Content {
					walk(media.Schema)
				}
			}
			for _, response := range operation.Responses {
				for _, media := range response.Content {
					walk(media.Schema)
				}
			}
		}
	}
}
