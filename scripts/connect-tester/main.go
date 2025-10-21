// Package main provides a Connect service testing tool.
//
// Overview:
//   - Responsibility: Test Connect service endpoints for correctness
//   - Key Types: Main function with comprehensive Connect testing
//   - Concurrency Model: Sequential testing with timeout handling
//   - Error Semantics: Detailed error reporting with exit codes
//   - Performance Notes: Fast testing with configurable timeouts
//
// Usage:
//
//	go run main.go <service_url> <service_name>
//	./connect-tester http://localhost:8080 minimal-service
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	greetv1 "github.com/eggybyte-technology/egg/examples/minimal-connect-service/gen/go/greet/v1"
	greetv1connect "github.com/eggybyte-technology/egg/examples/minimal-connect-service/gen/go/greet/v1/greetv1connect"
	userv1 "github.com/eggybyte-technology/egg/examples/user-service/gen/go/user/v1"
	userv1connect "github.com/eggybyte-technology/egg/examples/user-service/gen/go/user/v1/userv1connect"
)

// TestResult represents the result of a Connect test.
type TestResult struct {
	Service   string    `json:"service"`
	Test      string    `json:"test"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
	Duration  string    `json:"duration"`
	Response  string    `json:"response,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// TestSuite represents a collection of test results.
type TestSuite struct {
	ServiceName string       `json:"service_name"`
	BaseURL     string       `json:"base_url"`
	Results     []TestResult `json:"results"`
	StartTime   time.Time    `json:"start_time"`
	EndTime     time.Time    `json:"end_time"`
	TotalTests  int          `json:"total_tests"`
	PassedTests int          `json:"passed_tests"`
	FailedTests int          `json:"failed_tests"`
}

// TestMinimalService tests the minimal Connect service endpoints.
func TestMinimalService(baseURL string) *TestSuite {
	suite := &TestSuite{
		ServiceName: "minimal-service",
		BaseURL:     baseURL,
		Results:     []TestResult{},
		StartTime:   time.Now(),
	}

	// Create Connect client
	client := greetv1connect.NewGreeterServiceClient(
		http.DefaultClient,
		baseURL,
		connect.WithGRPC(),
	)

	// Test SayHello endpoint
	testSayHello(suite, client)

	// Test SayHelloStream endpoint
	testSayHelloStream(suite, client)

	suite.EndTime = time.Now()
	suite.TotalTests = len(suite.Results)
	for _, result := range suite.Results {
		if result.Success {
			suite.PassedTests++
		} else {
			suite.FailedTests++
		}
	}

	return suite
}

// TestUserService tests the user Connect service endpoints.
func TestUserService(baseURL string) *TestSuite {
	suite := &TestSuite{
		ServiceName: "user-service",
		BaseURL:     baseURL,
		Results:     []TestResult{},
		StartTime:   time.Now(),
	}

	// Create Connect client
	client := userv1connect.NewUserServiceClient(
		http.DefaultClient,
		baseURL,
		connect.WithGRPC(),
	)

	// Test CreateUser endpoint and get the created user ID
	createdUserID := testCreateUser(suite, client)

	// Test GetUser endpoint with the created user ID
	testGetUser(suite, client, createdUserID)

	// Test ListUsers endpoint
	testListUsers(suite, client)

	suite.EndTime = time.Now()
	suite.TotalTests = len(suite.Results)
	for _, result := range suite.Results {
		if result.Success {
			suite.PassedTests++
		} else {
			suite.FailedTests++
		}
	}

	return suite
}

// testSayHello tests the SayHello endpoint.
func testSayHello(suite *TestSuite, client greetv1connect.GreeterServiceClient) {
	start := time.Now()
	result := TestResult{
		Service:   suite.ServiceName,
		Test:      "SayHello",
		Timestamp: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &greetv1.SayHelloRequest{
		Name:     "TestUser",
		Language: "en",
	}

	resp, err := client.SayHello(ctx, connect.NewRequest(req))
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
		if resp.Msg.Message != "" {
			result.Response = resp.Msg.Message
		}
	}

	result.Duration = time.Since(start).String()
	suite.Results = append(suite.Results, result)
}

// testSayHelloStream tests the SayHelloStream endpoint.
func testSayHelloStream(suite *TestSuite, client greetv1connect.GreeterServiceClient) {
	start := time.Now()
	result := TestResult{
		Service:   suite.ServiceName,
		Test:      "SayHelloStream",
		Timestamp: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req := &greetv1.SayHelloStreamRequest{
		Name:  "TestUser",
		Count: 3,
	}

	stream, err := client.SayHelloStream(ctx, connect.NewRequest(req))
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		var messages []string
		for stream.Receive() {
			msg := stream.Msg()
			messages = append(messages, msg.Message)
		}
		if err := stream.Err(); err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
			result.Response = fmt.Sprintf("Received %d messages: %s", len(messages), strings.Join(messages, ", "))
		}
	}

	result.Duration = time.Since(start).String()
	suite.Results = append(suite.Results, result)
}

