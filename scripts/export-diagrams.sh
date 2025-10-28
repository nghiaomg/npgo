#!/bin/bash

# npgo Diagram Export Script
# This script exports Mermaid diagrams to PNG images

set -e

# Check if mermaid-cli is installed
if ! command -v mmdc &> /dev/null; then
    echo "mermaid-cli is not installed. Installing..."
    npm install -g @mermaid-js/mermaid-cli
fi

# Create exported directory if it doesn't exist
mkdir -p diagrams/exported

# Export diagrams
echo "Exporting diagrams..."

# Export fetch flow diagram
if [ -f "diagrams/fetch_flow.md" ]; then
    echo "Exporting fetch_flow.mmd..."
    # Extract mermaid code from markdown and export
    grep -A 50 "```mermaid" diagrams/fetch_flow.md | grep -v "```mermaid" | grep -B 50 "```" | grep -v "```" > diagrams/src/fetch_flow.mmd
    mmdc -i diagrams/src/fetch_flow.mmd -o diagrams/exported/fetch_flow.png
fi

# Export install sequence diagram
if [ -f "diagrams/install_sequence.md" ]; then
    echo "Exporting install_sequence.mmd..."
    # Extract mermaid code from markdown and export
    grep -A 50 "```mermaid" diagrams/install_sequence.md | grep -v "```mermaid" | grep -B 50 "```" | grep -v "```" > diagrams/src/install_sequence.mmd
    mmdc -i diagrams/src/install_sequence.mmd -o diagrams/exported/install_sequence.png
fi

echo "Diagrams exported successfully to diagrams/exported/"
