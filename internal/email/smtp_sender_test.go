package email_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/hfleury/horsemarketplacebk/internal/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startSilentSMTPServer accepts connections and never writes, simulating a
// hanging SMTP server. It returns the port it listens on.
func startSilentSMTPServer(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	var mu sync.Mutex
	var accepted []net.Conn
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			accepted = append(accepted, conn)
			mu.Unlock()
		}
	}()

	t.Cleanup(func() {
		_ = listener.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range accepted {
			_ = conn.Close()
		}
	})
	return listener.Addr().(*net.TCPAddr).Port
}

func TestSMTPSender_Send_ReturnsErrorWhenServerHangs(t *testing.T) {
	port := startSilentSMTPServer(t)
	sender := email.NewSMTPSender("127.0.0.1", port, "user", "pass", "from@example.com", 200*time.Millisecond)

	start := time.Now()
	err := sender.Send(context.Background(), "to@example.com", "subject", "body")

	assert.Error(t, err)
	assert.Less(t, time.Since(start), 2*time.Second)
}

func TestSMTPSender_Send_ReturnsErrorWhenContextAlreadyCancelled(t *testing.T) {
	port := startSilentSMTPServer(t)
	sender := email.NewSMTPSender("127.0.0.1", port, "user", "pass", "from@example.com", 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := sender.Send(ctx, "to@example.com", "subject", "body")

	assert.Error(t, err)
	assert.Less(t, time.Since(start), time.Second)
}

func TestNewSMTPSender_DefaultsNonPositiveTimeout(t *testing.T) {
	sender := email.NewSMTPSender("127.0.0.1", 25, "user", "pass", "from@example.com", 0)

	assert.Equal(t, 10*time.Second, sender.Timeout)
}
