package enckeystep

import (
	"net"
	"testing"
	"time"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
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

func TestEncKeyStep_Name(t *testing.T) {
	step := &EncKeyStep{}
	if got := step.Name(); got != "enckey" {
		t.Fatalf("Name() = %q, want %q", got, "enckey")
	}
}

func TestNewEncKeyStep(t *testing.T) {
	step, err := NewEncKeyStep(&config.Config{})
	if err != nil {
		t.Fatalf("NewEncKeyStep returned error: %v", err)
	}
	if step == nil {
		t.Fatal("NewEncKeyStep returned nil")
	}
}

func TestEncKeyStep_HappyPath(t *testing.T) {
	step := &EncKeyStep{}
	key := []byte("super-secret-key")
	serverCtx := &core.HandshakeContext{EncryptorKey: key}
	clientCtx := &core.HandshakeContext{}

	clientErr, serverErr := runHandshake(t, step, serverCtx, clientCtx)

	if clientErr != nil {
		t.Fatalf("client error: %v", clientErr)
	}
	if serverErr != nil {
		t.Fatalf("server error: %v", serverErr)
	}
	if string(clientCtx.EncryptorKey) != string(key) {
		t.Fatalf("client EncryptorKey = %q, want %q", clientCtx.EncryptorKey, key)
	}
	if string(serverCtx.EncryptorKey) != string(key) {
		t.Fatalf("server EncryptorKey changed to %q, want %q", serverCtx.EncryptorKey, key)
	}
}

func TestEncKeyStep_HappyPath_EmptyKey(t *testing.T) {
	step := &EncKeyStep{}
	serverCtx := &core.HandshakeContext{EncryptorKey: []byte{}}
	clientCtx := &core.HandshakeContext{}

	clientErr, serverErr := runHandshake(t, step, serverCtx, clientCtx)

	if clientErr != nil {
		t.Fatalf("client error: %v", clientErr)
	}
	if serverErr != nil {
		t.Fatalf("server error: %v", serverErr)
	}
	if len(clientCtx.EncryptorKey) != 0 {
		t.Fatalf("client EncryptorKey = %q, want empty", clientCtx.EncryptorKey)
	}
}

func TestEncKeyStep_Server_InvalidHello(t *testing.T) {
	step := &EncKeyStep{}

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	go func() {
		// клиент шлёт неверный hello
		clientConn.Write([]byte("bad-hello"))
		clientConn.Close()
	}()

	ctx := &core.HandshakeContext{EncryptorKey: []byte("key")}
	err := step.ServerSide(ctx, serverConn)
	if err == nil {
		t.Fatal("expected error for invalid hello, got nil")
	}
}

func TestEncKeyStep_Client_InvalidResponse(t *testing.T) {
	step := &EncKeyStep{}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	go func() {
		// читаем hello от клиента
		buf := make([]byte, 100)
		serverConn.Read(buf)
		// отвечаем неверно
		serverConn.Write([]byte("wrong-response:key"))
		serverConn.Close()
	}()

	ctx := &core.HandshakeContext{}
	err := step.ClientSide(ctx, clientConn)
	if err == nil {
		t.Fatal("expected error for invalid response, got nil")
	}
}

func TestEncKeyStep_Client_InvalidResponsePrefix(t *testing.T) {
	step := &EncKeyStep{}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	go func() {
		buf := make([]byte, 100)
		serverConn.Read(buf)
		// префикс верный, но без двоеточия — tokens < 2
		serverConn.Write([]byte("vpn-enckey-r"))
		serverConn.Close()
	}()

	ctx := &core.HandshakeContext{}
	err := step.ClientSide(ctx, clientConn)
	if err == nil {
		t.Fatal("expected error for response without separator, got nil")
	}
}

func TestEncKeyStep_Client_ConnectionClosed(t *testing.T) {
	step := &EncKeyStep{}

	clientConn, serverConn := net.Pipe()
	serverConn.Close() // сразу закрываем — клиент не сможет ни писать, ни читать
	defer clientConn.Close()

	ctx := &core.HandshakeContext{}
	err := step.ClientSide(ctx, clientConn)
	if err == nil {
		t.Fatal("expected error on closed connection, got nil")
	}
}

func TestEncKeyStep_Server_ConnectionClosed(t *testing.T) {
	step := &EncKeyStep{}

	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	clientConn.Close()

	ctx := &core.HandshakeContext{}
	err := step.ServerSide(ctx, clientConn)
	if err == nil {
		t.Fatal("expected error on closed connection, got nil")
	}
}
