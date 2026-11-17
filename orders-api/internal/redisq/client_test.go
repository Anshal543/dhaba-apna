package redisq

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"dhaba/orders-api/internal/order"
)

// fakeRedis is a minimal RESP server: it records every command and answers
// with whatever reply(cmd) returns (a raw RESP string).
type fakeRedis struct {
	ln    net.Listener
	mu    sync.Mutex
	cmds  [][]string
	reply func(cmd []string) string
}

func newFakeRedis(t *testing.T, reply func(cmd []string) string) *fakeRedis {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen on loopback: %v", err)
	}
	f := &fakeRedis{ln: ln, reply: reply}
	go f.serve()
	t.Cleanup(func() { ln.Close() })
	return f
}

func (f *fakeRedis) addr() string { return f.ln.Addr().String() }

func (f *fakeRedis) commands() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]string(nil), f.cmds...)
}

func (f *fakeRedis) serve() {
	for {
		conn, err := f.ln.Accept()
		if err != nil {
			return
		}
		go func() {
			defer conn.Close()
			r := bufio.NewReader(conn)
			for {
				cmd, err := readCommand(r)
				if err != nil {
					return
				}
				f.mu.Lock()
				f.cmds = append(f.cmds, cmd)
				f.mu.Unlock()
				io.WriteString(conn, f.reply(cmd))
			}
		}()
	}
}

func readCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	n, _ := strconv.Atoi(strings.TrimSpace(line[1:]))
	cmd := make([]string, 0, n)
	for i := 0; i < n; i++ {
		if _, err := r.ReadString('\n'); err != nil { // $len
			return nil, err
		}
		arg, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		cmd = append(cmd, strings.TrimRight(arg, "\r\n"))
	}
	return cmd, nil
}

func okReply(cmd []string) string {
	switch cmd[0] {
	case "PING":
		return "+PONG\r\n"
	case "AUTH":
		return "+OK\r\n"
	}
	return ":1\r\n"
}

func TestPing(t *testing.T) {
	f := newFakeRedis(t, okReply)
	if err := NewClient(f.addr(), "").Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPingConnectionRefused(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	if err := NewClient(addr, "").Ping(context.Background()); err == nil {
		t.Fatal("expected an error when nothing is listening")
	}
}

func TestEnqueueSendsJobJSON(t *testing.T) {
	f := newFakeRedis(t, okReply)
	p := NewPublisher(NewClient(f.addr(), ""))
	if err := p.Enqueue(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	got := f.commands()
	want := []string{"LPUSH", "orders:queue", `{"order_id":42}`}
	if len(got) != 1 || strings.Join(got[0], "|") != strings.Join(want, "|") {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestRecordOrderCommands(t *testing.T) {
	f := newFakeRedis(t, okReply)
	p := NewPublisher(NewClient(f.addr(), ""))
	at := time.Unix(1_800_000_090, 0) // minute = 30000001
	items := []order.Item{{Name: "Chicken Karahi", Qty: 2}, {Name: "Naan", Qty: 4}}
	if err := p.RecordOrder(context.Background(), items, at); err != nil {
		t.Fatal(err)
	}

	minuteKey := fmt.Sprintf("stats:orders:%d", at.Unix()/60)
	want := []string{
		"INCR|" + minuteKey,
		"EXPIRE|" + minuteKey + "|3600",
		"ZINCRBY|stats:dishes|2|Chicken Karahi",
		"ZINCRBY|stats:dishes|4|Naan",
	}
	got := f.commands()
	if len(got) != len(want) {
		t.Fatalf("got %d commands, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if strings.Join(got[i], "|") != want[i] {
			t.Errorf("command %d: got %v want %s", i, got[i], want[i])
		}
	}
}

func TestPasswordSendsAuthFirst(t *testing.T) {
	f := newFakeRedis(t, okReply)
	c := NewClient(f.addr(), "s3cret")
	if err := c.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := f.commands()
	if len(got) != 2 || got[0][0] != "AUTH" || got[0][1] != "s3cret" || got[1][0] != "PING" {
		t.Fatalf("expected AUTH then PING, got %v", got)
	}
}

func TestServerErrorReply(t *testing.T) {
	f := newFakeRedis(t, func([]string) string { return "-WRONGTYPE bad key\r\n" })
	_, err := NewClient(f.addr(), "").Do(context.Background(), "INCR", "x")
	var re Error
	if !errors.As(err, &re) || !strings.Contains(err.Error(), "WRONGTYPE") {
		t.Fatalf("expected a redis Error, got %v", err)
	}
}

func TestReplyTypes(t *testing.T) {
	replies := map[string]string{
		"STATUS": "+OK\r\n",
		"INT":    ":7\r\n",
		"BULK":   "$5\r\nhello\r\n",
		"NIL":    "$-1\r\n",
		"ARRAY":  "*2\r\n$1\r\na\r\n:2\r\n",
	}
	f := newFakeRedis(t, func(cmd []string) string { return replies[cmd[0]] })
	c := NewClient(f.addr(), "")
	ctx := context.Background()

	if v, _ := c.Do(ctx, "STATUS"); v != "OK" {
		t.Errorf("status: %v", v)
	}
	if v, _ := c.Do(ctx, "INT"); v != int64(7) {
		t.Errorf("int: %v", v)
	}
	if v, _ := c.Do(ctx, "BULK"); v != "hello" {
		t.Errorf("bulk: %v", v)
	}
	if v, err := c.Do(ctx, "NIL"); v != nil || err != nil {
		t.Errorf("nil: %v %v", v, err)
	}
	v, _ := c.Do(ctx, "ARRAY")
	arr, ok := v.([]any)
	if !ok || len(arr) != 2 || arr[0] != "a" || arr[1] != int64(2) {
		t.Errorf("array: %v", v)
	}
}
