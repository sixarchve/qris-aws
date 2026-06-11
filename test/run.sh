#!/bin/bash
# Run K6 load test against the backend
# Usage: ./k6/run.sh <test>
#   ./k6/run.sh qris
#   ./k6/run.sh async

set -e

TEST=$1
BASE_URL="http://host.docker.internal:8080"

case "$TEST" in
  qris)
    SCRIPT="qris_generation.js"
    ;;
  async)
    SCRIPT="scan_async_payment.js"
    ;;
  *)
    echo "Usage: ./k6/run.sh <qris|async>"
    exit 1
    ;;
esac

echo "Running K6 load test: $SCRIPT (Normal Network)"
echo "   Target: $BASE_URL"
echo ""

docker run --rm \
  --add-host=host.docker.internal:host-gateway \
  -v "$(pwd)/k6:/scripts" \
  -e BASE_URL="$BASE_URL" \
  grafana/k6 run "/scripts/$SCRIPT"
