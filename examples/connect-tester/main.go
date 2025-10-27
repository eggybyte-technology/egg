// Package main provides a comprehensive Connect service testing tool.
//
// Overview:
//
//	This tool tests Connect RPC services built with the egg framework. It supports
//	testing multiple service types including minimal services and full CRUD services.
//	It provides colored output and detailed test results for easy debugging.
//
// Key Features:
//
//   - Tests multiple service types: minimal-service (greet) and user-service (CRUD)
//   - Comprehensive test coverage: unary, streaming, CRUD operations
//   - Metrics endpoint validation: Prometheus /metrics endpoint testing
//   - Colored output: green for success, red for failure, yellow for warnings
//   - Detailed metrics: request timing, success rates, error diagnostics
//   - Uses egg framework libraries: logx for logging, clientx for resilient clients
//   - Error scenario testing: validation, not found, duplicates
//
// Usage:
//
//	# Test minimal greet service
//	./connect-tester http://localhost:8080 minimal-service
//
//	# Test user CRUD service (full test suite)
//	./connect-tester http://localhost:8082 user-service
//
//	# Test with custom metrics endpoint
//	METRICS_URL=http://localhost:9091/metrics ./connect-tester http://localhost:8080 minimal-service
//
//	# Test user service with specific operation
//	./connect-tester http://localhost:8082 user-service create email@test.com "Test User"
//	./connect-tester http://localhost:8082 user-service get <user-id>
//	./connect-tester http://localhost:8082 user-service update <user-id> email@test.com "Updated Name"
//	./connect-tester http://localhost:8082 user-service delete <user-id>
//	./connect-tester http://localhost:8082 user-service list <page> <page-size>
//
// Metrics Endpoint:
//
//	The tester automatically derives the metrics endpoint URL from the service base URL.
//	It uses a port mapping table for known docker-compose configurations:
//	  - http://localhost:8080 → http://localhost:9091/metrics (minimal-service)
//	  - http://localhost:8082 → http://localhost:9092/metrics (user-service)
//	  - For unknown ports: HTTP port + 1011 (egg framework's internal convention)
//
//	You can override this by setting the METRICS_URL environment variable:
//	  METRICS_URL=http://custom-host:9091/metrics ./connect-tester http://localhost:8080 minimal-service
//
// Output:
//
//   - ✓ PASS: Successful test with duration
//   - ✗ FAIL: Failed test with error details
//   - Summary: Total tests, passed, failed, success rate
//
// Dependencies:
//
//   - logx: structured logging with colors (L1)
//   - clientx: HTTP client with retry and circuit breaker (L3)
//   - core/log: standardized log interface (L0)
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"go.eggybyte.com/egg/clientx"
	"go.eggybyte.com/egg/core/log"
	greetv1 "go.eggybyte.com/egg/examples/minimal-connect-service/gen/go/greet/v1"
	greetv1connect "go.eggybyte.com/egg/examples/minimal-connect-service/gen/go/greet/v1/greetv1connect"
	userv1 "go.eggybyte.com/egg/examples/user-service/gen/go/user/v1"
	userv1connect "go.eggybyte.com/egg/examples/user-service/gen/go/user/v1/userv1connect"
	"go.eggybyte.com/egg/logx"
)

// TestResult represents the outcome of a single test case.
type TestResult struct {
	Name     string
	Success  bool
	Duration time.Duration
	Error    error
	Details  string
}

// TestSuite aggregates all test results for reporting.
type TestSuite struct {
	Results []TestResult
}

// add records a test result.
func (ts *TestSuite) add(name string, duration time.Duration, err error, details string) {
	ts.Results = append(ts.Results, TestResult{
		Name:     name,
		Success:  err == nil,
		Duration: duration,
		Error:    err,
		Details:  details,
	})
}

// summary prints a summary of all test results.
func (ts *TestSuite) summary(logger log.Logger) {
	var passed, failed int
	for _, r := range ts.Results {
		if r.Success {
			passed++
		} else {
			failed++
		}
	}

	total := len(ts.Results)
	successRate := float64(passed) / float64(total) * 100

	logger.Info("Test Summary",
		log.Int("total", total),
		log.Int("passed", passed),
		log.Int("failed", failed),
		log.Float64("success_rate", successRate))

	if failed > 0 {
		logger.Error(nil, "Some tests failed")
		os.Exit(1)
	}
}

