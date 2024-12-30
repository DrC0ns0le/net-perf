package watchdog

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
	SocketPath              = flag.String("socket.path", "/opt/wg-mesh/net-perf.sock", "path for unix socket")
	SocketConnectionTimeout = flag.Duration("socket.timeout", 5*time.Second, "timeout for socket connection")
)

func Serve(global *system.Node) error {
	if err := os.RemoveAll(*SocketPath); err != nil {
		return fmt.Errorf("error removing existing socket: %w", err)
	}

	listener, err := net.Listen("unix", *SocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	global.Logger.With("component", "socket").Infof("Watchdog socket listening at %s", *SocketPath)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-global.StopCh:
					listener.Close()
					return
				default:
					global.Logger.With("component", "socket").Errorf("error accepting connection: %v", err)
					continue
				}
			}
			go HandleConnection(conn, global.WGUpdateCh, global.Logger.With("component", "socket"))
		}
	}()

	return nil
}

func HandleConnection(conn net.Conn, WGUpdateCh chan netctl.WGInterface, logger logging.Logger) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(*SocketConnectionTimeout))

	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				logger.Errorf("error reading from socket: %v", err)
			}
			return
		}

		message = strings.TrimSpace(message)
		logger.Debugf("Received message from socket: %s", message)

		if strings.HasPrefix(message, "wg") {
			wgIface, err := netctl.ParseWGInterface(message)
			if err != nil {
				logger.Errorf("error parsing wg interface: %v", err)
				conn.Write([]byte(fmt.Sprintf("ERROR: %v\n", err)))
				continue
			}

			select {
			case WGUpdateCh <- wgIface:
				conn.Write([]byte("OK\n"))
			case <-time.After(*SocketConnectionTimeout):
				conn.Write([]byte("ERROR: timeout writing to channel\n"))
			}
		} else {
			conn.Write([]byte("ERROR: invalid message format\n"))
		}
	}
}
