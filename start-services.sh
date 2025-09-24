#!/bin/bash

# Crescendo Services Startup Script
# This script manages backend and frontend services with proper port handling

set -e

echo "🎵 Starting Crescendo Services..."

# Configuration
BACKEND_PREFERRED_PORT=8080
FRONTEND_PREFERRED_PORT=5173
PROJECT_DIR="$(pwd)"
FRONTEND_DIR="$PROJECT_DIR/crescendo-frontend/crescendo-frontend"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to find an available port
find_available_port() {
    local start_port=$1
    local port=$start_port

    while lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; do
        port=$((port + 1))
        if [ $port -gt $((start_port + 50)) ]; then
            log_error "Could not find available port starting from $start_port"
            exit 1
        fi
    done

    echo $port
}

# Function to cleanup processes
cleanup() {
    log_info "Cleaning up processes..."

    # Kill existing Crescendo processes
    pkill -f "crescendo --server" 2>/dev/null && log_info "Killed existing backend processes" || true
    pkill -f "npm run dev" 2>/dev/null && log_info "Killed existing frontend processes" || true

    # Wait for processes to terminate
    sleep 2

    # Force kill if still running
    lsof -ti:$BACKEND_PREFERRED_PORT,$FRONTEND_PREFERRED_PORT | xargs kill -9 2>/dev/null || true

    log_success "Cleanup completed"
}

# Function to start backend
start_backend() {
    log_info "Starting Crescendo backend..."

    # Check if crescendo binary exists
    if [ ! -f "./crescendo" ]; then
        log_error "Crescendo binary not found. Please build it first with: go build -o crescendo"
        exit 1
    fi

    # Find available port for backend
    BACKEND_PORT=$(find_available_port $BACKEND_PREFERRED_PORT)

    if [ $BACKEND_PORT -ne $BACKEND_PREFERRED_PORT ]; then
        log_warning "Port $BACKEND_PREFERRED_PORT is in use, using port $BACKEND_PORT instead"
    fi

    log_info "Starting backend on port $BACKEND_PORT..."

    # Start backend in background
    ./crescendo --server --port $BACKEND_PORT > backend.log 2>&1 &
    BACKEND_PID=$!

    # Wait for backend to start
    log_info "Waiting for backend to start..."
    for i in {1..30}; do
        if curl -s "http://localhost:$BACKEND_PORT/health" > /dev/null 2>&1; then
            log_success "Backend started successfully on port $BACKEND_PORT"
            echo $BACKEND_PORT > .backend-port
            echo $BACKEND_PID > .backend-pid
            return 0
        fi
        sleep 1
    done

    log_error "Backend failed to start within 30 seconds"
    kill $BACKEND_PID 2>/dev/null || true
    exit 1
}

# Function to start frontend
start_frontend() {
    local backend_port=$1

    log_info "Starting Crescendo frontend..."

    # Check if frontend directory exists
    if [ ! -d "$FRONTEND_DIR" ]; then
        log_error "Frontend directory not found: $FRONTEND_DIR"
        exit 1
    fi

    # Check if package.json exists
    if [ ! -f "$FRONTEND_DIR/package.json" ]; then
        log_error "Frontend package.json not found. Please run 'npm install' in the frontend directory"
        exit 1
    fi

    # Find available port for frontend
    FRONTEND_PORT=$(find_available_port $FRONTEND_PREFERRED_PORT)

    if [ $FRONTEND_PORT -ne $FRONTEND_PREFERRED_PORT ]; then
        log_warning "Port $FRONTEND_PREFERRED_PORT is in use, using port $FRONTEND_PORT instead"
    fi

    log_info "Starting frontend on port $FRONTEND_PORT with backend URL: http://localhost:$backend_port"

    # Change to frontend directory and start
    cd "$FRONTEND_DIR"

    # Set environment variables for the frontend
    export VITE_API_URL="http://localhost:$backend_port"
    export VITE_WS_URL="ws://localhost:$backend_port"
    export PORT=$FRONTEND_PORT

    # Start frontend in background
    npm run dev -- --port $FRONTEND_PORT > ../../frontend.log 2>&1 &
    FRONTEND_PID=$!

    # Wait for frontend to start
    log_info "Waiting for frontend to start..."
    for i in {1..30}; do
        if curl -s "http://localhost:$FRONTEND_PORT" > /dev/null 2>&1; then
            log_success "Frontend started successfully on port $FRONTEND_PORT"
            echo $FRONTEND_PORT > ../../.frontend-port
            echo $FRONTEND_PID > ../../.frontend-pid
            cd "$PROJECT_DIR"
            return 0
        fi
        sleep 1
    done

    log_error "Frontend failed to start within 30 seconds"
    kill $FRONTEND_PID 2>/dev/null || true
    cd "$PROJECT_DIR"
    exit 1
}

# Function to display status
show_status() {
    local backend_port=$1
    local frontend_port=$2

    echo ""
    echo "🎵 Crescendo Services Started Successfully! 🎵"
    echo "=============================================="
    echo ""
    log_success "Backend (API + WebSocket): http://localhost:$backend_port"
    log_success "Frontend (Web UI):         http://localhost:$frontend_port"
    echo ""
    echo "📊 Service Information:"
    echo "  • Backend Health:     http://localhost:$backend_port/health"
    echo "  • API Status:         http://localhost:$backend_port/api/status"
    echo "  • WebSocket:          ws://localhost:$backend_port/api/ws/downloads/:jobId"
    echo ""
    echo "📝 Logs:"
    echo "  • Backend logs:       tail -f backend.log"
    echo "  • Frontend logs:      tail -f frontend.log"
    echo ""
    echo "🛑 To stop services:   ./stop-services.sh"
    echo ""
}

# Main execution
main() {
    # Handle cleanup on script exit
    trap cleanup EXIT

    # Parse command line arguments
    if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
        echo "Usage: $0 [--clean]"
        echo ""
        echo "Options:"
        echo "  --clean    Clean up existing processes before starting"
        echo "  --help     Show this help message"
        exit 0
    fi

    # Clean up existing processes if requested
    if [ "$1" = "--clean" ]; then
        cleanup
    fi

    # Start services
    start_backend
    BACKEND_PORT=$(cat .backend-port)

    start_frontend $BACKEND_PORT
    FRONTEND_PORT=$(cat .frontend-port)

    # Show status
    show_status $BACKEND_PORT $FRONTEND_PORT

    # Wait for services (keep script running)
    log_info "Services running. Press Ctrl+C to stop."
    wait
}

# Run main function
main "$@"