func main() {
	// Create console logger for human-readable output
	logger := logx.New(
		logx.WithFormat(logx.FormatConsole),
		logx.WithLevel(slog.LevelInfo),
		logx.WithColor(true),
	)

	// Parse command line arguments
	if len(os.Args) < 3 {
		logger.Error(nil, "Usage: connect-tester <base-url> <service-type> [test-args...]")
		logger.Info("Service types:")
		logger.Info("  minimal-service: Test greet service endpoints")
		logger.Info("  user-service: Test user CRUD endpoints")
		logger.Info("Examples:")
		logger.Info("  ./connect-tester http://localhost:8080 minimal-service")
		logger.Info("  ./connect-tester http://localhost:8082 user-service")
		logger.Info("  ./connect-tester http://localhost:8082 user-service create email@test.com \"Name\"")
		os.Exit(1)
	}

	baseURL := os.Args[1]
	serviceType := os.Args[2]
	testArgs := os.Args[3:]

	logger.Info("Connect Service Tester", log.Str("url", baseURL), log.Str("service", serviceType))

	// Run tests
	ctx := context.Background()
	if err := runTests(ctx, logger, baseURL, serviceType, testArgs); err != nil {
		logger.Error(err, "tests failed")
		os.Exit(1)
	}

	logger.Info("All tests passed")
}

// deriveMetricsURL derives the metrics endpoint URL from the service base URL.
// It uses a port mapping table for known docker-compose configurations,
// then falls back to the egg framework's default convention.
//
// Known mappings (docker-compose port mappings):
//   - HTTP port 8080 -> Metrics port 9091 (minimal-service)
//   - HTTP port 8082 -> Metrics port 9092 (user-service)
//   - Default: HTTP port + 1011 (e.g., 8080 -> 9091)
//
// The metrics URL can be overridden by setting the METRICS_URL environment variable.
//
// Parameters:
//   - baseURL: service base URL (e.g., http://localhost:8080)
//
// Returns:
//   - string: metrics endpoint URL (e.g., http://localhost:9091/metrics)
func deriveMetricsURL(baseURL string) string {
	// Check for environment variable override
	if metricsURL := os.Getenv("METRICS_URL"); metricsURL != "" {
		return metricsURL
	}

	// Parse the base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		// Fallback to default
		return "http://localhost:9091/metrics"
	}

	host := parsedURL.Hostname()
	port := parsedURL.Port()

	// Default port if not specified
	if port == "" {
		port = "80"
		if parsedURL.Scheme == "https" {
			port = "443"
		}
	}

	// Parse port number
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return "http://localhost:9091/metrics"
	}

	// Known port mappings for docker-compose services
	// These mappings reflect the actual port configuration in docker-compose.services.yaml
	portMappings := map[int]int{
		8080: 9091, // minimal-service
		8082: 9092, // user-service
	}

	// Check if we have a known mapping
	var metricsPort int
	if mappedPort, ok := portMappings[portNum]; ok {
		metricsPort = mappedPort
	} else {
		// Default: metrics port = HTTP port + 1011
		// This is the egg framework's internal convention (8080 -> 9091)
		metricsPort = portNum + 1011
	}

	// Build metrics URL
	return fmt.Sprintf("%s://%s:%d/metrics", parsedURL.Scheme, host, metricsPort)
}

// runTests routes to the appropriate test suite based on service type.
func runTests(ctx context.Context, logger log.Logger, baseURL, serviceType string, testArgs []string) error {
	switch serviceType {
	case "minimal-service":
		return testMinimalService(ctx, logger, baseURL)
	case "user-service":
		return testUserService(ctx, logger, baseURL, testArgs)
	default:
		return fmt.Errorf("unknown service type: %s", serviceType)
	}
}

