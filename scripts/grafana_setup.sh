#!/bin/bash

echo "COMPLETE GRAFANA SETUP"
echo "=============================="
echo ""

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
PURPLE='\033[0;35m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_demo() { echo -e "${PURPLE}[DEMO]${NC} $1"; }

log_info "Checking if services are running..."

if ! curl -s http://localhost:8080/health > /dev/null; then
    log_error "KubeSight is not running!"
    echo "Please start services first:"
    echo "  make docker-up"
    echo "  # or"
    echo "  docker-compose up -d"
    exit 1
fi

if ! curl -s http://localhost:3000/api/health > /dev/null; then
    log_error "Grafana is not running!"
    echo "Please start services first: docker-compose up -d"
    exit 1
fi

log_success "All services are running!"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

log_info "Creating Grafana dashboard..."
chmod +x "$SCRIPT_DIR/grafana.sh"
"$SCRIPT_DIR/grafana.sh"

log_info "Generating activity for demo..."
chmod +x "$SCRIPT_DIR/generate_query.sh"
"$SCRIPT_DIR/generate_query.sh"

log_info "Verifying dashboard setup..."
sleep 10

stats=$(curl -s http://localhost:8080/api/v1/stats)
total_metrics=$(echo "$stats" | jq -r '.total_metrics // 0')

if [ "$total_metrics" -gt 10000 ]; then
    log_success "$total_metrics metrics processed"
else
    log_warning "Only $total_metrics metrics found, generating more..."
    curl -X POST http://localhost:8080/api/v1/demo/generate \
      -d '{"count": 25000}' > /dev/null
fi

log_info "Final verification..."

if curl -s http://localhost:9090/api/v1/query?query=kubesight_queries_total | grep -q "data"; then
    log_success "Prometheus has KubeSight metrics"
else
    log_warning "Prometheus metrics may need a moment to appear"
fi

echo ""
echo "GRAFANA SETUP COMPLETE!"
echo "==============================="
echo ""
echo "Ready for Demo:"
echo "   Dashboard: http://localhost:3000 (admin/admin)"
echo ""
