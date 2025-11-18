#!/bin/bash

# Install pre-commit hooks
echo "Installing pre-commit hooks..."

# Check if pre-commit is installed
if ! command -v pre-commit &> /dev/null; then
    echo "pre-commit is not installed. Installing via pip..."
    pip install pre-commit || {
        echo "Failed to install pre-commit. Please install it manually:"
        echo "  pip install pre-commit"
        echo "or"
        echo "  brew install pre-commit"
        exit 1
    }
fi

# Install the hooks
pre-commit install

echo "Pre-commit hooks installed successfully!"
echo "To run hooks manually: pre-commit run --all-files"
