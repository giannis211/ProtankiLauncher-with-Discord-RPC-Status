package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const CLIENT_ID = "1515462594272559145"
const GAME_EXE = "ProTanki.exe"


const (
	opHandshake = 0
	opFrame     = 1
	opClose     = 2
)

type rpcConn struct{ conn net.Conn }

func readUsername(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "User.txt"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "User=") {
			name := strings.TrimPrefix(line, "User=")
			name = strings.Trim(name, `"' `) 
			return name
		}
	}
	return ""
}

func connectToDiscord() (*rpcConn, error) {
	for i := 0; i < 10; i++ {
		pipe := fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i)
		conn, err := dialPipe(pipe)
		if err == nil {
			return &rpcConn{conn: conn}, nil
		}
	}
	return nil, fmt.Errorf("no discord pipe found")
}

func (r *rpcConn) send(op uint32, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], op)
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(data)))
	r.conn.Write(header)
	_, err = r.conn.Write(data)
	return err
}

func (r *rpcConn) read() error {
	header := make([]byte, 8)
	if _, err := r.conn.Read(header); err != nil {
		return err
	}
	length := binary.LittleEndian.Uint32(header[4:8])
	body := make([]byte, length)
	r.conn.Read(body)
	return nil
}

func (r *rpcConn) close() {
	r.send(opClose, map[string]any{"v": 1, "client_id": CLIENT_ID})
	r.conn.Close()
}

func (r *rpcConn) setActivity(startTime time.Time, username string) error {
	nonce := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())

	activity := map[string]any{
		"large_image": "protanki",
		"large_text":  "Protanki Online",
		"instance":    false,
		"timestamps":  map[string]any{"start": startTime.Unix()},
		"buttons": []map[string]any{
			{"label": "Discord Server", "url": "https://discord.com/invite/protanki"},
		},
	}

	if username != "" {
		activity["state"] = username
	}

	payload := map[string]any{
		"cmd":   "SET_ACTIVITY",
		"nonce": nonce,
		"args": map[string]any{
			"pid":      os.Getpid(),
			"activity": activity,
		},
	}
	return r.send(opFrame, payload)
}

func main() {
	exePath, err := os.Executable()
	if err != nil {
		os.Exit(1)
	}
	dir := filepath.Dir(exePath)

	username := readUsername(dir)

	gamePath := filepath.Join(dir, GAME_EXE)
	gameCmd := exec.Command(gamePath)
	gameCmd.Dir = dir
	if err := gameCmd.Start(); err != nil {
		os.Exit(1)
	}

	startTime := time.Now()

	var rpc *rpcConn
	for attempt := 1; attempt <= 12; attempt++ {
		c, err := connectToDiscord()
		if err == nil {
			rpc = c
			break
		}
		time.Sleep(5 * time.Second)
	}

	if rpc != nil {
		rpc.send(opHandshake, map[string]any{"v": 1, "client_id": CLIENT_ID})
		rpc.read()
		rpc.setActivity(startTime, username)
		rpc.read()

		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := rpc.setActivity(startTime, username); err != nil {
					return
				}
				rpc.read()
			}
		}()
	}

	gameDone := make(chan struct{})
	go func() {
		gameCmd.Wait()
		close(gameDone)
	}()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-gameDone:
	case <-osSignal:
		gameCmd.Process.Kill()
	}

	if rpc != nil {
		rpc.close()
	}
}
