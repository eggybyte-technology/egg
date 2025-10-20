package identity

import (
	"context"
	"testing"
)

func TestWithUser(t *testing.T) {
	ctx := context.Background()
	user := &UserInfo{
		UserID:   "123",
		UserName: "testuser",
		Tenant:   "test-tenant",
		Roles:    []string{"admin", "user"},
	}

	ctxWithUser := WithUser(ctx, user)

	// Verify user was stored
	retrievedUser, ok := UserFrom(ctxWithUser)
	if !ok {
		t.Fatal("User should be found in context")
	}

	if retrievedUser.UserID != user.UserID {
		t.Errorf("Expected UserID %q, got %q", user.UserID, retrievedUser.UserID)
	}

	if retrievedUser.UserName != user.UserName {
		t.Errorf("Expected UserName %q, got %q", user.UserName, retrievedUser.UserName)
	}

	if retrievedUser.Tenant != user.Tenant {
		t.Errorf("Expected Tenant %q, got %q", user.Tenant, retrievedUser.Tenant)
	}

	if len(retrievedUser.Roles) != len(user.Roles) {
		t.Errorf("Expected %d roles, got %d", len(user.Roles), len(retrievedUser.Roles))
	}

	for i, role := range user.Roles {
		if retrievedUser.Roles[i] != role {
			t.Errorf("Expected role %q at index %d, got %q", role, i, retrievedUser.Roles[i])
		}
	}
}

func TestUserFrom(t *testing.T) {
	ctx := context.Background()

	// Test with no user
	_, ok := UserFrom(ctx)
	if ok {
		t.Error("Should not find user in empty context")
	}

	// Test with user
	user := &UserInfo{UserID: "123"}
	ctxWithUser := WithUser(ctx, user)

	retrievedUser, ok := UserFrom(ctxWithUser)
	if !ok {
		t.Fatal("Should find user in context")
	}

	if retrievedUser.UserID != user.UserID {
		t.Errorf("Expected UserID %q, got %q", user.UserID, retrievedUser.UserID)
	}
}

func TestWithMeta(t *testing.T) {
	ctx := context.Background()
	meta := &RequestMeta{
		RequestID:     "req-123",
		InternalToken: "token-456",
		RemoteIP:      "192.168.1.1",
		UserAgent:     "test-agent",
	}

	ctxWithMeta := WithMeta(ctx, meta)

	// Verify meta was stored
	retrievedMeta, ok := MetaFrom(ctxWithMeta)
	if !ok {
		t.Fatal("Meta should be found in context")
	}

	if retrievedMeta.RequestID != meta.RequestID {
		t.Errorf("Expected RequestID %q, got %q", meta.RequestID, retrievedMeta.RequestID)
	}

	if retrievedMeta.InternalToken != meta.InternalToken {
		t.Errorf("Expected InternalToken %q, got %q", meta.InternalToken, retrievedMeta.InternalToken)
	}

	if retrievedMeta.RemoteIP != meta.RemoteIP {
		t.Errorf("Expected RemoteIP %q, got %q", meta.RemoteIP, retrievedMeta.RemoteIP)
	}

	if retrievedMeta.UserAgent != meta.UserAgent {
		t.Errorf("Expected UserAgent %q, got %q", meta.UserAgent, retrievedMeta.UserAgent)
	}
}

func TestMetaFrom(t *testing.T) {
	ctx := context.Background()

	// Test with no meta
	_, ok := MetaFrom(ctx)
	if ok {
		t.Error("Should not find meta in empty context")
	}

	// Test with meta
	meta := &RequestMeta{RequestID: "req-123"}
	ctxWithMeta := WithMeta(ctx, meta)

	retrievedMeta, ok := MetaFrom(ctxWithMeta)
	if !ok {
		t.Fatal("Should find meta in context")
	}

	if retrievedMeta.RequestID != meta.RequestID {
		t.Errorf("Expected RequestID %q, got %q", meta.RequestID, retrievedMeta.RequestID)
	}
}

func TestContextChaining(t *testing.T) {
	ctx := context.Background()

	user := &UserInfo{UserID: "123"}
	meta := &RequestMeta{RequestID: "req-123"}

	// Chain context operations
	ctx = WithUser(ctx, user)
	ctx = WithMeta(ctx, meta)

	// Verify both are present
	retrievedUser, ok := UserFrom(ctx)
	if !ok {
		t.Fatal("Should find user after chaining")
	}

	retrievedMeta, ok := MetaFrom(ctx)
	if !ok {
		t.Fatal("Should find meta after chaining")
	}

	if retrievedUser.UserID != user.UserID {
		t.Errorf("Expected UserID %q, got %q", user.UserID, retrievedUser.UserID)
	}

	if retrievedMeta.RequestID != meta.RequestID {
		t.Errorf("Expected RequestID %q, got %q", meta.RequestID, retrievedMeta.RequestID)
	}
}
