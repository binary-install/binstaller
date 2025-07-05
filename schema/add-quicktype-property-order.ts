#!/usr/bin/env -S deno run --allow-read --allow-write

import { compile, navigateProgram, NodeHost, Model } from "npm:@typespec/compiler";

interface JsonSchemaObject {
  type?: string;
  properties?: Record<string, any>;
  quicktypePropertyOrder?: string[];
  $defs?: Record<string, JsonSchemaObject>;
  [key: string]: any;
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
      for (const [propName, _prop] of model.properties) {
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
 * Map TypeSpec model names to JSON Schema definition names
 */
function mapModelNameToSchemaName(modelName: string): string | null {
  const mapping: Record<string, string> = {
    'InstallSpec': '.',  // Root schema
    'Platform': 'Platform',
    'AssetConfig': 'AssetConfig',
    'AssetRule': 'AssetRule',
    'Binary': 'Binary',
    'PlatformCondition': 'PlatformCondition',
    'NamingConvention': 'NamingConvention',
    'ArchEmulation': 'ArchEmulation',
    'ChecksumConfig': 'ChecksumConfig',
    'EmbeddedChecksum': 'EmbeddedChecksum',
    'UnpackConfig': 'UnpackConfig'
  };
  
  return mapping[modelName] || null;
}

/**
 * Add quicktypePropertyOrder to a JSON Schema object
 */
function addPropertyOrder(schema: JsonSchemaObject, propertyOrder: string[]): JsonSchemaObject {
  if (!schema.properties) {
    return schema;
  }
  
  // Filter property order to only include properties that exist in the schema
  const existingProps = Object.keys(schema.properties);
  const filteredOrder = propertyOrder.filter(prop => existingProps.includes(prop));
  
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
    if (key === 'properties') {
      result.quicktypePropertyOrder = filteredOrder;
    }
  }
  
  return result;
}

/**
 * Process JSON Schema file and add quicktypePropertyOrder based on TypeSpec models
 */
async function processJsonSchema(jsonSchemaPath: string, typespecModels: TypeSpecModel[]): Promise<void> {
  // Read JSON Schema
  const schemaContent = await Deno.readTextFile(jsonSchemaPath);
  const schema: JsonSchemaObject = JSON.parse(schemaContent);
  
  // Create a map of schema names to property orders
  const propertyOrderMap: Record<string, string[]> = {};
  for (const model of typespecModels) {
    const schemaName = mapModelNameToSchemaName(model.name);
    if (schemaName) {
      propertyOrderMap[schemaName] = model.properties;
    }
  }
  
  // Process root schema
  if (propertyOrderMap['.']) {
    Object.assign(schema, addPropertyOrder(schema, propertyOrderMap['.']));
  }
  
  // Process definitions
  if (schema.$defs) {
    for (const [defName, defSchema] of Object.entries(schema.$defs)) {
      if (propertyOrderMap[defName]) {
        schema.$defs[defName] = addPropertyOrder(defSchema, propertyOrderMap[defName]);
      }
    }
  }
  
  // Write back the modified schema
  await Deno.writeTextFile(jsonSchemaPath, JSON.stringify(schema, null, 4) + '\n');
  console.log(`✅ Added quicktypePropertyOrder to ${jsonSchemaPath}`);
}

// Main execution
async function main() {
  const scriptDir = new URL('.', import.meta.url).pathname;
  const typespecPath = `${scriptDir}main.tsp`;
  const jsonSchemaPath = `${scriptDir}output/@typespec/json-schema/InstallSpec.json`;
  
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
    console.error('Please run TypeSpec compilation first.');
    Deno.exit(1);
  }
  
  console.log('📖 Parsing TypeSpec file using @typespec/compiler...');
  const models = await parseTypeSpecFile(typespecPath);
  console.log(`Found ${models.length} models`);
  
  console.log('🔧 Processing JSON Schema...');
  await processJsonSchema(jsonSchemaPath, models);
  
  console.log('✨ Done!');
}

// Run if called directly
if (import.meta.main) {
  await main();
}