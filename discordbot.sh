#!/bin/bash

# Discord Radio - Minimal script to stream audio to Discord voice channel
# Requirements: ffmpeg, jq, curl, and a Discord bot token

# Load environment variables
if [ -f .env ]; then
export $(grep -v '^#' .env | xargs)
fi

# Check required variables
if [ -z "$DISCORD_TOKEN" ]; then
echo "Error: DISCORD_TOKEN not set"
exit 1
fi

if [ -z "$GUILD_ID" ]; then
echo "Error: GUILD_ID not set"
exit 1
fi

if [ -z "$VC_ID" ]; then
echo "Error: VC_ID not set"
exit 1
fi

# Set stream URL (default to LISTEN.moe if not specified)
STREAM_URL="${STREAM_URL:-https://listen.moe/stream}"
echo "Using stream URL: $STREAM_URL"

# Function to join voice channel and get voice session info
join_voice_channel() {
echo "Joining voice channel..."

# Get voice server info
voice_state=$(curl -s -X POST \
    -H "Authorization: Bot $DISCORD_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"channel_id\":\"$VC_ID\"}" \
    "https://discord.com/api/v10/guilds/$GUILD_ID/voice/states/@me")

session_id=$(echo "$voice_state" | jq -r '.session_id')
token=$(echo "$voice_state" | jq -r '.token')
endpoint=$(echo "$voice_state" | jq -r '.endpoint' | sed 's/:.*$//')

echo "Connected to voice channel at $endpoint"
echo "Session ID: $session_id"

# Return the voice server info
echo "$endpoint $token $session_id"
}

# Main loop - if connection fails, retry
while true; do
# Join voice channel and get voice server info
voice_info=$(join_voice_channel)
endpoint=$(echo "$voice_info" | cut -d' ' -f1)
token=$(echo "$voice_info" | cut -d' ' -f2)
session_id=$(echo "$voice_info" | cut -d' ' -f3)

if [ -z "$endpoint" ] || [ "$endpoint" = "null" ]; then
    echo "Failed to get voice server info, retrying in 5 seconds..."
    sleep 5
    continue
fi

# Stream audio using FFmpeg
echo "Starting audio stream..."
ffmpeg -reconnect 1 -reconnect_streamed 1 -reconnect_delay_max 5 \
    -i "$STREAM_URL" \
    -acodec libopus -f opus -ar 48000 -ac 2 \
    "udp://$endpoint:4000?pkt_size=1452&ssrc=1&token=$token&session_id=$session_id" \
    -loglevel warning

echo "Stream ended, reconnecting in 5 seconds..."
sleep 5
done