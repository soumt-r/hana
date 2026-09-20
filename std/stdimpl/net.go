package stdimpl

import (
	"bufio"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/soumt-r/hana/errs"
)

// NetAccess is asked before every network operation, like FileAccess for files.
// op names it ("connect", "listen", "http"); target is the address or URL.
// `hana run --allow-net=false` swaps in DenyNet.
var NetAccess = func(op, target string) error { return nil }

// DenyNet is the policy that turns network access off.
func DenyNet(op, target string) error { return errs.New(errs.NetBlocked) }

const (
	defaultNetTimeout = 30 * time.Second
	maxNetSeconds     = 3600
)

// A socket is what a handle number stands for: a connection or a listener.
type socket struct {
	conn    net.Conn
	reader  *bufio.Reader
	ln      net.Listener
	timeout time.Duration // 0: wait as long as it takes
}

// Sockets are values only as numbers (the language has no handle type); the
// table below turns a number back into the open socket.
var sockets = struct {
	sync.Mutex
	next int
	byID map[int]*socket
}{next: 1, byID: map[int]*socket{}}

func addSocket(s *socket) float64 {
	sockets.Lock()
	defer sockets.Unlock()
	id := sockets.next
	sockets.next++
	sockets.byID[id] = s
	return float64(id)
}

func socketArg(args []interface{}, i int) (*socket, int, error) {
	id, err := integerArg(args, i)
	if err != nil {
		return nil, 0, err
	}
	sockets.Lock()
	s := sockets.byID[int(id)]
	sockets.Unlock()
	if s == nil {
		return nil, 0, errs.New(errs.SocketBadHandle, strconv.FormatInt(id, 10))
	}
	return s, int(id), nil
}

func connectionArg(args []interface{}, i int) (*socket, error) {
	s, _, err := socketArg(args, i)
	if err != nil {
		return nil, err
	}
	if s.conn == nil {
		return nil, errs.New(errs.SocketNotConnection)
	}
	return s, nil
}

func portArg(args []interface{}, i int) (int, error) {
	port, err := integerArg(args, i)
	if err != nil {
		return 0, err
	}
	if port < 0 || port > 65535 {
		return 0, errs.New(errs.SocketPort)
	}
	return int(port), nil
}

// netError turns what the network said into one of hana's own wordings; a
// timeout is told apart because a script often waits on purpose.
func netError(err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return errs.New(errs.SocketTimeout)
	}
	return errs.New(errs.SocketFailed)
}

// deadline applies the socket's time limit to the operation about to run.
func (s *socket) deadline() {
	var t time.Time
	if s.timeout > 0 {
		t = time.Now().Add(s.timeout)
	}
	if s.conn != nil {
		s.conn.SetDeadline(t)
	} else if l, ok := s.ln.(*net.TCPListener); ok {
		l.SetDeadline(t)
	}
}

func socketConnect(args []interface{}) (interface{}, error) {
	if err := between(args, 2, 3); err != nil {
		return nil, err
	}
	host, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	port, err := portArg(args, 1)
	if err != nil {
		return nil, err
	}
	timeout := defaultNetTimeout
	if len(args) == 3 {
		if timeout, err = secondsArg(args, 2); err != nil {
			return nil, err
		}
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	if err := NetAccess("connect", addr); err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, errs.New(errs.SocketConnectFailed, addr)
	}
	return addSocket(&socket{conn: conn, reader: bufio.NewReader(conn)}), nil
}

// secondsArg reads a time limit in seconds (0 to 3600).
func secondsArg(args []interface{}, i int) (time.Duration, error) {
	seconds, err := numberArg(args, i)
	if err != nil {
		return 0, err
	}
	if seconds < 0 || seconds > maxNetSeconds {
		return 0, errs.New(errs.SleepRange)
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

// socketListen starts waiting for connections on a port (0 lets the system
// pick one — socketAddress tells which). Only this computer can reach it unless
// a host such as "0.0.0.0" is given.
func socketListen(args []interface{}) (interface{}, error) {
	if err := between(args, 1, 2); err != nil {
		return nil, err
	}
	port, err := portArg(args, 0)
	if err != nil {
		return nil, err
	}
	host := "127.0.0.1"
	if len(args) == 2 {
		if host, err = stringArg(args, 1); err != nil {
			return nil, err
		}
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	if err := NetAccess("listen", addr); err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, errs.New(errs.SocketListenFailed, addr)
	}
	return addSocket(&socket{ln: ln}), nil
}

// socketAccept waits for the next connection to a listening socket.
func socketAccept(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	s, _, err := socketArg(args, 0)
	if err != nil {
		return nil, err
	}
	if s.ln == nil {
		return nil, errs.New(errs.SocketNotListener)
	}
	s.deadline()
	conn, err := s.ln.Accept()
	if err != nil {
		return nil, netError(err)
	}
	return addSocket(&socket{conn: conn, reader: bufio.NewReader(conn)}), nil
}

func socketSend(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	s, err := connectionArg(args, 0)
	if err != nil {
		return nil, err
	}
	text, err := stringArg(args, 1)
	if err != nil {
		return nil, err
	}
	s.deadline()
	if _, err := io.WriteString(s.conn, text); err != nil {
		return nil, netError(err)
	}
	return nil, nil
}

// socketReceive waits for something to arrive and returns up to n characters of
// it; the empty text means the other side has closed the connection.
func socketReceive(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	s, err := connectionArg(args, 0)
	if err != nil {
		return nil, err
	}
	n, err := integerArg(args, 1)
	if err != nil {
		return nil, err
	}
	if n < 1 {
		return nil, errs.New(errs.NativeArgInteger, 2)
	}
	s.deadline()
	var out strings.Builder
	for count := int64(0); count < n; count++ {
		if count > 0 && s.reader.Buffered() == 0 {
			break // everything that has arrived; do not wait for more
		}
		r, _, err := s.reader.ReadRune()
		if err != nil {
			if err == io.EOF && out.Len() > 0 {
				break
			}
			if err == io.EOF {
				return "", nil
			}
			return nil, netError(err)
		}
		out.WriteRune(r)
	}
	return out.String(), nil
}

// socketReceiveLine returns the next line without its line break, or 비어있음
// once the other side has closed the connection and nothing is left.
func socketReceiveLine(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	s, err := connectionArg(args, 0)
	if err != nil {
		return nil, err
	}
	s.deadline()
	line, err := s.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && line == "" {
			return nil, nil
		}
		if err != io.EOF {
			return nil, netError(err)
		}
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// socketTimeout sets how long the socket's later operations may wait (seconds;
// 0 means as long as it takes).
func socketTimeout(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	s, _, err := socketArg(args, 0)
	if err != nil {
		return nil, err
	}
	if s.timeout, err = secondsArg(args, 1); err != nil {
		return nil, err
	}
	return nil, nil
}

// socketAddress is where a listening socket waits, or who a connection is
// with, as "host:port".
func socketAddress(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	s, _, err := socketArg(args, 0)
	if err != nil {
		return nil, err
	}
	if s.ln != nil {
		return s.ln.Addr().String(), nil
	}
	return s.conn.RemoteAddr().String(), nil
}

func socketClose(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	s, id, err := socketArg(args, 0)
	if err != nil {
		return nil, err
	}
	sockets.Lock()
	delete(sockets.byID, id)
	sockets.Unlock()
	if s.ln != nil {
		s.ln.Close()
	} else {
		s.conn.Close()
	}
	return nil, nil
}
