package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInvalidStateDoesNotAbortValidCallback(t *testing.T) {
	s := NewCallbackServer(0)
	if err := s.Start("expected"); err != nil {
		t.Fatal(err)
	}
	defer s.server.Close()

	bad := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(bad, httptest.NewRequest(http.MethodGet, "/callback?state=wrong&code=attacker", nil))
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("wrong-state status = %d, want 400", bad.Code)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	var code string
	if err := s.Wait(ctx, &code); err != context.DeadlineExceeded {
		t.Fatalf("wrong-state callback ended login: %v", err)
	}

	good := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(good, httptest.NewRequest(http.MethodGet, "/callback?state=expected&code=real", nil))
	if err := s.Wait(context.Background(), &code); err != nil || code != "real" {
		t.Fatalf("valid callback = (%q, %v), want real code", code, err)
	}
	if s.server.Addr != "127.0.0.1:0" {
		t.Fatalf("callback listener address = %q, want loopback", s.server.Addr)
	}
}
