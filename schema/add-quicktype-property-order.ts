#!/usr/bin/env -S deno run --allow-read --allow-write --allow-env

import {
  compile,
  Model,
  navigateProgram,
  NodeHost,
} from "npm:@typespec/compiler";
import { parse } from "https://deno.land/std@0.224.0/flags/mod.ts";

interface JsonSchemaObject {
  type?: string;
  properties?: Record<string, unknown>;
  quicktypePropertyOrder?: string[];
  $defs?: Record<string, JsonSchemaObject>;
  [key: string]: unknown;
}

interface TypeSpecModel {
  name: string;
  properties: string[];
}

/**
 * Parse TypeSpec file using @typespec/compiler to extract model definitions and their property order
 */
async function parseTypeSpecFile(filePath: string): Promise<TypeSpecModel[]> {
  const models: TypeSpecModel[] = [];

  // Compile the TypeSpec file
  const program = await compile(NodeHost, filePath, {
    noEmit: true, // Just parse, don't emit output
  });

  // Check for compilation errors
  if (program.diagnostics.length > 0) {
    console.error("TypeSpec compilation errors:");
    for (const diagnostic of program.diagnostics) {
      console.error(`  ${diagnostic.message}`);
    }
    throw new Error("Failed to compile TypeSpec file");
  }

  // Navigate through all types in the program
  navigateProgram(program, {
    model(model: Model) {
      // Extract property names in order
      const properties: string[] = [];
      for (const [propName] of model.properties) {
        properties.push(propName);
      }

      models.push({
        name: model.name,
        properties,
      });
    },
  });

  return models;
}

/**
 * Find the TypeSpec model that matches the root schema properties
 */
function findRootModel(
  schema: JsonSchemaObject,
  models: TypeSpecModel[],
): string | null {
  if (!schema.properties) return null;

  const rootPropNames = Object.keys(schema.properties);

  // Find the model that has the same properties as the root schema
  for (const model of models) {
    if (
      model.properties.length === rootPropNames.length &&
      model.properties.every((prop) => rootPropNames.includes(prop))
    ) {
      return model.name;
    }
  }

  return null;
}

/**
 * Add quicktypePropertyOrder to a JSON Schema object
 */
function addPropertyOrder(
  schema: JsonSchemaObject,
  propertyOrder: string[],
): JsonSchemaObject {
  if (!schema.properties) {
    return schema;
  }

  // Filter property order to only include properties that exist in the schema
  const existingProps = Object.keys(schema.properties);
  const filteredOrder = propertyOrder.filter((prop) =>
    existingProps.includes(prop)
  );

  // Add any properties not in the order at the end
  for (const prop of existingProps) {
    if (!filteredOrder.includes(prop)) {
      filteredOrder.push(prop);
    }
  }

  // Create new object with quicktypePropertyOrder inserted after properties
  const result: JsonSchemaObject = {};
  for (const [key, value] of Object.entries(schema)) {
    result[key] = value;
    if (key === "properties") {
      result.quicktypePropertyOrder = filteredOrder;
    }
  }

  return result;
}

/**
 * Process JSON Schema file and add quicktypePropertyOrder based on TypeSpec models
 */
async function processJsonSchema(
  jsonSchemaPath: string,
  typespecModels: TypeSpecModel[],
  outputPath: string,
  rootModelName?: string,
): Promise<void> {
  // Read JSON Schema
  const schemaContent = await Deno.readTextFile(jsonSchemaPath);
  const schema: JsonSchemaObject = JSON.parse(schemaContent);

  // Auto-detect or use provided root model name
  const detectedRootModel = rootModelName ||
    findRootModel(schema, typespecModels);

  if (!detectedRootModel && !rootModelName) {
    console.warn(
      "⚠️  Could not auto-detect root model. Properties may not be ordered correctly for root schema.",
    );
  } else if (!rootModelName) {
    console.log(`🔍 Auto-detected root model: ${detectedRootModel}`);
  }

  // Create a map of schema names to property orders
  const propertyOrderMap: Record<string, string[]> = {};
  for (const model of typespecModels) {
    if (model.name === detectedRootModel) {
      // Map root model to '.'
      propertyOrderMap["."] = model.properties;
    } else {
      // Use model name as-is for definitions
      propertyOrderMap[model.name] = model.properties;
    }
  }

  // Process root schema
  if (propertyOrderMap["."]) {
    Object.assign(schema, addPropertyOrder(schema, propertyOrderMap["."]));
  }

  // Process definitions
  if (schema.$defs) {
    for (const [defName, defSchema] of Object.entries(schema.$defs)) {
      if (propertyOrderMap[defName]) {
        schema.$defs[defName] = addPropertyOrder(
          defSchema,
          propertyOrderMap[defName],
        );
      }
    }
  }

  // Write to specified output path
  await Deno.writeTextFile(
    outputPath,
    JSON.stringify(schema, null, 4) + "\n",
  );
  console.log(`✅ Added quicktypePropertyOrder to ${outputPath}`);
}

// Main execution
async function main() {
  const args = parse(Deno.args, {
    string: ["typespec", "schema", "output", "root"],
    alias: {
      t: "typespec",
      s: "schema",
      o: "output",
      r: "root",
    },
    default: {
      typespec: "main.tsp",
      schema: "output/@typespec/json-schema/InstallSpec.json",
    },
  });

  const scriptDir = new URL(".", import.meta.url).pathname;
  const typespecPath = args.typespec.startsWith("/")
    ? args.typespec
    : `${scriptDir}${args.typespec}`;
  const jsonSchemaPath = args.schema.startsWith("/")
    ? args.schema
    : `${scriptDir}${args.schema}`;

  try {
    await Deno.stat(typespecPath);
  } catch {
    console.error(`❌ TypeSpec file not found: ${typespecPath}`);
    Deno.exit(1);
  }

  try {
    await Deno.stat(jsonSchemaPath);
  } catch {
    console.error(`❌ JSON Schema file not found: ${jsonSchemaPath}`);
    console.error("Please run TypeSpec compilation first.");
    Deno.exit(1);
  }

  // Output path is required
  if (!args.output) {
    console.error("❌ Output path is required. Use --output or -o option.");
    Deno.exit(1);
  }

  const outputPath = args.output as string;

  console.log("📖 Parsing TypeSpec file using @typespec/compiler...");
  const models = await parseTypeSpecFile(typespecPath);
  console.log(`Found ${models.length} models`);

  console.log("🔧 Processing JSON Schema...");
  await processJsonSchema(
    jsonSchemaPath,
    models,
    outputPath,
    args.root as string | undefined,
  );

  console.log("✨ Done!");

  // Show usage if needed
  if (args.help) {
    console.log(`
Usage: deno run --allow-read --allow-write add-quicktype-property-order.ts [options]

Options:
  -t, --typespec <path>   Path to TypeSpec file (default: main.tsp)
  -s, --schema <path>     Path to JSON Schema file (default: output/@typespec/json-schema/InstallSpec.json)
  -o, --output <path>     Output path for modified schema (required)
  -r, --root <name>       Root model name (auto-detected if not provided)
  --help                  Show this help message
`);
  }
}

// Run if called directly
if (import.meta.main) {
  await main();
}
