#!/bin/bash
set -euo pipefail

# Shadow Coordinator + Prover Setup Script
# Usage: ./setup.sh [--postgres] [--coordinator] [--prover] [--all]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/configs"

# Load .env if present
if [ -f "$SCRIPT_DIR/.env" ]; then
    export $(grep -v '^#' "$SCRIPT_DIR/.env" | xargs)
fi

# Build DSNs from components if not already set
PROD_DB_PASSWORD="${PROD_DB_PASSWORD:-}"
SHADOW_DB_PASSWORD="${SHADOW_DB_PASSWORD:-}"
PROD_DB="${PROD_DB:-postgresql://$PROD_DB_USER:$PROD_DB_PASSWORD@$PROD_DB_HOST:$PROD_DB_PORT/$PROD_DB_NAME}"
SHADOW_DB="${SHADOW_DB:-postgresql://$SHADOW_DB_USER:$SHADOW_DB_PASSWORD@$SHADOW_DB_HOST:$SHADOW_DB_PORT/$SHADOW_DB_NAME}"

VERIFIER_DIR="${VERIFIER_DIR:-/tmp/shadow-verifier-assets}"
IMAGE_TAG="${IMAGE_TAG:-v4.7.13-openvm16}"
L2_RPC="${L2_RPC:-https://mainnet-rpc.scroll.io}"

show_help() {
    cat <<EOF
Shadow Coordinator + Prover Setup

Usage: $0 [OPTIONS]

Options:
    --postgres          Start/restart shadow PostgreSQL container
    --coordinator       Start shadow coordinator API container
    --prover            Start local prover (binary or docker)
    --all               Run all steps in sequence
    --stop              Stop all shadow containers
    --status            Show status of shadow services
    -h, --help          Show this help

Environment Variables:
    PROD_DB             Production RDS connection string
    SHADOW_DB           Shadow DB connection string
    VERIFIER_DIR        Path to verifier assets
    IMAGE_TAG           Docker image tag (default: $IMAGE_TAG)
    L2_RPC              L2 RPC endpoint (default: $L2_RPC)

Examples:
    # Full setup
    $0 --all

    # Individual steps
    $0 --postgres
    $0 --coordinator
    $0 --prover

    # Stop everything
    $0 --stop
EOF
}

log_info() {
    echo "[INFO] $1"
}

log_error() {
    echo "[ERROR] $1" >&2
}

wait_for_postgres() {
    local max_attempts=30
    local attempt=1
    while [ $attempt -le $max_attempts ]; do
        if docker exec shadow-coordinator-postgres pg_isready -U postgres >/dev/null 2>&1; then
            log_info "PostgreSQL is ready"
            return 0
        fi
        log_info "Waiting for PostgreSQL... ($attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    log_error "PostgreSQL failed to start"
    return 1
}

setup_postgres() {
    log_info "Setting up shadow PostgreSQL..."

    # Stop and remove existing container
    if docker ps -a --format '{{.Names}}' | grep -q '^shadow-coordinator-postgres$'; then
        log_info "Removing existing PostgreSQL container..."
        docker rm -f shadow-coordinator-postgres >/dev/null
    fi

    docker run -d \
        --name shadow-coordinator-postgres \
        -e POSTGRES_USER=postgres \
        -e POSTGRES_PASSWORD="${SHADOW_DB_PASSWORD:?SHADOW_DB_PASSWORD must be set}" \
        -e POSTGRES_DB=shadow_rollup \
        -p 5433:5432 \
        -v shadow-coordinator-postgres-data:/var/lib/postgresql/data \
        postgres:15 >/dev/null

    wait_for_postgres

    # Apply migrations if db_cli image is available
    if docker images --format '{{.Repository}}:{{.Tag}}' | grep -q "zhuoatscroll/db_cli:$IMAGE_TAG"; then
        log_info "Running database migrations..."
        docker run --rm \
            --network host \
            -e DATABASE_URL="$SHADOW_DB?sslmode=disable" \
            zhuoatscroll/db_cli:$IMAGE_TAG \
            migrate up || log_info "Migration may have failed or already applied"
    fi

    log_info "PostgreSQL setup complete at $SHADOW_DB"
}

