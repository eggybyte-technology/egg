// Package internal provides tests for connectx internal interceptors.
package internal

import (
	"database/sql"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"go.eggybyte.com/egg/core/errors"
	"gorm.io/gorm"
)

func TestMapErrorToConnectCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode connect.Code
	}{
		{"InvalidArgument", errors.New(errors.CodeInvalidArgument, "test"), connect.CodeInvalidArgument},
		{"NotFound", errors.New(errors.CodeNotFound, "test"), connect.CodeNotFound},
		{"AlreadyExists", errors.New(errors.CodeAlreadyExists, "test"), connect.CodeAlreadyExists},
		{"PermissionDenied", errors.New(errors.CodePermissionDenied, "test"), connect.CodePermissionDenied},
		{"Unauthenticated", errors.New(errors.CodeUnauthenticated, "test"), connect.CodeUnauthenticated},
		{"ResourceExhausted", errors.New(errors.CodeResourceExhausted, "test"), connect.CodeResourceExhausted},
		{"Internal", errors.New(errors.CodeInternal, "test"), connect.CodeInternal},
		{"Unavailable", errors.New(errors.CodeUnavailable, "test"), connect.CodeUnavailable},
		{"DeadlineExceeded", errors.New(errors.CodeDeadlineExceeded, "test"), connect.CodeDeadlineExceeded},
		{"Unimplemented", errors.New(errors.CodeUnimplemented, "test"), connect.CodeUnimplemented},
		{"Aborted", errors.New(errors.CodeAborted, "test"), connect.CodeAborted},
		{"OutOfRange", errors.New(errors.CodeOutOfRange, "test"), connect.CodeOutOfRange},
		{"DataLoss", errors.New(errors.CodeDataLoss, "test"), connect.CodeDataLoss},
		{"Unknown", errors.New("UNKNOWN_CODE", "test"), connect.CodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := mapErrorToConnectCode(tt.err)
			if code != tt.wantCode {
				t.Errorf("mapErrorToConnectCode() = %v, want %v", code, tt.wantCode)
			}
		})
	}
}

func TestIsServerError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantResult bool
	}{
		{"nil error", nil, false},
		{"sql.ErrConnDone", sql.ErrConnDone, true},
		{"sql.ErrTxDone", sql.ErrTxDone, true},
		{"connection refused", errors.New(errors.CodeInternal, "connection refused"), true},
		{"connection reset", errors.New(errors.CodeInternal, "connection reset"), true},
		{"timeout", errors.New(errors.CodeInternal, "timeout"), true},
		{"network unreachable", errors.New(errors.CodeInternal, "network is unreachable"), true},
		{"gorm.ErrRecordNotFound", gorm.ErrRecordNotFound, false},
		{"NotFound code", errors.New(errors.CodeNotFound, "not found"), false},
		{"AlreadyExists code", errors.New(errors.CodeAlreadyExists, "exists"), false},
		{"InvalidArgument code", errors.New(errors.CodeInvalidArgument, "invalid"), false},
		{"Internal code", errors.New(errors.CodeInternal, "internal"), true},
		{"Unavailable code", errors.New(errors.CodeUnavailable, "unavailable"), true},
		{"Unknown code", errors.New("UNKNOWN", "test"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isServerError(tt.err)
			if result != tt.wantResult {
				t.Errorf("isServerError() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestExtractIdentityFromConnectHeaders(t *testing.T) {
	mapping := HeaderMapping{
		UserID:    "X-User-Id",
		UserName:  "X-User-Name",
		Roles:     "X-User-Roles",
		RequestID: "X-Request-Id",
		RealIP:    "X-Real-IP",
		UserAgent: "User-Agent",
	}

	headers := http.Header{
		"X-User-Id":    []string{"u-123"},
		"X-User-Name":  []string{"testuser"},
		"X-User-Roles": []string{"admin, user"},
		"X-Request-Id": []string{"req-123"},
		"X-Real-IP":    []string{"127.0.0.1"},
		"User-Agent":   []string{"test-agent"},
	}

	user, meta := extractIdentityFromConnectHeaders(headers, mapping)

	if user == nil {
		t.Fatal("User should be extracted")
	}
	if user.UserID != "u-123" {
		t.Errorf("UserID = %q, want %q", user.UserID, "u-123")
	}
	if user.UserName != "testuser" {
		t.Errorf("UserName = %q, want %q", user.UserName, "testuser")
	}
	if len(user.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(user.Roles))
	}

	if meta == nil {
		t.Fatal("Meta should be extracted")
	}
	if meta.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want %q", meta.RequestID, "req-123")
	}
	if meta.UserAgent != "test-agent" {
		t.Errorf("UserAgent = %q, want %q", meta.UserAgent, "test-agent")
	}
	// Note: RemoteIP extraction is tested separately in TestExtractIdentityFromConnectHeaders_ForwardedFor
}

func TestExtractIdentityFromConnectHeaders_EmptyHeaders(t *testing.T) {
	mapping := HeaderMapping{
		UserID: "X-User-Id",
	}

	headers := http.Header{}

	user, meta := extractIdentityFromConnectHeaders(headers, mapping)

	if user != nil {
		t.Error("User should be nil when headers are empty")
	}
	if meta == nil {
		t.Fatal("Meta should always be created")
	}
}

func TestExtractIdentityFromConnectHeaders_RolesTrimmed(t *testing.T) {
	mapping := HeaderMapping{
		UserID: "X-User-Id",
		Roles:  "X-User-Roles",
	}

	headers := http.Header{
		"X-User-Id":    []string{"u-123"},
		"X-User-Roles": []string{" admin , user "},
	}

	user, _ := extractIdentityFromConnectHeaders(headers, mapping)

	if user == nil {
		t.Fatal("User should be extracted")
	}
	if len(user.Roles) != 2 {
		t.Fatalf("Expected 2 roles, got %d", len(user.Roles))
	}
	if user.Roles[0] != "admin" {
		t.Errorf("Role[0] = %q, want %q", user.Roles[0], "admin")
	}
	if user.Roles[1] != "user" {
		t.Errorf("Role[1] = %q, want %q", user.Roles[1], "user")
	}
}

func TestExtractIdentityFromConnectHeaders_ForwardedFor(t *testing.T) {
	mapping := HeaderMapping{
		ForwardedFor: "X-Forwarded-For",
	}

	headers := http.Header{
		"X-Forwarded-For": []string{"192.168.1.1, 10.0.0.1"},
	}

	_, meta := extractIdentityFromConnectHeaders(headers, mapping)

	if meta == nil {
		t.Fatal("Meta should be extracted")
	}
	if meta.RemoteIP != "192.168.1.1" {
		t.Errorf("RemoteIP = %q, want %q (first IP from X-Forwarded-For)", meta.RemoteIP, "192.168.1.1")
	}
}

func TestExtractIdentityFromConnectHeaders_NoUserID(t *testing.T) {
	mapping := HeaderMapping{
		UserID: "X-User-Id",
	}

	headers := http.Header{
		"X-Request-Id": []string{"req-123"},
	}

	user, meta := extractIdentityFromConnectHeaders(headers, mapping)

	if user != nil {
		t.Error("User should be nil when UserID header is missing")
	}
	if meta == nil {
		t.Fatal("Meta should always be created")
	}
}
