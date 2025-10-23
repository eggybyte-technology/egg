#!/bin/bash
# Test script to verify egg log format compliance

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}=== Testing Egg Log Format ===${NC}\n"

# Test 1: Build minimal service
echo -e "${YELLOW}Building minimal-connect-service...${NC}"
cd examples/minimal-connect-service
go build -o bin/greet-service main.go
echo -e "${GREEN}✓ Build successful${NC}\n"

# Test 2: Start service and capture logs
echo -e "${YELLOW}Starting service and capturing logs...${NC}"
LOG_COLOR=false ./bin/greet-service > /tmp/greet-service.log 2>&1 &
SERVICE_PID=$!
sleep 2

# Kill service
kill $SERVICE_PID 2>/dev/null || true
sleep 1

# Test 3: Analyze log format
echo -e "${YELLOW}Analyzing log format...${NC}\n"

echo -e "${CYAN}Sample logs:${NC}"
head -20 /tmp/greet-service.log
echo ""

# Verify format requirements
echo -e "${CYAN}Format validation:${NC}"

# Check 1: Single line format
MULTILINE=$(grep -E '^\s+' /tmp/greet-service.log || true)
if [ -z "$MULTILINE" ]; then
    echo -e "${GREEN}✓ All logs are single-line${NC}"
else
    echo -e "${RED}✗ Found multi-line logs${NC}"
fi

# Check 2: level=XXX format
LEVEL_COUNT=$(grep -c 'level=' /tmp/greet-service.log || true)
TOTAL_LINES=$(wc -l < /tmp/greet-service.log)
if [ "$LEVEL_COUNT" -eq "$TOTAL_LINES" ]; then
    echo -e "${GREEN}✓ All logs have level field${NC}"
else
    echo -e "${RED}✗ Some logs missing level field${NC}"
fi

# Check 3: msg="..." format (quoted)
MSG_QUOTED=$(grep -c 'msg="[^"]*"' /tmp/greet-service.log || true)
if [ "$MSG_QUOTED" -eq "$TOTAL_LINES" ]; then
    echo -e "${GREEN}✓ All messages are properly quoted${NC}"
else
    echo -e "${RED}✗ Some messages not properly quoted${NC}"
fi

# Check 4: No timestamps (time= should not appear)
TIME_COUNT=$(grep -c 'time=' /tmp/greet-service.log || true)
if [ "$TIME_COUNT" -eq "0" ]; then
    echo -e "${GREEN}✓ No timestamps in logs (good for containers)${NC}"
else
    echo -e "${YELLOW}⚠ Found $TIME_COUNT logs with timestamps${NC}"
fi

# Check 5: service field present
SERVICE_COUNT=$(grep -c 'service=' /tmp/greet-service.log || true)
if [ "$SERVICE_COUNT" -gt "0" ]; then
    echo -e "${GREEN}✓ Service field present${NC}"
else
    echo -e "${RED}✗ Service field missing${NC}"
fi

# Check 6: version field present
VERSION_COUNT=$(grep -c 'version=' /tmp/greet-service.log || true)
if [ "$VERSION_COUNT" -gt "0" ]; then
    echo -e "${GREEN}✓ Version field present${NC}"
else
    echo -e "${RED}✗ Version field missing${NC}"
fi

# Check 7: No brackets [key value] format
BRACKET_COUNT=$(grep -c '\[.*=.*\]' /tmp/greet-service.log || true)
if [ "$BRACKET_COUNT" -eq "0" ]; then
    echo -e "${GREEN}✓ No bracket notation found${NC}"
else
    echo -e "${RED}✗ Found $BRACKET_COUNT logs with bracket notation${NC}"
fi

echo -e "\n${CYAN}=== Log Format Test Complete ===${NC}"

# Cleanup
rm -f /tmp/greet-service.log

cd ../..

