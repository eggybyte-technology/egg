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

// TestUserInfoRoles tests role-related functionality.
func TestUserInfoRoles(t *testing.T) {
	tests := []struct {
		name  string
		user  *UserInfo
		roles []string
	}{
		{
			name: "single role",
			user: &UserInfo{
				UserID: "123",
				Roles:  []string{"admin"},
			},
			roles: []string{"admin"},
		},
		{
			name: "multiple roles",
			user: &UserInfo{
				UserID: "123",
				Roles:  []string{"admin", "user", "editor"},
			},
			roles: []string{"admin", "user", "editor"},
		},
		{
			name: "no roles",
			user: &UserInfo{
				UserID: "123",
				Roles:  []string{},
			},
			roles: []string{},
		},
		{
			name: "nil roles",
			user: &UserInfo{
				UserID: "123",
				Roles:  nil,
			},
			roles: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithUser(context.Background(), tt.user)
			retrieved, ok := UserFrom(ctx)
			if !ok {
				t.Fatal("Should find user in context")
			}

			if len(retrieved.Roles) != len(tt.roles) {
				t.Errorf("Expected %d roles, got %d", len(tt.roles), len(retrieved.Roles))
			}

			for i, role := range tt.roles {
				if retrieved.Roles[i] != role {
					t.Errorf("Expected role %q at index %d, got %q", role, i, retrieved.Roles[i])
				}
			}
		})
	}
}

// TestRequestMetaFields tests all RequestMeta fields.
func TestRequestMetaFields(t *testing.T) {
	tests := []struct {
		name string
		meta *RequestMeta
	}{
		{
			name: "all fields populated",
			meta: &RequestMeta{
				RequestID:     "req-123",
				InternalToken: "token-456",
				RemoteIP:      "192.168.1.1",
				UserAgent:     "Mozilla/5.0",
			},
		},
		{
			name: "minimal fields",
			meta: &RequestMeta{
				RequestID: "req-123",
			},
		},
		{
			name: "empty fields",
			meta: &RequestMeta{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithMeta(context.Background(), tt.meta)
			retrieved, ok := MetaFrom(ctx)
			if !ok {
				t.Fatal("Should find meta in context")
			}

			if retrieved.RequestID != tt.meta.RequestID {
				t.Errorf("RequestID mismatch: got %q, want %q", retrieved.RequestID, tt.meta.RequestID)
			}
			if retrieved.InternalToken != tt.meta.InternalToken {
				t.Errorf("InternalToken mismatch: got %q, want %q", retrieved.InternalToken, tt.meta.InternalToken)
			}
			if retrieved.RemoteIP != tt.meta.RemoteIP {
				t.Errorf("RemoteIP mismatch: got %q, want %q", retrieved.RemoteIP, tt.meta.RemoteIP)
			}
			if retrieved.UserAgent != tt.meta.UserAgent {
				t.Errorf("UserAgent mismatch: got %q, want %q", retrieved.UserAgent, tt.meta.UserAgent)
			}
		})
	}
}

// TestNilUserInfo tests handling of nil UserInfo.
func TestNilUserInfo(t *testing.T) {
	ctx := WithUser(context.Background(), nil)
	_, ok := UserFrom(ctx)
	if ok {
		t.Error("Should not find user when nil was stored")
	}
}

// TestNilRequestMeta tests handling of nil RequestMeta.
func TestNilRequestMeta(t *testing.T) {
	ctx := WithMeta(context.Background(), nil)
	_, ok := MetaFrom(ctx)
	if ok {
		t.Error("Should not find meta when nil was stored")
	}
}

// TestContextIsolation tests that contexts are properly isolated.
func TestContextIsolation(t *testing.T) {
	ctx1 := context.Background()
	ctx2 := context.Background()

	user1 := &UserInfo{UserID: "user1"}
	user2 := &UserInfo{UserID: "user2"}

	ctx1 = WithUser(ctx1, user1)
	ctx2 = WithUser(ctx2, user2)

	// Verify ctx1 has user1
	retrieved1, ok := UserFrom(ctx1)
	if !ok || retrieved1.UserID != "user1" {
		t.Error("Context 1 should have user1")
	}

	// Verify ctx2 has user2
	retrieved2, ok := UserFrom(ctx2)
	if !ok || retrieved2.UserID != "user2" {
		t.Error("Context 2 should have user2")
	}
}

// TestOverwriteUser tests that user can be overwritten.
func TestOverwriteUser(t *testing.T) {
	ctx := context.Background()
	user1 := &UserInfo{UserID: "user1"}
	user2 := &UserInfo{UserID: "user2"}

	ctx = WithUser(ctx, user1)
	ctx = WithUser(ctx, user2)

	retrieved, ok := UserFrom(ctx)
	if !ok {
		t.Fatal("Should find user in context")
	}

	if retrieved.UserID != "user2" {
		t.Errorf("Expected user2, got %q", retrieved.UserID)
	}
}

// TestOverwriteMeta tests that meta can be overwritten.
func TestOverwriteMeta(t *testing.T) {
	ctx := context.Background()
	meta1 := &RequestMeta{RequestID: "req1"}
	meta2 := &RequestMeta{RequestID: "req2"}

	ctx = WithMeta(ctx, meta1)
	ctx = WithMeta(ctx, meta2)

	retrieved, ok := MetaFrom(ctx)
	if !ok {
		t.Fatal("Should find meta in context")
	}

	if retrieved.RequestID != "req2" {
		t.Errorf("Expected req2, got %q", retrieved.RequestID)
	}
}
