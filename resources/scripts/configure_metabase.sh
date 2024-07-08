#!/usr/bin/env bash

metabase_host="localhost"
metabase_port="8083"

# Wait for Metabase to be ready
echo "Waiting for Metabase to start..."
until $(curl --output /dev/null --silent --head --fail http://$metabase_host:$metabase_port/api/health); do
    printf '.'
    sleep 5
done
echo "\nMetabase is ready."

echo "Fetching setup token..."
setup_token_response=$(curl -s "http://$metabase_host:$metabase_port/api/session/properties")
setup_token=$(echo "$setup_token_response" | jq -r '.["setup-token"]')

if [ -z "$setup_token" ] || [ "$setup_token" == "null" ]; then
    echo "Failed to fetch setup token."
    exit 1
fi

echo "Setup token fetched: $setup_token"

echo "Setting up Metabase..."
setup_response=$(curl -s -X POST "http://$metabase_host:$metabase_port/api/setup/" \
    -H "Content-Type: application/json" \
    -d "{
        \"token\": \"$setup_token\",
        \"user\": {
            \"email\": \"nada@nav.no\",
            \"password\": \"superdupersecret1\",
            \"first_name\": \"Nada\",
            \"last_name\": \"Backend\"
        },
        \"prefs\": {
            \"allow_tracking\": false,
            \"site_name\": \"Nada Backend\"
        }
    }")

echo "Metabase setup response: $setup_response"
