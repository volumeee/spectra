#!/bin/bash
set -e

BUILD_DIR="bin/plugins"
mkdir -p "$BUILD_DIR"

for dir in plugins/*/; do
    name=$(basename "$dir")
    echo "Building plugin: $name"
    go build -o "$BUILD_DIR/$name" "./$dir"

    # Copy manifest if exists
    if [ -f "$dir/plugin.json" ]; then
        cp "$dir/plugin.json" "$BUILD_DIR/${name}.json"
    fi
done

echo "All plugins built in $BUILD_DIR"
