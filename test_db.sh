#!/bin/bash

# Test database connection by running the Go application with appropriate environment variables

echo "Testing database connection..."
DB_HOST=localhost DB_PORT=5433 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=transport_db ./main

echo "Exit code: $?" 