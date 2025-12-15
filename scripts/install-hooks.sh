#!/bin/sh
# Install git hooks

if ! command -v pre-commit &> /dev/null; then
    echo "pre-commit not found. Please install it with 'pip install pre-commit' or 'brew install pre-commit'."
    # Don't exit, just warn
else
    pre-commit install
fi
