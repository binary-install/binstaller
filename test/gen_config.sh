#!/bin/bash
set -e

CONFIG_FILE="test/gen_config_list.yml"

echo "Generating test configurations..."

# Create temporary directory for task scripts
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Extract tasks and create script files
echo "Preparing tasks..."
task_count=$(yq eval '.tasks | length' "$CONFIG_FILE")

for i in $(seq 0 $((task_count - 1))); do
    name=$(yq eval ".tasks[$i].name" "$CONFIG_FILE")
    script_file="$TEMP_DIR/task_$(printf "%03d" $i)_${name}.sh"
    
    # Write the run command to a script file
    {
        echo "#!/bin/bash"
        echo "set -e"
        echo "echo \"Running task: $name\""
        yq eval ".tasks[$i].run" "$CONFIG_FILE"
    } > "$script_file"
    
    chmod +x "$script_file"
done

# Run all tasks in parallel using rush
echo "Running $task_count tasks in parallel..."
find "$TEMP_DIR" -name "*.sh" -type f | sort | rush -j5 -k

echo "Test configurations generated successfully!"