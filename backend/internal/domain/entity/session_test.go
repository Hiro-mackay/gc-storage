package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestSession(createdAt, expiresAt time.Time) *Session {
	return &Session{
		ID:         "test-session-id",
		UserID:     uuid.New(),
		UserAgent:  "test-agent",
		IPAddress:  "127.0.0.1",
		ExpiresAt:  expiresAt,
		CreatedAt:  createdAt,
		LastUsedAt: time.Now(),
	}
}

func TestSession_IsExpired_WithinSlidingWindow_NotExpired(t *testing.T) {
	now := time.Now()
	session := newTestSession(now, now.Add(SessionTTL))

	if session.IsExpired() {
		t.Error("session within sliding window should not be expired")
	}
}

func TestSession_IsExpired_SlidingWindowExpired(t *testing.T) {
	now := time.Now()
	session := newTestSession(now, now.Add(-1*time.Hour))

	if !session.IsExpired() {
		t.Error("session past ExpiresAt should be expired")
	}
}

func TestSession_IsExpired_AbsoluteLifetimeExceeded(t *testing.T) {
	// Created 31 days ago, but ExpiresAt is in the future (sliding window refreshed)
	createdAt := time.Now().Add(-31 * 24 * time.Hour)
	expiresAt := time.Now().Add(SessionTTL)
	session := newTestSession(createdAt, expiresAt)

	if !session.IsExpired() {
		t.Error("session exceeding MaxSessionLifetime should be expired even if ExpiresAt is in the future")
	}
}

func TestSession_IsExpired_WithinAbsoluteLifetime(t *testing.T) {
	// Created 29 days ago, ExpiresAt in the future
	createdAt := time.Now().Add(-29 * 24 * time.Hour)
	expiresAt := time.Now().Add(SessionTTL)
	session := newTestSession(createdAt, expiresAt)

	if session.IsExpired() {
		t.Error("session within MaxSessionLifetime should not be expired")
	}
}

func TestSession_IsExpired_ExactlyAtAbsoluteLifetime(t *testing.T) {
	// Created exactly 30 days ago — boundary case
	createdAt := time.Now().Add(-MaxSessionLifetime)
	expiresAt := time.Now().Add(SessionTTL)
	session := newTestSession(createdAt, expiresAt)

	// time.Now() in IsExpired() is slightly after createdAt.Add(MaxSessionLifetime)
	// so this should be expired
	if !session.IsExpired() {
		t.Error("session at exact MaxSessionLifetime boundary should be expired")
	}
}

func TestSession_IsValid_ReturnsInverseOfIsExpired(t *testing.T) {
	now := time.Now()

	validSession := newTestSession(now, now.Add(SessionTTL))
	if !validSession.IsValid() {
		t.Error("valid session should return true for IsValid")
	}

	expiredSession := newTestSession(now, now.Add(-1*time.Hour))
	if expiredSession.IsValid() {
		t.Error("expired session should return false for IsValid")
	}
}

func TestSession_Refresh_ExtendsExpiresAt(t *testing.T) {
	now := time.Now()
	session := newTestSession(now, now.Add(1*time.Hour))

	oldExpiresAt := session.ExpiresAt
	session.Refresh()

	if !session.ExpiresAt.After(oldExpiresAt) {
		t.Error("Refresh should extend ExpiresAt")
	}

	// ExpiresAt should be approximately now + SessionTTL
	expected := time.Now().Add(SessionTTL)
	diff := session.ExpiresAt.Sub(expected)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("ExpiresAt should be ~now+SessionTTL, got diff=%v", diff)
	}

	// LastUsedAt should be updated
	lastUsedDiff := time.Since(session.LastUsedAt)
	if lastUsedDiff > time.Second {
		t.Error("Refresh should update LastUsedAt to approximately now")
	}
}
