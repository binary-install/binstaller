package schema

import (
	_ "embed"
)

//go:embed main.tsp
var typeSpecSource []byte

//go:embed output/@typespec/json-schema/InstallSpec.json
var installSpecSchemaJSON []byte

//go:embed binstaller-schema.yaml
var installSpecSchemaYAML []byte

// GetTypeSpecSource returns the embedded TypeSpec source file
func GetTypeSpecSource() []byte {
	return typeSpecSource
}

// GetInstallSpecSchemaJSON returns the raw InstallSpec JSON schema bytes
func GetInstallSpecSchemaJSON() []byte {
	return installSpecSchemaJSON
}

// GetInstallSpecSchemaYAML returns the raw InstallSpec YAML schema bytes
func GetInstallSpecSchemaYAML() []byte {
	return installSpecSchemaYAML
}