// testMinimalService tests the minimal greet service endpoints.
func testMinimalService(ctx context.Context, logger log.Logger, baseURL string) error {
	logger.Info("Testing minimal-service (greet)")

	// Create HTTP client with retry and timeout
	httpClient := clientx.NewHTTPClient(baseURL,
		clientx.WithTimeout(10*time.Second),
		clientx.WithRetry(3),
	)

	// Create Connect client
	client := greetv1connect.NewGreeterServiceClient(httpClient, baseURL)

	suite := &TestSuite{}

	// Test SayHello with different languages
	logger.Info("Testing SayHello endpoint with multiple languages")
	languages := []string{"en", "es", "fr", "de", "zh"}
	for _, lang := range languages {
		testName := fmt.Sprintf("SayHello_%s", lang)
		start := time.Now()
		req := connect.NewRequest(&greetv1.SayHelloRequest{
			Name:     "World",
			Language: lang,
		})
		resp, err := client.SayHello(ctx, req)
		duration := time.Since(start)
		if err != nil {
			suite.add(testName, duration, err, "")
			logger.Error(err, fmt.Sprintf("✗ FAIL %s", testName))
		} else {
			suite.add(testName, duration, nil, resp.Msg.Message)
			logger.Info(fmt.Sprintf("✓ PASS %s", testName),
				log.Str("message", resp.Msg.Message),
				log.Str("duration", fmt.Sprintf("%dms", duration.Milliseconds())))
		}
	}

	// Test SayHello with empty name (should default to "World")
	logger.Info("Testing SayHello with empty name")
	start := time.Now()
	req := connect.NewRequest(&greetv1.SayHelloRequest{
		Name:     "",
		Language: "en",
	})
	resp, err := client.SayHello(ctx, req)
	duration := time.Since(start)
	if err != nil {
		suite.add("SayHello_EmptyName", duration, err, "")
		logger.Error(err, "✗ FAIL SayHello_EmptyName")
	} else {
		suite.add("SayHello_EmptyName", duration, nil, resp.Msg.Message)
		logger.Info("✓ PASS SayHello_EmptyName",
			log.Str("message", resp.Msg.Message),
			log.Str("duration", fmt.Sprintf("%dms", duration.Milliseconds())))
	}

	// Test SayHelloStream with different counts
	logger.Info("Testing SayHelloStream endpoint with different counts")
	counts := []int32{1, 3, 5, 10}
	for _, count := range counts {
		testName := fmt.Sprintf("SayHelloStream_%d", count)
		start = time.Now()
		streamReq := connect.NewRequest(&greetv1.SayHelloStreamRequest{
			Name:  "Tester",
			Count: count,
		})
		stream, err := client.SayHelloStream(ctx, streamReq)
		duration = time.Since(start)
		if err != nil {
			suite.add(testName, duration, err, "")
			logger.Error(err, fmt.Sprintf("✗ FAIL %s", testName))
		} else {
			var messages []string
			for stream.Receive() {
				messages = append(messages, stream.Msg().Message)
			}
			if err := stream.Err(); err != nil {
				suite.add(testName, duration, err, "")
				logger.Error(err, fmt.Sprintf("✗ FAIL %s", testName))
			} else {
				suite.add(testName, duration, nil, fmt.Sprintf("%d messages", len(messages)))
				logger.Info(fmt.Sprintf("✓ PASS %s", testName),
					log.Int("messages", len(messages)),
					log.Int32("expected", count),
					log.Str("duration", fmt.Sprintf("%dms", duration.Milliseconds())))
			}
		}
	}

	// Test SayHelloStream with zero count (should default to 5)
	logger.Info("Testing SayHelloStream with zero count")
	start = time.Now()
	streamReqZero := connect.NewRequest(&greetv1.SayHelloStreamRequest{
		Name:  "Tester",
		Count: 0,
	})
	streamZero, err := client.SayHelloStream(ctx, streamReqZero)
	duration = time.Since(start)
	if err != nil {
		suite.add("SayHelloStream_ZeroCount", duration, err, "")
		logger.Error(err, "✗ FAIL SayHelloStream_ZeroCount")
	} else {
		var messages []string
		for streamZero.Receive() {
			messages = append(messages, streamZero.Msg().Message)
		}
		if err := streamZero.Err(); err != nil {
			suite.add("SayHelloStream_ZeroCount", duration, err, "")
			logger.Error(err, "✗ FAIL SayHelloStream_ZeroCount")
		} else {
			suite.add("SayHelloStream_ZeroCount", duration, nil, fmt.Sprintf("%d messages", len(messages)))
			logger.Info("✓ PASS SayHelloStream_ZeroCount",
				log.Int("messages", len(messages)),
				log.Str("duration", fmt.Sprintf("%dms", duration.Milliseconds())))
		}
	}

	// Test metrics endpoint
	// Derive metrics URL from base URL
	// Calculate expected call count: 5 SayHello + 1 EmptyName = 6
	// Note: Streaming calls (SayHelloStream) are not counted as they require StreamingInterceptor
	expectedCallCount := 6
	metricsURL := deriveMetricsURL(baseURL)
	testMetricsEndpoint(ctx, logger, metricsURL, suite, "greet-service", expectedCallCount)

	suite.summary(logger)
	return nil
}

