// Package redisq is a tiny Redis client (RESP over TCP, standard library only)
// with the handful of commands orders-api needs: queue push and stats counters.
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
	"time"
)

const (
	QueueKey      = "orders:queue"
	DishesKey     = "stats:dishes"
	MinuteKeyFmt  = "stats:orders:%d"
	minuteKeyTTL  = 3600 // seconds
	defaultTimout = 3 * time.Second
)

// Error is an error reply sent by the Redis server.
type Error string

func (e Error) Error() string { return "redis: " + string(e) }

type Client struct {
	Addr     string
	Password string
	Timeout  time.Duration
}

func NewClient(addr, password string) *Client {
	return &Client{Addr: addr, Password: password, Timeout: defaultTimout}
}

// Do runs a single command.
func (c *Client) Do(ctx context.Context, args ...string) (any, error) {
	res, err := c.Pipeline(ctx, args)
	if err != nil {
		return nil, err
	}
	return res[0], nil
}

// Pipeline sends all commands on one connection and reads every reply.
// It returns the first server error reply, if any, after reading all replies.
func (c *Client) Pipeline(ctx context.Context, cmds ...[]string) ([]any, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = defaultTimout
	}
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", c.Addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	all := cmds
	if c.Password != "" {
		all = append([][]string{{"AUTH", c.Password}}, cmds...)
	}

	w := bufio.NewWriter(conn)
	for _, cmd := range all {
		writeCommand(w, cmd)
	}
	if err := w.Flush(); err != nil {
		return nil, err
	}

	r := bufio.NewReader(conn)
	replies := make([]any, 0, len(all))
	var firstErr error
	for range all {
		v, err := readReply(r)
		if err != nil {
			var re Error
			if errors.As(err, &re) {
				if firstErr == nil {
					firstErr = err
				}
				replies = append(replies, err)
				continue
			}
			return nil, err
		}
		replies = append(replies, v)
	}
	if c.Password != "" {
		replies = replies[1:]
	}
	return replies, firstErr
}

// Ping returns nil when the server answers PONG.
func (c *Client) Ping(ctx context.Context) error {
	v, err := c.Do(ctx, "PING")
	if err != nil {
		return err
	}
	if s, _ := v.(string); s != "PONG" {
		return fmt.Errorf("unexpected PING reply %v", v)
	}
	return nil
}

func writeCommand(w *bufio.Writer, args []string) {
	fmt.Fprintf(w, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(w, "$%d\r\n%s\r\n", len(a), a)
	}
}

func readReply(r *bufio.Reader) (any, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	if line == "" {
		return nil, errors.New("redis: empty reply")
	}
	body := line[1:]
	switch line[0] {
	case '+':
		return body, nil
	case '-':
		return nil, Error(body)
	case ':':
		return strconv.ParseInt(body, 10, 64)
	case '$':
		n, err := strconv.Atoi(body)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, nil
		}
		buf := make([]byte, n+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		return string(buf[:n]), nil
	case '*':
		n, err := strconv.Atoi(body)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, nil
		}
		out := make([]any, n)
		for i := range out {
			v, err := readReply(r)
			if err != nil {
				var re Error
				if errors.As(err, &re) {
					out[i] = err
					continue
				}
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	}
	return nil, fmt.Errorf("redis: unknown reply type %q", line[0])
}

func readLine(r *bufio.Reader) (string, error) {
	s, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(s, "\r\n"), nil
}
