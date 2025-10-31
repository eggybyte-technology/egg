#!/bin/bash
#
# Test Configuration
#
# Defines test configuration constants and variables.
# This file is sourced by other test scripts.

# Script directory detection
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECT_ROOT="$(cd "$CLI_ROOT/.." && pwd)"

# Source the unified logger from project root
source "$PROJECT_ROOT/scripts/logger.sh"

# Source test helpers (must be sourced after logger.sh)
source "$SCRIPT_DIR/test-helpers.sh"

# Test configuration
PROJECT_NAME="test-project"
TEST_WORKSPACE="$CLI_ROOT/tmp"  # Tests run in cli/tmp/
TEST_DIR="$TEST_WORKSPACE/$PROJECT_NAME"  # Full path to test project
BACKEND_SERVICE="user"  # Main service with CRUD proto
BACKEND_PING_SERVICE="ping"  # Secondary service with echo proto
FRONTEND_SERVICE="admin_portal"  # Use underscore for Dart compatibility
KEEP_TEST_DIR=true  # Default to keep test directory

# Service ports (from egg.yaml)
PING_HTTP_PORT=8090
PING_HEALTH_PORT=8091
PING_METRICS_PORT=9092
USER_HTTP_PORT=8080
USER_HEALTH_PORT=8081
USER_METRICS_PORT=9091
FRONTEND_PORT=3000

# Connect RPC paths
PING_SERVICE_PATH="/eggybyte_test.test_project.ping.v1.PingService/Ping"
USER_SERVICE_CREATE="/eggybyte_test.test_project.user.v1.UserService/CreateUser"
USER_SERVICE_GET="/eggybyte_test.test_project.user.v1.UserService/GetUser"
USER_SERVICE_UPDATE="/eggybyte_test.test_project.user.v1.UserService/UpdateUser"
USER_SERVICE_DELETE="/eggybyte_test.test_project.user.v1.UserService/DeleteUser"
USER_SERVICE_LIST="/eggybyte_test.test_project.user.v1.UserService/ListUsers"

# Get absolute path to egg CLI (should be already built by Makefile)
EGG_CLI="$CLI_ROOT/bin/egg"

# Parse command line arguments
for arg in "$@"; do
  case $arg in
    --remove)
      KEEP_TEST_DIR=false
      shift
      ;;
  esac
done

