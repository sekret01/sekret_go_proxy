package ipstep

import (
	"net"
	"testing"
	"time"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/handshake"
)

func runHandshake(
	t *testing.T,
	step core.HandshakeStep,
	serverCtx, clientCtx *core.HandshakeContext,
) (clientErr, serverErr error) {
	t.Helper()

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	done := make(chan struct{}, 2)

	go func() {
		clientErr = step.ClientSide(clientCtx, clientConn)
		done <- struct{}{}
	}()
	go func() {
		serverErr = step.ServerSide(serverCtx, serverConn)
		done <- struct{}{}
	}()

	timeout := time.After(2 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Fatalf("handshake timed out")
		}
	}
	return
}

func TestIpStep_Name(t *testing.T) {
	step := &IpStep{}
	if got := step.Name(); got != "ip" {
		t.Fatalf("Name() = %q, want %q", got, "ip")
	}
}

func TestNewIpStep(t *testing.T) {
	step, err := NewIpStep(&handshake.HandshakeStepConfig{})
	if err != nil {
		t.Fatalf("NewIpStep returned error: %v", err)
	}
	if step == nil {
		t.Fatal("NewIpStep returned nil")
	}
}

func TestIpStep_HappyPath(t *testing.T) {
	step := &IpStep{}
	serverCtx := &core.HandshakeContext{ClientIP: "10.0.0.2"}
	clientCtx := &core.HandshakeContext{}

	clientErr, serverErr := runHandshake(t, step, serverCtx, clientCtx)

	if clientErr != nil {
		t.Fatalf("client error: %v", clientErr)
	}
	if serverErr != nil {
		t.Fatalf("server error: %v", serverErr)
	}
	if clientCtx.ClientIP != "10.0.0.2" {
		t.Fatalf("client ClientIP = %q, want %q", clientCtx.ClientIP, "10.0.0.2")
	}
	if serverCtx.ClientIP != "10.0.0.2" {
		t.Fatalf("server ClientIP changed to %q, want %q", serverCtx.ClientIP, "10.0.0.2")
	}
}

func TestIpStep_Server_InvalidHello(t *testing.T) {
	step := &IpStep{}

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	go func() {
		// клиент шлёт неверный hello
		clientConn.Write([]byte("hello"))
		// читаем ответ (его не будет), но клиентская сторона не нужна
		clientConn.Close()
	}()

	ctx := &core.HandshakeContext{ClientIP: "10.0.0.2"}
	err := step.ServerSide(ctx, serverConn)
	if err == nil {
		t.Fatal("expected error for invalid hello, got nil")
	}
}

func TestIpStep_Client_ConnectionClosed(t *testing.T) {
	step := &IpStep{}

	clientConn, serverConn := net.Pipe()
	serverConn.Close() // сразу закрываем — клиент не сможет ни писать, ни читать
	defer clientConn.Close()

	ctx := &core.HandshakeContext{}
	err := step.ClientSide(ctx, clientConn)
	if err == nil {
		t.Fatal("expected error on closed connection, got nil")
	}
}

func TestIpStep_Server_ConnectionClosed(t *testing.T) {
	step := &IpStep{}

	clientConn, serverConn := net.Pipe()
	defer serverConn.Close() // сразу закрываем — клиент не сможет ни писать, ни читать
	clientConn.Close()

	ctx := &core.HandshakeContext{}
	err := step.ServerSide(ctx, clientConn)
	if err == nil {
		t.Fatal("expected error on closed connection, got nil")
	}
}
