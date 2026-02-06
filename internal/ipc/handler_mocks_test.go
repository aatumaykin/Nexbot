package ipc

import (
	"bytes"
	"net"
	"time"
)

// mockConn имитирует net.Conn для тестов
type mockConn struct {
	net.Conn
	readBuf  bytes.Buffer
	writeBuf bytes.Buffer
	closed   bool
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.readBuf.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.writeBuf.Write(b)
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) RemoteAddr() net.Addr {
	return &net.UnixAddr{Name: "/mock/test", Net: "unix"}
}

func (m *mockConn) LocalAddr() net.Addr {
	return &net.UnixAddr{Name: "/mock/local", Net: "unix"}
}

func (m *mockConn) SetDeadline(t time.Time) error     { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}