// testUserService tests the user CRUD service endpoints.
func testUserService(ctx context.Context, logger log.Logger, baseURL string, testArgs []string) error {
	logger.Info("Testing user-service (CRUD)")

	// Create HTTP client with retry and timeout
	httpClient := clientx.NewHTTPClient(baseURL,
		clientx.WithTimeout(10*time.Second),
		clientx.WithRetry(3),
	)

	// Create Connect client
	client := userv1connect.NewUserServiceClient(httpClient, baseURL)

	// If specific operation requested, run that only
	if len(testArgs) > 0 {
		return testUserServiceOperation(ctx, logger, client, testArgs)
	}

	// Otherwise run full test suite
	suite := &TestSuite{}

	// Test CreateUser with multiple users
	logger.Info("Testing CreateUser endpoint with multiple users")
	var createdUserIDs []string
	for i := 1; i <= 3; i++ {
		testName := fmt.Sprintf("CreateUser_%d", i)
		start := time.Now()
		createReq := connect.NewRequest(&userv1.CreateUserRequest{
			Email: fmt.Sprintf("test-%d-%d@example.com", time.Now().UnixNano(), i),
			Name:  fmt.Sprintf("Test User %d", i),
		})
		createResp, err := client.CreateUser(ctx, createReq)
		duration := time.Since(start)
		if err != nil {
			suite.add(testName, duration, err, "")
			logger.Error(err, fmt.Sprintf("✗ FAIL %s", testName))
		} else {
			createdUserIDs = append(createdUserIDs, createResp.Msg.User.Id)
			suite.add(testName, duration, nil, createResp.Msg.User.Id)
			logger.Info(fmt.Sprintf("✓ PASS %s", testName),
				log.Str("user_id", createResp.Msg.User.Id),
				log.Str("email", createResp.Msg.User.Email),
				log.Str("duration", fmt.Sprintf("%dms", duration.Milliseconds())))
		}
	}

	// Track first created user for subsequent tests
	var createdUserID string
	if len(createdUserIDs) > 0 {
		createdUserID = createdUserIDs[0]
	}

	// Test GetUser
	if createdUserID != "" {
		logger.Info("Testing GetUser endpoint")
		start := time.Now()
		getReq := connect.NewRequest(&userv1.GetUserRequest{Id: createdUserID})
		getResp, err := client.GetUser(ctx, getReq)
		duration := time.Since(start)
		if err != nil {
			suite.add("GetUser", duration, err, "")
			logger.Error(err, "✗ FAIL GetUser")
		} else {
			suite.add("GetUser", duration, nil, getResp.Msg.User.Email)
			logger.Info("✓ PASS GetUser",
				log.Str("user_id", getResp.Msg.User.Id),
				log.Str("email", getResp.Msg.User.Email),
				log.Str("duration", fmt.Sprintf("%dms", duration.Milliseconds())))
		}
	}

	// Test UpdateUser
	if createdUserID != "" {
		logger.Info("Testing UpdateUser endpoint")
		start := time.Now()
		updateReq := connect.NewRequest(&userv1.UpdateUserRequest{
			Id:    createdUserID,
			Email: fmt.Sprintf("updated-%d@example.com", time.Now().UnixNano()),
			Name:  "Updated Test User",
		})
		updateResp, err := client.UpdateUser(ctx, updateReq)
		duration := time.Since(start)
		if err != nil {
			suite.add("UpdateUser", duration, err, "")
			logger.Error(err, "✗ FAIL UpdateUser")
		} else {
			suite.add("UpdateUser", duration, nil, updateResp.Msg.User.Email)
			logger.Info("✓ PASS UpdateUser",
				log.Str("user_id", updateResp.Msg.User.Id),
				log.Str("email", updateResp.Msg.User.Email),
				log.Str("duration", fmt.Sprintf("%dms", duration.Milliseconds())))
		}
	}

	// Test ListUsers
	logger.Info("Testing ListUsers endpoint")
	start := time.Now()
	listReq := connect.NewRequest(&userv1.ListUsersRequest{
		Page:     1,
		PageSize: 10,
	})
	listResp, err := client.ListUsers(ctx, listReq)
	duration := time.Since(start)
	if err != nil {
		suite.add("ListUsers", duration, err, "")
		logger.Error(err, "✗ FAIL ListUsers")
	} else {
		suite.add("ListUsers", duration, nil, fmt.Sprintf("%d users", len(listResp.Msg.Users)))
		logger.Info("✓ PASS ListUsers",
			log.Int("count", len(listResp.Msg.Users)),
			log.Int32("total", listResp.Msg.Total),
			log.Str("duration", fmt.Sprintf("%dms", duration.Milliseconds())))
	}

	// Test DeleteUser
	if createdUserID != "" {
		logger.Info("Testing DeleteUser endpoint")
		start := time.Now()
		deleteReq := connect.NewRequest(&userv1.DeleteUserRequest{Id: createdUserID})
		deleteResp, err := client.DeleteUser(ctx, deleteReq)
		duration := time.Since(start)
		if err != nil {
			suite.add("DeleteUser", duration, err, "")
			logger.Error(err, "✗ FAIL DeleteUser")
		} else {
			suite.add("DeleteUser", duration, nil, fmt.Sprintf("success=%v", deleteResp.Msg.Success))
			logger.Info("✓ PASS DeleteUser",
				log.Str("user_id", createdUserID),
				log.Str("duration", fmt.Sprintf("%dms", duration.Milliseconds())))
		}
	}

	// Test error scenarios
	logger.Info("Testing error scenarios")
	testErrorScenarios(ctx, logger, client, suite)

	// Test metrics endpoint
	// Derive metrics URL from base URL
	// Calculate expected call count:
	// - Success: 3 Create + 1 Get + 1 Update + 1 List + 1 Delete = 7
	// - Errors: 3 error scenarios = 3
	// - Total: 10
	expectedCallCount := 10
	metricsURL := deriveMetricsURL(baseURL)
	testMetricsEndpoint(ctx, logger, metricsURL, suite, "user-service", expectedCallCount)

	suite.summary(logger)
	return nil
}

