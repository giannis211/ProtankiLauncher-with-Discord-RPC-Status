package main

import (
	"net"
	"os"
	"syscall"
	"time"
)

func dialPipe(path string) (net.Conn, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	handle, err := syscall.CreateFile(
		pathPtr,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0, 
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,  
	)
	if err != nil {
		return nil, err
	}


	file := os.NewFile(uintptr(handle), path)
	return &pipeConn{file: file}, nil
}


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
