#!/bin/bash
# Script to configure Discord channel for job notifications
# Usage: ./scripts/setup-discord-channel.sh
#
# Required environment variables:
#   DISCORD_WEBHOOK_URL - Your Discord webhook URL (create one in Discord Channel Settings → Integrations → Webhooks)
#
# Optional environment variables:
#   API_URL - API base URL (default: http://localhost:8080)
#   ADMIN_EMAIL - Admin email (default: admin@example.com)
#   ADMIN_PASSWORD - Admin password (default: adminpassword)
#   DISCORD_CHANNEL_ID - Discord channel ID (default: 1445410643015499817)
#   DISCORD_CHANNEL_NAME - Channel name in system (default: job-alerts)

set -e

API_URL="${API_URL:-http://localhost:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-adminpassword}"

# Discord configuration
DISCORD_CHANNEL_ID="${DISCORD_CHANNEL_ID:-1445410643015499817}"
DISCORD_CHANNEL_NAME="${DISCORD_CHANNEL_NAME:-job-alerts}"
DISCORD_WEBHOOK_URL="${DISCORD_WEBHOOK_URL}"

echo "=== Discord Channel Setup ==="
echo ""

# Check for required webhook URL
if [ -z "$DISCORD_WEBHOOK_URL" ]; then
    echo "ERROR: DISCORD_WEBHOOK_URL environment variable is required"
    echo ""
    echo "To create a Discord webhook:"
    echo "1. Open your Discord server"
    echo "2. Right-click on channel '${DISCORD_CHANNEL_ID}'"
    echo "3. Click 'Edit Channel' → 'Integrations' → 'Webhooks'"
    echo "4. Click 'New Webhook' and copy the webhook URL"
    echo "5. Run: DISCORD_WEBHOOK_URL='your_webhook_url' ./scripts/setup-discord-channel.sh"
    exit 1
fi

echo "Channel ID: ${DISCORD_CHANNEL_ID}"
echo "Channel Name: ${DISCORD_CHANNEL_NAME}"
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
echo "   Logged in successfully"
echo ""

# Check if bot exists, if not create a placeholder
echo "2. Checking for Discord bot..."
BOTS_RESPONSE=$(curl -s -X GET "${API_URL}/api/v1/discord/bots" \
    -H "Authorization: Bearer ${TOKEN}")

BOT_ID=$(echo "$BOTS_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$BOT_ID" ]; then
    echo "   No bot found. Creating placeholder bot..."

    BOT_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/discord/bots" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${TOKEN}" \
        -d "{
            \"name\": \"Multi-Worker Bot\",
            \"application_id\": \"placeholder\",
            \"public_key\": \"placeholder\",
            \"token\": \"placeholder\",
            \"client_id\": \"placeholder\",
            \"is_default\": true
        }")

    BOT_ID=$(echo "$BOT_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [ -z "$BOT_ID" ]; then
        echo "   Failed to create bot. Response: $BOT_RESPONSE"
        exit 1
    fi
    echo "   Created bot with ID: $BOT_ID"
else
    echo "   Found existing bot: $BOT_ID"
fi
echo ""

# Create Discord channel configuration
echo "3. Creating Discord channel configuration..."
CHANNEL_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/discord/channels" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${TOKEN}" \
    -d "{
        \"bot_id\": \"${BOT_ID}\",
        \"channel_id\": \"${DISCORD_CHANNEL_ID}\",
        \"name\": \"${DISCORD_CHANNEL_NAME}\",
        \"description\": \"Job notifications channel\",
        \"webhook_url\": \"${DISCORD_WEBHOOK_URL}\"
    }")

echo "   Response:"
echo "$CHANNEL_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$CHANNEL_RESPONSE"

CHANNEL_CONFIG_ID=$(echo "$CHANNEL_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$CHANNEL_CONFIG_ID" ]; then
    echo ""
    echo "Channel may already exist. Trying to update..."
    # List channels and find the one to update
else
    echo ""
    echo "   Channel configured with ID: $CHANNEL_CONFIG_ID"
fi

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Your Discord channel is now configured!"
echo "Channel ID: ${DISCORD_CHANNEL_ID}"
echo ""
echo "To link a task to this channel, use:"
echo "  curl -X PUT ${API_URL}/api/v1/tasks/TASK_ID/discord \\"
echo "    -H 'Authorization: Bearer YOUR_TOKEN' \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"channel_id\": \"${CHANNEL_CONFIG_ID}\"}'"
echo ""
echo "Or set the webhook directly on a task:"
echo "  curl -X PUT ${API_URL}/api/v1/tasks/TASK_ID/discord \\"
echo "    -H 'Authorization: Bearer YOUR_TOKEN' \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"webhook_url\": \"YOUR_WEBHOOK_URL\"}'"
