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
//	# Test user service with specific operation
//	./connect-tester http://localhost:8082 user-service create email@test.com "Test User"
//	./connect-tester http://localhost:8082 user-service get <user-id>
//	./connect-tester http://localhost:8082 user-service update <user-id> email@test.com "Updated Name"
//	./connect-tester http://localhost:8082 user-service delete <user-id>
//	./connect-tester http://localhost:8082 user-service list <page> <page-size>
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
	"log/slog"
	"os"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/clientx"
	"github.com/eggybyte-technology/egg/core/log"
	greetv1 "github.com/eggybyte-technology/egg/examples/minimal-connect-service/gen/go/greet/v1"
	greetv1connect "github.com/eggybyte-technology/egg/examples/minimal-connect-service/gen/go/greet/v1/greetv1connect"
	userv1 "github.com/eggybyte-technology/egg/examples/user-service/gen/go/user/v1"
	userv1connect "github.com/eggybyte-technology/egg/examples/user-service/gen/go/user/v1/userv1connect"
	"github.com/eggybyte-technology/egg/logx"
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
