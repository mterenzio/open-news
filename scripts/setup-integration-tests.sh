#!/bin/bash

# Integration test setup script
set -e

echo "Setting up integration test environment..."

# Check if PostgreSQL is running
if ! pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
    echo "Error: PostgreSQL is not running on localhost:5432"
    echo "Please start PostgreSQL before running integration tests"
    exit 1
fi

# Create test database if it doesn't exist
echo "Creating test database..."
createdb open_news_test 2>/dev/null || echo "Test database already exists"

# Set environment for tests
export ENV_FILE=.env.test.integration

echo "Running database migrations on test database..."
go run cmd/migrate.go

echo "Integration test environment ready!"
echo "To run integration tests: make test-integration"