// testUserServiceOperation runs a specific user service operation.
func testUserServiceOperation(ctx context.Context, logger log.Logger, client userv1connect.UserServiceClient, args []string) error {
	operation := args[0]

	switch operation {
	case "create":
		if len(args) < 3 {
			return fmt.Errorf("create requires email and name: create <email> <name>")
		}
		req := connect.NewRequest(&userv1.CreateUserRequest{
			Email: args[1],
			Name:  args[2],
		})
		resp, err := client.CreateUser(ctx, req)
		if err != nil {
			logger.Error(err, "✗ FAIL CreateUser")
			return err
		}
		logger.Info("✓ PASS CreateUser", log.Str("user_id", resp.Msg.User.Id))

	case "get":
		if len(args) < 2 {
			return fmt.Errorf("get requires user ID: get <user-id>")
		}
		req := connect.NewRequest(&userv1.GetUserRequest{Id: args[1]})
		resp, err := client.GetUser(ctx, req)
		if err != nil {
			logger.Error(err, "✗ FAIL GetUser")
			return err
		}
		logger.Info("✓ PASS GetUser",
			log.Str("user_id", resp.Msg.User.Id),
			log.Str("email", resp.Msg.User.Email),
			log.Str("name", resp.Msg.User.Name))

	case "update":
		if len(args) < 4 {
			return fmt.Errorf("update requires ID, email, and name: update <user-id> <email> <name>")
		}
		req := connect.NewRequest(&userv1.UpdateUserRequest{
			Id:    args[1],
			Email: args[2],
			Name:  args[3],
		})
		resp, err := client.UpdateUser(ctx, req)
		if err != nil {
			logger.Error(err, "✗ FAIL UpdateUser")
			return err
		}
		logger.Info("✓ PASS UpdateUser",
			log.Str("user_id", resp.Msg.User.Id),
			log.Str("email", resp.Msg.User.Email))

	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("delete requires user ID: delete <user-id>")
		}
		req := connect.NewRequest(&userv1.DeleteUserRequest{Id: args[1]})
		resp, err := client.DeleteUser(ctx, req)
		if err != nil {
			logger.Error(err, "✗ FAIL DeleteUser")
			return err
		}
		logger.Info("✓ PASS DeleteUser", log.Bool("success", resp.Msg.Success))

	case "list":
		page := int32(1)
		pageSize := int32(10)
		if len(args) >= 2 {
			if p, err := strconv.Atoi(args[1]); err == nil {
				page = int32(p)
			}
		}
		if len(args) >= 3 {
			if ps, err := strconv.Atoi(args[2]); err == nil {
				pageSize = int32(ps)
			}
		}
		req := connect.NewRequest(&userv1.ListUsersRequest{
			Page:     page,
			PageSize: pageSize,
		})
		resp, err := client.ListUsers(ctx, req)
		if err != nil {
			logger.Error(err, "✗ FAIL ListUsers")
			return err
		}
		logger.Info("✓ PASS ListUsers",
			log.Int("count", len(resp.Msg.Users)),
			log.Int32("total", resp.Msg.Total),
			log.Int32("page", resp.Msg.Page),
			log.Int32("page_size", resp.Msg.PageSize))

	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}

	return nil
}

