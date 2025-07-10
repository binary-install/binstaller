package schema

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed main.tsp
var typeSpecSource []byte

//go:embed output/@typespec/json-schema/InstallSpec.json
var installSpecSchema []byte

// GetTypeSpecSource returns the embedded TypeSpec source file
func GetTypeSpecSource() []byte {
	return typeSpecSource
}

// GetInstallSpecSchema returns the embedded InstallSpec JSON schema
func GetInstallSpecSchema() (interface{}, error) {
	var jsonSchema interface{}
	if err := json.Unmarshal(installSpecSchema, &jsonSchema); err != nil {
		return nil, fmt.Errorf("failed to parse JSON schema: %w", err)
	}
	return jsonSchema, nil
}

// GetInstallSpecSchemaRaw returns the raw InstallSpec JSON schema bytes
func GetInstallSpecSchemaRaw() []byte {
	return installSpecSchema
}
