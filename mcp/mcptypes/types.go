package mcptypes

import "github.com/google/jsonschema-go/jsonschema"

type Error struct {
	Error string `json:"error,omitempty" jsonschema:"the error, if any"`
}

func (e Error) JSONSchema() *jsonschema.Schema {
	schema, err := jsonschema.For[Error](nil)
	if err != nil {
		panic(err)
	}
	return schema
}
