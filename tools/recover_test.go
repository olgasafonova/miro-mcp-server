package tools

import (
	"strings"
	"testing"
)

// TestRecoverPanic_AssignsErrorWithCorrelationID is a regression test for HG-1.
// The dispatcher's deferred recover MUST reassign the named `err` return so a
// panicking handler surfaces as a structured error to the MCP caller, not as
// `(nil, zero, nil)` which the SDK would treat as a successful empty response.
func TestRecoverPanic_AssignsErrorWithCorrelationID(t *testing.T) {
	registry := newTestRegistry(&MockClient{})

	err := func() (err error) {
		defer registry.recoverPanic("test_tool", &err)
		panic("secret panic value")
	}()

	if err == nil {
		t.Fatal("expected non-nil error after panic, got nil — HG-1 silent fake-success regression")
	}
	msg := err.Error()
	if !strings.Contains(msg, "test_tool: internal error") {
		t.Errorf("error message missing tool name and 'internal error': %q", msg)
	}
	if !strings.Contains(msg, "correlation_id=") {
		t.Errorf("error message missing correlation_id: %q", msg)
	}
	if strings.Contains(msg, "secret panic value") {
		t.Errorf("error message leaked panic value: %q", msg)
	}
}

// TestRecoverPanic_NoPanicNoError verifies the success path is unchanged.
func TestRecoverPanic_NoPanicNoError(t *testing.T) {
	registry := newTestRegistry(&MockClient{})

	err := func() (err error) {
		defer registry.recoverPanic("test_tool", &err)
		return nil
	}()

	if err != nil {
		t.Errorf("expected no error when no panic, got: %v", err)
	}
}

// TestRecoverPanic_PreservesExistingError verifies that recoverPanic on a
// non-panicking path doesn't clobber an existing returned error.
func TestRecoverPanic_PreservesExistingError(t *testing.T) {
	registry := newTestRegistry(&MockClient{})

	sentinel := errStub("real error")
	err := func() (err error) {
		defer registry.recoverPanic("test_tool", &err)
		return sentinel
	}()

	if err != sentinel {
		t.Errorf("recoverPanic clobbered existing error: got %v, want %v", err, sentinel)
	}
}

// TestNewCorrelationID_DistinctAndShaped verifies the correlation ID generator.
func TestNewCorrelationID_DistinctAndShaped(t *testing.T) {
	a := newCorrelationID()
	b := newCorrelationID()
	if a == b {
		t.Errorf("expected distinct correlation IDs, got same: %q", a)
	}
	if len(a) != 16 {
		t.Errorf("expected 16-char hex ID, got len %d: %q", len(a), a)
	}
}

type errStub string

func (e errStub) Error() string { return string(e) }