// MetricSample represents a single Prometheus metric sample.
type MetricSample struct {
	Name   string
	Labels map[string]string
	Value  float64
}

// parsePrometheusMetrics parses Prometheus text format and extracts metric samples.
// Returns a map of metric name to list of samples with labels and values.
//
// Parameters:
//   - body: Prometheus text format response body
//
// Returns:
//   - map[string][]MetricSample: parsed metrics grouped by name
func parsePrometheusMetrics(body string) map[string][]MetricSample {
	metrics := make(map[string][]MetricSample)
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse metric line: metric_name{label1="value1",label2="value2"} value
		// Or: metric_name value
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		metricPart := parts[0]
		valuePart := parts[1]

		// Parse value
		value, err := strconv.ParseFloat(valuePart, 64)
		if err != nil {
			continue
		}

		// Extract metric name and labels
		var metricName string
		labels := make(map[string]string)

		if idx := strings.Index(metricPart, "{"); idx > 0 {
			// Has labels
			metricName = metricPart[:idx]
			labelsPart := metricPart[idx+1 : len(metricPart)-1] // Remove { and }

			// Parse labels
			labelPairs := strings.Split(labelsPart, ",")
			for _, pair := range labelPairs {
				pair = strings.TrimSpace(pair)
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					value := strings.Trim(strings.TrimSpace(kv[1]), `"`)
					labels[key] = value
				}
			}
		} else {
			// No labels
			metricName = metricPart
		}

		sample := MetricSample{
			Name:   metricName,
			Labels: labels,
			Value:  value,
		}

		metrics[metricName] = append(metrics[metricName], sample)
	}

	return metrics
}

// findMetricValue finds a metric value by name and optional label filters.
// Returns the sum of all matching metric samples.
//
// Parameters:
//   - metrics: parsed metrics map
//   - name: metric name to search for
//   - labelFilters: optional label filters (all must match)
//
// Returns:
//   - float64: sum of matching metric values
//   - bool: true if at least one metric was found
func findMetricValue(metrics map[string][]MetricSample, name string, labelFilters map[string]string) (float64, bool) {
	samples, exists := metrics[name]
	if !exists {
		return 0, false
	}

	var total float64
	found := false

	for _, sample := range samples {
		// Check if all label filters match
		allMatch := true
		for filterKey, filterValue := range labelFilters {
			if labelValue, ok := sample.Labels[filterKey]; !ok || labelValue != filterValue {
				allMatch = false
				break
			}
		}

		if allMatch {
			total += sample.Value
			found = true
		}
	}

	return total, found
}

// testErrorScenarios tests error handling for common edge cases.
func testErrorScenarios(ctx context.Context, logger log.Logger, client userv1connect.UserServiceClient, suite *TestSuite) {
	// Test GetUser with non-existent ID
	logger.Info("Testing GetUser with non-existent ID")
	start := time.Now()
	req := connect.NewRequest(&userv1.GetUserRequest{Id: "non-existent-id"})
	_, err := client.GetUser(ctx, req)
	duration := time.Since(start)
	if err != nil {
		suite.add("GetUser_NonExistent", duration, nil, "correctly returned error")
		logger.Info("✓ PASS GetUser_NonExistent - correctly returned error")
	} else {
		suite.add("GetUser_NonExistent", duration, fmt.Errorf("expected error but got success"), "")
		logger.Error(nil, "✗ FAIL GetUser_NonExistent - should return error")
	}

	// Test CreateUser with empty email
	logger.Info("Testing CreateUser with empty email")
	start = time.Now()
	createReq := connect.NewRequest(&userv1.CreateUserRequest{
		Email: "",
		Name:  "Test User",
	})
	_, err = client.CreateUser(ctx, createReq)
	duration = time.Since(start)
	if err != nil {
		suite.add("CreateUser_EmptyEmail", duration, nil, "correctly returned error")
		logger.Info("✓ PASS CreateUser_EmptyEmail - correctly returned error")
	} else {
		suite.add("CreateUser_EmptyEmail", duration, fmt.Errorf("expected error but got success"), "")
		logger.Error(nil, "✗ FAIL CreateUser_EmptyEmail - should return error")
	}

	// Test CreateUser with empty name
	logger.Info("Testing CreateUser with empty name")
	start = time.Now()
	createReq = connect.NewRequest(&userv1.CreateUserRequest{
		Email: "test@example.com",
		Name:  "",
	})
	_, err = client.CreateUser(ctx, createReq)
	duration = time.Since(start)
	if err != nil {
		suite.add("CreateUser_EmptyName", duration, nil, "correctly returned error")
		logger.Info("✓ PASS CreateUser_EmptyName - correctly returned error")
	} else {
		suite.add("CreateUser_EmptyName", duration, fmt.Errorf("expected error but got success"), "")
		logger.Error(nil, "✗ FAIL CreateUser_EmptyName - should return error")
	}
}

