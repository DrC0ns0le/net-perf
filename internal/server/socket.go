package server

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

var (
	socketPath              = flag.String("socket.path", "/opt/wg-mesh/net-perf.sock", "path for unix socket")
	socketConnectionTimeout = flag.Duration("socket.timeout", 5*time.Second, "timeout for socket connection")
)

type SocketServer struct {
	wgUpdateCh      chan netctl.WGInterface
	measureUpdateCh chan struct{}
	socketPath      string

	listener net.Listener
	logger   logging.Logger
}

func NewSocketServer(global *system.Node) *SocketServer {
	return &SocketServer{
		wgUpdateCh:      global.WGUpdateCh,
		measureUpdateCh: global.MeasureUpdateCh,

		socketPath: *socketPath,
		logger:     global.Logger.With("component", "socket"),
	}
}

func (s *SocketServer) Start() error {
	if err := os.RemoveAll(s.socketPath); err != nil {
		return fmt.Errorf("error removing existing socket: %w", err)
	}

	var (
		conn net.Conn
		err  error
	)

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = listener

	s.logger.With("component", "socket").Infof("watchdog socket listening at %s", s.socketPath)
	go func() {
		for {
			conn, err = s.listener.Accept()
			if err != nil {
				s.logger.With("component", "socket").Errorf("error accepting connection: %v", err)
				continue
			}
			go s.handleConnection(conn)
		}
	}()

	return nil
}

func (s *SocketServer) Stop() error {
	return s.listener.Close()
}

func (s *SocketServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(*socketConnectionTimeout))

	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				s.logger.Errorf("error reading from socket: %v", err)
			}
			return
		}

		message = strings.TrimSpace(message)
		s.logger.Infof("received message from socket: %s", message)

		switch {
		case strings.HasPrefix(message, "WGADD"):
			// parse the wg interface
			ifname := strings.TrimPrefix(message, "WGADD")
			wgIface, err := netctl.ParseWGInterface(strings.TrimSpace(ifname))
			if err != nil {
				s.logger.Errorf("error parsing wg interface: %v", err)
				conn.Write([]byte(fmt.Sprintf("ERROR: %v\n", err)))
				continue
			}

			select {
			case s.wgUpdateCh <- wgIface:
				conn.Write([]byte("OK\n"))
			case <-time.After(*socketConnectionTimeout):
				conn.Write([]byte("ERROR: timeout writing to channel\n"))
			}
		case strings.HasPrefix(message, "WGRESTART"):
			select {
			case s.measureUpdateCh <- struct{}{}:
				conn.Write([]byte("OK\n"))
			case <-time.After(*socketConnectionTimeout):
				conn.Write([]byte("ERROR: timeout writing to channel\n"))
			}
		default:
			conn.Write([]byte("ERROR: invalid message format\n"))
		}
	}
}
