//go:build windows

package main

import (
	"net"
	"os"
	"syscall"
	"time"
)

// dialPipe opens a Windows named pipe as a net.Conn using raw syscalls.
// This is how we talk to Discord on Windows without any external libraries.
func dialPipe(path string) (net.Conn, error) {
	// Convert path to UTF-16 for Windows API
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	// Open the named pipe
	handle, err := syscall.CreateFile(
		pathPtr,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0,    // no sharing
		nil,  // default security
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,    // no template
	)
	if err != nil {
		return nil, err
	}

	// Wrap the handle as an os.File so we can use it as net.Conn
	file := os.NewFile(uintptr(handle), path)
	return &pipeConn{file: file}, nil
}

// pipeConn wraps an os.File to satisfy the net.Conn interface
type pipeConn struct {
	file *os.File
}

func (p *pipeConn) Read(b []byte) (int, error)  { return p.file.Read(b) }
func (p *pipeConn) Write(b []byte) (int, error) { return p.file.Write(b) }
func (p *pipeConn) Close() error                { return p.file.Close() }

func (p *pipeConn) LocalAddr() net.Addr             { return pipeAddr{} }
func (p *pipeConn) RemoteAddr() net.Addr            { return pipeAddr{} }
func (p *pipeConn) SetDeadline(t time.Time) error      { return nil }
func (p *pipeConn) SetReadDeadline(t time.Time) error  { return nil }
func (p *pipeConn) SetWriteDeadline(t time.Time) error { return nil }

type pipeAddr struct{}

func (pipeAddr) Network() string { return "pipe" }
func (pipeAddr) String() string  { return "discord-ipc" }
