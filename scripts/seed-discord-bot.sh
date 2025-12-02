#!/bin/bash
# Script to seed a Discord bot via the API
# Usage: ./scripts/seed-discord-bot.sh

set -e

API_URL="${API_URL:-http://localhost:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-adminpassword}"

echo "=== Discord Bot Seeder ==="
echo ""

# Login and get token
echo "1. Logging in as admin..."
LOGIN_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"${ADMIN_EMAIL}\", \"password\": \"${ADMIN_PASSWORD}\"}")

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "Failed to login. Response: $LOGIN_RESPONSE"
    exit 1
fi
echo "   Got token: ${TOKEN:0:20}..."
echo ""

# Create Discord Bot
echo "2. Creating Discord bot..."

# Replace these with your actual Discord bot credentials
BOT_NAME="${BOT_NAME:-Multi-Worker Bot}"
BOT_APP_ID="${BOT_APP_ID:-YOUR_APPLICATION_ID}"
BOT_PUBLIC_KEY="${BOT_PUBLIC_KEY:-YOUR_PUBLIC_KEY}"
BOT_TOKEN="${BOT_TOKEN:-YOUR_BOT_TOKEN}"
BOT_CLIENT_ID="${BOT_CLIENT_ID:-YOUR_CLIENT_ID}"
BOT_CLIENT_SECRET="${BOT_CLIENT_SECRET:-YOUR_CLIENT_SECRET}"

BOT_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/discord/bots" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${TOKEN}" \
    -d "{
        \"name\": \"${BOT_NAME}\",
        \"application_id\": \"${BOT_APP_ID}\",
        \"public_key\": \"${BOT_PUBLIC_KEY}\",
        \"token\": \"${BOT_TOKEN}\",
        \"client_id\": \"${BOT_CLIENT_ID}\",
        \"client_secret\": \"${BOT_CLIENT_SECRET}\",
        \"is_default\": true
    }")

echo "   Response:"
echo "$BOT_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$BOT_RESPONSE"
echo ""

BOT_ID=$(echo "$BOT_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$BOT_ID" ]; then
    echo "Bot may already exist or creation failed."
else
    echo "   Bot created with ID: $BOT_ID"
fi

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Next steps:"
echo "1. Add the bot to your Discord server:"
echo "   https://discord.com/api/oauth2/authorize?client_id=YOUR_CLIENT_ID&permissions=2048&scope=bot"
echo ""
echo "2. Create a channel configuration:"
echo "   curl -X POST ${API_URL}/api/v1/discord/channels \\"
echo "     -H 'Authorization: Bearer YOUR_TOKEN' \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"bot_id\": \"BOT_ID\", \"channel_id\": \"YOUR_CHANNEL_ID\", \"name\": \"alerts\", \"webhook_url\": \"YOUR_WEBHOOK\"}'"
echo ""
echo "3. Or set a webhook directly on a task:"
echo "   curl -X PUT ${API_URL}/api/v1/tasks/TASK_ID/discord \\"
echo "     -H 'Authorization: Bearer YOUR_TOKEN' \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"webhook_url\": \"YOUR_DISCORD_WEBHOOK_URL\"}'"