setup_coordinator() {
    log_info "Setting up shadow coordinator..."

    # Check prerequisites
    if [ ! -d "$VERIFIER_DIR/openvm-0.5.6" ] || [ ! -d "$VERIFIER_DIR/openvm-v0.8.0" ]; then
        log_error "Verifier assets not found at $VERIFIER_DIR"
        log_error "Please download verifier assets first."
        exit 1
    fi

    # Generate config with correct L2 RPC
    local config_file="/tmp/shadow-coordinator-config.json"
    cp "$CONFIG_DIR/shadow-coordinator-config.json" "$config_file"
    # Update L2 RPC if different from default
    if [ "$L2_RPC" != "https://mainnet-rpc.scroll.io" ]; then
        sed -i "s|https://mainnet-rpc.scroll.io|$L2_RPC|g" "$config_file"
    fi

    # Stop existing container
    if docker ps -a --format '{{.Names}}' | grep -q '^shadow-coordinator-api-test$'; then
        log_info "Removing existing coordinator container..."
        docker rm -f shadow-coordinator-api-test >/dev/null
    fi

    # Kill any stale coordinator processes on host
    pkill -f "coordinator_api" 2>/dev/null || true

    log_info "Starting coordinator container (this will take 2-3 min for OpenVM keygen)..."
    docker run -d \
        --name shadow-coordinator-api-test \
        --network host \
        -v "$config_file":/app/conf/config.json \
        -v "$VERIFIER_DIR":/verifier:ro \
        zhuoatscroll/coordinator-api:$IMAGE_TAG >/dev/null

    log_info "Waiting for coordinator to start..."
    local attempt=1
    local max_attempts=60
    while [ $attempt -le $max_attempts ]; do
        if docker logs shadow-coordinator-api-test 2>&1 | grep -q "Start coordinator api successfully"; then
            log_info "Coordinator is ready at http://localhost:8390"
            return 0
        fi
        if ! docker ps --format '{{.Names}}' | grep -q '^shadow-coordinator-api-test$'; then
            log_error "Coordinator container exited unexpectedly"
            docker logs shadow-coordinator-api-test --tail 50
            exit 1
        fi
        echo -n "."
        sleep 5
        ((attempt++))
    done
    log_error "Coordinator failed to start within timeout"
    docker logs shadow-coordinator-api-test --tail 100
    exit 1
}

setup_prover() {
    log_info "Setting up prover..."

    # Check for prover binary or use docker
    local prover_binary=""
    if [ -f "$SCRIPT_DIR/../../target/release/prover" ]; then
        prover_binary="$SCRIPT_DIR/../../target/release/prover"
    elif [ -f "$(pwd)/target/release/prover" ]; then
        prover_binary="$(pwd)/target/release/prover"
    fi

    local config_file="/tmp/prover-local.json"
    cp "$CONFIG_DIR/prover-local.json" "$config_file"

    if [ -n "$prover_binary" ]; then
        log_info "Using local prover binary: $prover_binary"
        log_info "Starting prover..."
        "$prover_binary" --config "$config_file" &
        log_info "Prover started in background (PID: $!)"
        log_info "Monitor with: tail -f /tmp/prover.log"
    else
        log_info "Prover binary not found, using Docker..."
        if docker ps -a --format '{{.Names}}' | grep -q '^shadow-prover$'; then
            docker rm -f shadow-prover >/dev/null
        fi
        docker run -d \
            --name shadow-prover \
            --network host \
            --gpus all \
            -v "$config_file":/app/config.json \
            -v "$HOME/.openvm/params":/root/.openvm/params:ro \
            zhuoatscroll/prover:$IMAGE_TAG >/dev/null
        log_info "Prover container started"
    fi

    log_info "Prover health check: curl http://localhost:10080/health"
}

stop_all() {
    log_info "Stopping all shadow services..."
    docker rm -f shadow-coordinator-api-test 2>/dev/null || true
    docker rm -f shadow-prover 2>/dev/null || true
    docker rm -f shadow-coordinator-postgres 2>/dev/null || true
    pkill -f "coordinator_api" 2>/dev/null || true
    pkill -f "prover " 2>/dev/null || true
    log_info "All shadow services stopped"
}

show_status() {
    echo "=== Shadow Services Status ==="
    echo ""
    echo "Containers:"
    docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' | grep -E 'shadow|NAMES' || echo "  No shadow containers running"
    echo ""
    echo "Port usage:"
    ss -tlnp 2>/dev/null | grep -E '8390|5433|10080' || echo "  No shadow ports in use"
    echo ""
    echo "Database:"
    if docker exec shadow-coordinator-postgres pg_isready -U postgres >/dev/null 2>&1; then
        echo "  PostgreSQL: RUNNING on :5433"
        psql "$SHADOW_DB" -c "SELECT 'batch' as table, COUNT(*) as cnt FROM batch UNION ALL SELECT 'chunk', COUNT(*) FROM chunk UNION ALL SELECT 'bundle', COUNT(*) FROM bundle UNION ALL SELECT 'l2_block', COUNT(*) FROM l2_block;" 2>/dev/null || echo "  (Unable to query)"
    else
        echo "  PostgreSQL: NOT RUNNING"
    fi
    echo ""
    echo "Coordinator API:"
    if curl -s http://localhost:8390/ >/dev/null 2>&1; then
        echo "  Coordinator: RESPONDING on :8390"
    else
        echo "  Coordinator: NOT RESPONDING"
    fi
    echo ""
    echo "Prover:"
    if curl -s http://localhost:10080/health >/dev/null 2>&1; then
        echo "  Prover: RESPONDING on :10080"
    else
        echo "  Prover: NOT RESPONDING"
    fi
}

# Main
if [ $# -eq 0 ]; then
    show_help
    exit 0
fi

while [ $# -gt 0 ]; do
    case "$1" in
        --postgres)
            setup_postgres
            ;;
        --coordinator)
            setup_coordinator
            ;;
        --prover)
            setup_prover
            ;;
        --all)
            setup_postgres
            setup_coordinator
            setup_prover
            ;;
        --stop)
            stop_all
            ;;
        --status)
            show_status
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
    shift
done
