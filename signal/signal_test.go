package signal

import (
	"testing"
	"time"
)

// TestNewInterceptorAllowsMultipleInstances verifies that embedded lnd
// runtimes can own independent shutdown interceptors in the same process.
func TestNewInterceptorAllowsMultipleInstances(t *testing.T) {
	t.Parallel()

	first := NewInterceptor()
	second := NewInterceptor()

	if !first.Listening() {
		t.Fatalf("first local interceptor should be listening")
	}
	if !second.Listening() {
		t.Fatalf("second local interceptor should be listening")
	}

	first.RequestShutdown()
	waitShutdown(t, first)

	if first.Listening() {
		t.Fatalf("first local interceptor should be stopped")
	}
	if !second.Listening() {
		t.Fatalf("second local interceptor should still be listening")
	}

	global, err := Intercept()
	if err != nil {
		t.Fatalf("global interceptor should coexist: %v", err)
	}
	global.RequestShutdown()
	waitShutdown(t, global)

	second.RequestShutdown()
	waitShutdown(t, second)
}

func waitShutdown(t *testing.T, interceptor Interceptor) {
	t.Helper()

	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	select {
	case <-interceptor.ShutdownChannel():

	case <-timer.C:
		t.Fatalf("timed out waiting for interceptor shutdown")
	}
}
