#!/bin/sh
set -e

echo "Templating ACL file..."
sed -e "s/\${VALKEY_DEFAULT_PASSWORD}/${VALKEY_DEFAULT_PASSWORD}/g" \
    -e "s/\${CONTROLPLANE_VALKEY_PASSWORD}/${CONTROLPLANE_VALKEY_PASSWORD}/g" \
    -e "s/\${WORKER_VALKEY_PASSWORD}/${WORKER_VALKEY_PASSWORD}/g" \
    /etc/valkey/users.acl.template > /etc/valkey/users.acl
echo "Starting valkey-server..."
# Start server in background
valkey-server /etc/valkey/valkey.conf &
VALKEY_PID=$!

# Wait for valkey to be ready
until valkey-cli -a "${VALKEY_DEFAULT_PASSWORD}" ping | grep -q PONG; do
  echo "Waiting for valkey to start..."
  sleep 1
done

echo "Creating consumer group if it doesn't exist..."
valkey-cli -a "${VALKEY_DEFAULT_PASSWORD}" XGROUP CREATE pipeline:jobs:transcode transcode-workers $ MKSTREAM || true

echo "Initialization complete. Bringing valkey to foreground."
wait $VALKEY_PID
