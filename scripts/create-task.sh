#!/bin/bash
# Script to create a task via the API
# Usage: ./scripts/create-task.sh <task-file.json> [token]

set -e

API_URL="${API_URL:-http://localhost:8080}"
TASK_FILE="${1:-examples/task-job-scraper.json}"
TOKEN="${2:-}"

if [ -z "$TOKEN" ]; then
    echo "No token provided. Getting token from login..."

    # Try to login with default admin credentials
    ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"
    ADMIN_PASSWORD="${ADMIN_PASSWORD:-adminpassword}"

    LOGIN_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"${ADMIN_EMAIL}\", \"password\": \"${ADMIN_PASSWORD}\"}")

    TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

    if [ -z "$TOKEN" ]; then
        echo "Failed to get token. Response: $LOGIN_RESPONSE"
        exit 1
    fi
    echo "Got token: ${TOKEN:0:20}..."
fi

echo "Creating task from: $TASK_FILE"

RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/tasks" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${TOKEN}" \
    -d @"$TASK_FILE")

echo "Response:"
echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
