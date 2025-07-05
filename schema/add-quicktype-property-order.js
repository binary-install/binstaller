#!/usr/bin/env node
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// TypeScript interfaces as comments for clarity
// interface JsonSchemaObject {
//   type?: string;
//   properties?: Record<string, any>;
//   quicktypePropertyOrder?: string[];
//   $defs?: Record<string, JsonSchemaObject>;
//   [key: string]: any;
// }
// interface TypeSpecModel {
//   name: string;
//   properties: string[];
// }

/**
 * Parse TypeSpec file to extract model definitions and their property order
 */
function parseTypeSpecFile(filePath) {
  const content = fs.readFileSync(filePath, 'utf-8');
  const models = [];
  
  // Regular expression to match model definitions
  const modelRegex = /^model\s+(\w+)\s*\{/gm;
  const propertyRegex = /^\s*(\w+)\s*[?:]?\s*:/gm;
  
  let match;
  while ((match = modelRegex.exec(content)) !== null) {
    const modelName = match[1];
    const modelStart = match.index + match[0].length;
    
    // Find the closing brace for this model
    let braceCount = 1;
    let pos = modelStart;
    while (braceCount > 0 && pos < content.length) {
      if (content[pos] === '{') braceCount++;
      else if (content[pos] === '}') braceCount--;
      pos++;
    }
    
    // Extract model body
    const modelBody = content.substring(modelStart, pos - 1);
    
    // Extract properties in order
    const properties = [];
    propertyRegex.lastIndex = 0;
    let propMatch;
    while ((propMatch = propertyRegex.exec(modelBody)) !== null) {
      properties.push(propMatch[1]);
    }
    
    models.push({ name: modelName, properties });
  }
  
  return models;
}

/**
 * Map TypeSpec model names to JSON Schema definition names
 */
function mapModelNameToSchemaName(modelName) {
  const mapping = {
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
function addPropertyOrder(schema, propertyOrder) {
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
  const result = {};
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
function processJsonSchema(jsonSchemaPath, typespecModels) {
  // Read JSON Schema
  const schemaContent = fs.readFileSync(jsonSchemaPath, 'utf-8');
  const schema = JSON.parse(schemaContent);
  
  // Create a map of schema names to property orders
  const propertyOrderMap = {};
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
  fs.writeFileSync(jsonSchemaPath, JSON.stringify(schema, null, 4) + '\n');
  console.log(`✅ Added quicktypePropertyOrder to ${jsonSchemaPath}`);
}

// Main execution
function main() {
  const typespecPath = path.join(__dirname, 'main.tsp');
  const jsonSchemaPath = path.join(__dirname, 'output/@typespec/json-schema/InstallSpec.json');
  
  if (!fs.existsSync(typespecPath)) {
    console.error(`❌ TypeSpec file not found: ${typespecPath}`);
    process.exit(1);
  }
  
  if (!fs.existsSync(jsonSchemaPath)) {
    console.error(`❌ JSON Schema file not found: ${jsonSchemaPath}`);
    console.error('Please run TypeSpec compilation first.');
    process.exit(1);
  }
  
  console.log('📖 Parsing TypeSpec file...');
  const models = parseTypeSpecFile(typespecPath);
  console.log(`Found ${models.length} models`);
  
  console.log('🔧 Processing JSON Schema...');
  processJsonSchema(jsonSchemaPath, models);
  
  console.log('✨ Done!');
}

// Run if called directly
main();