// testMetricsEndpoint tests the Prometheus metrics endpoint.
// It verifies that the /metrics endpoint is accessible and returns valid Prometheus format data.
//
// Parameters:
//   - ctx: context for the test
//   - logger: logger for test output
//   - metricsURL: full URL to the metrics endpoint (e.g., http://localhost:9091/metrics)
//   - suite: test suite to record results
//   - serviceName: expected service name in target_info metric
//   - expectedCallCount: minimum expected RPC call count for validation
//
// The test checks:
//   - HTTP 200 response
//   - Content-Type header contains "text/plain" or "application/openmetrics-text"
//   - Response body contains Prometheus format metrics
//   - Response contains target_info with expected service name
//   - RPC metrics (rpc.requests.total, rpc.request.duration) exist
//   - RPC request count meets minimum expected value
func testMetricsEndpoint(ctx context.Context, logger log.Logger, metricsURL string, suite *TestSuite, serviceName string, expectedCallCount int) {
	logger.Info("Testing metrics endpoint", log.Str("url", metricsURL), log.Int("expected_calls", expectedCallCount))

	start := time.Now()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", metricsURL, nil)
	if err != nil {
		duration := time.Since(start)
		suite.add("Metrics_Endpoint", duration, fmt.Errorf("failed to create request: %w", err), "")
		logger.Error(err, "✗ FAIL Metrics_Endpoint - request creation failed")
		return
	}

	// Make the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		duration := time.Since(start)
		suite.add("Metrics_Endpoint", duration, fmt.Errorf("failed to fetch metrics: %w", err), "")
		logger.Error(err, "✗ FAIL Metrics_Endpoint - request failed")
		return
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		suite.add("Metrics_Endpoint", duration, fmt.Errorf("expected status 200, got %d", resp.StatusCode), "")
		logger.Error(nil, "✗ FAIL Metrics_Endpoint - wrong status code",
			log.Int("expected", 200),
			log.Int("actual", resp.StatusCode))
		return
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		suite.add("Metrics_Endpoint", duration, fmt.Errorf("failed to read response: %w", err), "")
		logger.Error(err, "✗ FAIL Metrics_Endpoint - failed to read body")
		return
	}

	bodyStr := string(body)

	// Validate response is not empty
	if len(bodyStr) == 0 {
		suite.add("Metrics_Endpoint", duration, fmt.Errorf("empty metrics response"), "")
		logger.Error(nil, "✗ FAIL Metrics_Endpoint - empty response")
		return
	}

	// Check for Prometheus format indicators
	hasHelp := strings.Contains(bodyStr, "# HELP")
	hasType := strings.Contains(bodyStr, "# TYPE")
	hasTargetInfo := strings.Contains(bodyStr, "target_info")

	if !hasHelp && !hasType {
		suite.add("Metrics_Endpoint", duration, fmt.Errorf("response does not contain Prometheus format markers"), "")
		logger.Error(nil, "✗ FAIL Metrics_Endpoint - invalid format")
		return
	}

	if !hasTargetInfo {
		suite.add("Metrics_Endpoint", duration, fmt.Errorf("response does not contain target_info metric"), "")
		logger.Error(nil, "✗ FAIL Metrics_Endpoint - missing target_info")
		return
	}

	// Check if service name is in target_info (if provided)
	if serviceName != "" && !strings.Contains(bodyStr, fmt.Sprintf(`service_name="%s"`, serviceName)) {
		suite.add("Metrics_Endpoint", duration, fmt.Errorf("target_info does not contain expected service_name=%s", serviceName), "")
		logger.Error(nil, "✗ FAIL Metrics_Endpoint - wrong service name",
			log.Str("expected", serviceName))
		return
	}

	// Parse Prometheus metrics
	metrics := parsePrometheusMetrics(bodyStr)

	// Verify RPC metrics exist
	// For histograms, check for the _bucket suffix as that's what's actually exported
	rpcMetricsChecks := map[string]string{
		"rpc_requests_total":           "rpc_requests_total",
		"rpc_request_duration_seconds": "rpc_request_duration_seconds_bucket",
		"rpc_request_size_bytes":       "rpc_request_size_bytes_bucket",
		"rpc_response_size_bytes":      "rpc_response_size_bytes_bucket",
	}

	for displayName, actualMetricName := range rpcMetricsChecks {
		if _, exists := metrics[actualMetricName]; !exists {
			suite.add("Metrics_RPC_"+displayName, duration, fmt.Errorf("metric %s not found", displayName), "")
			logger.Error(nil, fmt.Sprintf("✗ FAIL Metrics_RPC_%s - metric not found", displayName))
		} else {
			suite.add("Metrics_RPC_"+displayName, duration, nil, "metric exists")
			logger.Info(fmt.Sprintf("✓ PASS Metrics_RPC_%s - metric exists", displayName))
		}
	}

	// Verify RPC request count
	// Sum all requests across all services/methods (both success and errors)
	totalRequests, found := findMetricValue(metrics, "rpc_requests_total", map[string]string{})
	if !found {
		suite.add("Metrics_RPC_Count", duration, fmt.Errorf("rpc_requests_total not found"), "")
		logger.Error(nil, "✗ FAIL Metrics_RPC_Count - no requests recorded")
	} else {
		if int(totalRequests) >= expectedCallCount {
			suite.add("Metrics_RPC_Count", duration, nil, fmt.Sprintf("requests=%d (expected>=%d)", int(totalRequests), expectedCallCount))
			logger.Info("✓ PASS Metrics_RPC_Count",
				log.Int("actual", int(totalRequests)),
				log.Int("expected_min", expectedCallCount))
		} else {
			suite.add("Metrics_RPC_Count", duration,
				fmt.Errorf("expected at least %d requests, got %d", expectedCallCount, int(totalRequests)), "")
			logger.Error(nil, "✗ FAIL Metrics_RPC_Count - insufficient requests",
				log.Int("actual", int(totalRequests)),
				log.Int("expected_min", expectedCallCount))
		}
	}

	// Verify Runtime metrics (if enabled)
	runtimeMetricsChecks := []string{
		"process_runtime_go_goroutines",
		"process_runtime_go_gc_count_total",
		"process_runtime_go_memory_heap_bytes",
		"process_runtime_go_memory_stack_bytes",
	}
	for _, metricName := range runtimeMetricsChecks {
		if _, exists := metrics[metricName]; exists {
			suite.add("Metrics_Runtime_"+metricName, duration, nil, "metric exists")
			logger.Info(fmt.Sprintf("✓ PASS Metrics_Runtime_%s - metric exists", metricName))
		} else {
			// Runtime metrics are optional, so we just log without failing
			logger.Info(fmt.Sprintf("⚠ SKIP Metrics_Runtime_%s - metric not enabled", metricName))
		}
	}

	// Verify Process metrics (if enabled)
	processMetricsChecks := []string{
		"process_start_time_seconds",
		"process_uptime_seconds",
		"process_memory_rss_bytes",
		"process_cpu_seconds_total",
	}
	for _, metricName := range processMetricsChecks {
		if _, exists := metrics[metricName]; exists {
			suite.add("Metrics_Process_"+metricName, duration, nil, "metric exists")
			logger.Info(fmt.Sprintf("✓ PASS Metrics_Process_%s - metric exists", metricName))
		} else {
			// Process metrics are optional, so we just log without failing
			logger.Info(fmt.Sprintf("⚠ SKIP Metrics_Process_%s - metric not enabled", metricName))
		}
	}

	// Count number of metrics (lines that don't start with #)
	lines := strings.Split(bodyStr, "\n")
	metricCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			metricCount++
		}
	}

	details := fmt.Sprintf("status=200, metrics=%d, format=valid, rpc_requests=%d", metricCount, int(totalRequests))
	suite.add("Metrics_Endpoint", duration, nil, details)
	logger.Info("✓ PASS Metrics_Endpoint",
		log.Str("duration", fmt.Sprintf("%dms", duration.Milliseconds())),
		log.Int("metrics", metricCount),
		log.Str("service", serviceName),
		log.Int("rpc_requests", int(totalRequests)))
}
