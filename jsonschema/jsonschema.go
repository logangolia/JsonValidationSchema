package jsonschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type SchemaValidator struct {
	schemaFilename string
	schema         *jsonschema.Schema
}

func NewSchemaValidator(jsonSchemaFilename string) (SchemaValidator, error) {
	if jsonSchemaFilename == "" {
		slog.Info("NewSchemaValidator: no json schema file specified")
		return SchemaValidator{
			schemaFilename: "",
			schema:         nil,
		}, nil
	}

	// create a JSON Schema compiler
	compiler := jsonschema.NewCompiler()

	// compile JSON schema
	schema, err := compiler.Compile(jsonSchemaFilename)
	if err != nil {
		slog.Error("NewSchemaValidator: schema compilation error", "error", err)
		return SchemaValidator{
			schemaFilename: jsonSchemaFilename,
			schema:         nil,
		}, err
	}

	// create new SchemaValidator
	sv := SchemaValidator{
		schemaFilename: jsonSchemaFilename,
		schema:         schema,
	}
	slog.Info("NewSchemaValidator: successfully created from", "filename", jsonSchemaFilename)
	return sv, nil
}

// Validates raw byte data based on json schema.
// If no schema is provided, returns as success (no err).
func (sv SchemaValidator) ValidateData(data []byte) error {

	if sv.schemaFilename == "" {
		slog.Info("validateSchema: no json schema file specified, skipping")
		return nil
	}
	// return error if filename exists but not schema
	if sv.schema == nil {
		slog.Error("validateSchema: invalid json schema specified")
		return errors.New("schema file does not have valid schema")
	}

	// unmarshal JSON data
	var d interface{}
	if err := json.Unmarshal(data, &d); err != nil {
		slog.Error("validateSchema: unable to unmarshal data", "data", data, "error", err)
		return err
	}

	// validate the JSON data against the compiled schema
	if err := sv.schema.Validate(d); err != nil {
		msg := fmt.Sprintf("%#v", err)
		slog.Error("validateSchema: data does not conform to the schema", "error", msg)
		return err
	}

	slog.Info("validateSchema: JSON data conforms to the schema")
	return nil
}