// testCreateUser tests the CreateUser endpoint.
// Returns the created user ID for use in subsequent tests.
func testCreateUser(suite *TestSuite, client userv1connect.UserServiceClient) string {
	start := time.Now()
	result := TestResult{
		Service:   suite.ServiceName,
		Test:      "CreateUser",
		Timestamp: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Generate unique email for each test run to avoid conflicts
	timestamp := time.Now().UnixNano()
	req := &userv1.CreateUserRequest{
		Email: fmt.Sprintf("test-%d@example.com", timestamp),
		Name:  "Test User",
	}

	resp, err := client.CreateUser(ctx, connect.NewRequest(req))
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(start).String()
		suite.Results = append(suite.Results, result)
		return ""
	} else {
		result.Success = true
		if resp.Msg.User != nil {
			result.Response = fmt.Sprintf("Created user: %s (%s)", resp.Msg.User.Name, resp.Msg.User.Email)
		}
	}

	result.Duration = time.Since(start).String()
	suite.Results = append(suite.Results, result)

	// Return the created user ID
	if resp.Msg.User != nil {
		return resp.Msg.User.Id
	}
	return ""
}

// testGetUser tests the GetUser endpoint.
// Uses the provided user ID to test retrieval of an existing user.
func testGetUser(suite *TestSuite, client userv1connect.UserServiceClient, userID string) {
	start := time.Now()
	result := TestResult{
		Service:   suite.ServiceName,
		Test:      "GetUser",
		Timestamp: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Skip test if no user ID was provided (e.g., CreateUser failed)
	if userID == "" {
		result.Success = false
		result.Error = "No user ID available (CreateUser may have failed)"
		result.Duration = time.Since(start).String()
		suite.Results = append(suite.Results, result)
		return
	}

	req := &userv1.GetUserRequest{
		Id: userID,
	}

	resp, err := client.GetUser(ctx, connect.NewRequest(req))
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
		if resp.Msg.User != nil {
			result.Response = fmt.Sprintf("Retrieved user: %s (%s)", resp.Msg.User.Name, resp.Msg.User.Email)
		}
	}

	result.Duration = time.Since(start).String()
	suite.Results = append(suite.Results, result)
}

// testListUsers tests the ListUsers endpoint.
func testListUsers(suite *TestSuite, client userv1connect.UserServiceClient) {
	start := time.Now()
	result := TestResult{
		Service:   suite.ServiceName,
		Test:      "ListUsers",
		Timestamp: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &userv1.ListUsersRequest{
		Page:     1,
		PageSize: 10,
	}

	resp, err := client.ListUsers(ctx, connect.NewRequest(req))
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
		result.Response = fmt.Sprintf("Listed %d users (total: %d)", len(resp.Msg.Users), resp.Msg.Total)
	}

	result.Duration = time.Since(start).String()
	suite.Results = append(suite.Results, result)
}

// printResults prints test results in a formatted way.
func printResults(suite *TestSuite) {
	fmt.Printf("\n=== Connect Service Test Results ===\n")
	fmt.Printf("Service: %s\n", suite.ServiceName)
	fmt.Printf("Base URL: %s\n", suite.BaseURL)
	fmt.Printf("Total Tests: %d\n", suite.TotalTests)
	fmt.Printf("Passed: %d\n", suite.PassedTests)
	fmt.Printf("Failed: %d\n", suite.FailedTests)
	fmt.Printf("Duration: %s\n", suite.EndTime.Sub(suite.StartTime).String())
	fmt.Printf("\n--- Test Details ---\n")

	for _, result := range suite.Results {
		status := "✓ PASS"
		if !result.Success {
			status = "✗ FAIL"
		}
		fmt.Printf("%s %s (%s)\n", status, result.Test, result.Duration)
		if result.Error != "" {
			fmt.Printf("    Error: %s\n", result.Error)
		}
		if result.Response != "" {
			fmt.Printf("    Response: %s\n", result.Response)
		}
	}
	fmt.Printf("\n")
}

// printJSONResults prints test results in JSON format.
func printJSONResults(suite *TestSuite) {
	jsonData, err := json.MarshalIndent(suite, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling results to JSON: %v\n", err)
		return
	}
	fmt.Printf("%s\n", jsonData)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <service_url> <service_name>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s http://localhost:8080 minimal-service\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s http://localhost:8082 user-service\n", os.Args[0])
		os.Exit(1)
	}

	baseURL := os.Args[1]
	serviceName := os.Args[2]

	var suite *TestSuite

	switch serviceName {
	case "minimal-service":
		suite = TestMinimalService(baseURL)
	case "user-service":
		suite = TestUserService(baseURL)
	default:
		fmt.Fprintf(os.Stderr, "Unknown service: %s\n", serviceName)
		fmt.Fprintf(os.Stderr, "Supported services: minimal-service, user-service\n")
		os.Exit(1)
	}

	// Print results
	printResults(suite)

	// Also print JSON results for programmatic consumption
	printJSONResults(suite)

	// Exit with appropriate code
	if suite.FailedTests > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}
