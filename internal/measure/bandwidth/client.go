package bandwidth

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

type Client struct {
	SourceIP net.IP
	TargetIP net.IP

	Logger logging.Logger
}

type client struct {
	// UDP connection
	conn net.Conn

	packetsCount uint32

	logger logging.Logger

	// Statistics
	result Result

	statsChan chan string
	errorChan chan error
	stopCh    chan struct{}

	wg sync.WaitGroup
}

func NewMeasureClient(sourceIP, targetIP string, logger logging.Logger) *Client {
	C := &Client{
		SourceIP: net.ParseIP(sourceIP),
		TargetIP: net.ParseIP(targetIP),
		Logger:   logger,
	}

	return C
}

func (c *Client) MeasureUDP(ctx context.Context) (Result, error) {

	dialer := net.Dialer{
		LocalAddr: &net.UDPAddr{IP: c.SourceIP},
	}

	conn, err := dialer.Dial("udp4", net.JoinHostPort(c.TargetIP.String(), strconv.Itoa(*bandwidthPort)))
	if err != nil {
		return Result{}, err
	}
	defer conn.Close()

	cl := &client{
		conn:      conn,
		statsChan: make(chan string),
		errorChan: make(chan error, 2),
		stopCh:    make(chan struct{}),

		logger: c.Logger,

		result: Result{
			Protocol:        "udp",
			TargetBandwidth: *bandwidthBandwidth,
			PacketSize:      *bandwidthPacketSize,
			TargetDuration:  int(bandwidthDuration.Seconds()),
		},

		packetsCount: (uint32(*bandwidthBandwidth) * 1000000 * uint32(bandwidthDuration.Seconds())) / (uint32(*bandwidthPacketSize) * 8),
	}
	defer cl.cleanup()

	err = cl.runTest()
	if err != nil {
		return cl.result, err
	}

	return cl.result, nil
}

func (c *client) cleanup() {
	close(c.stopCh)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		c.logger.Warnf("cleanup timed out, some goroutines may not have finished")
	}

	close(c.statsChan)
	close(c.errorChan)
}

// runTest runs the test, sending packets to the server and receiving stats.
// It starts a goroutine to receive messages from the server and will return an error
// if any error is received from the server. It also sends an end of test notification
// once all packets have been sent and waits for the final stats from the server.
// If no final stats are received within a certain time, it will retry up to
// bandwidthMaxRetries times. If all retries fail, it will return an error.
func (c *client) runTest() error {
	// goroutine to receive messages from the server
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.receiveMessage()
	}()

	interval := *bandwidthDuration / time.Duration(c.packetsCount)
	var seqNumber uint32
	startTime := time.Now()

	buffer := make([]byte, *bandwidthPacketSize)
	for seqNumber = 1; seqNumber < c.packetsCount; seqNumber++ {
		targetTime := startTime.Add(time.Duration(seqNumber) * interval)

		packet := Packet{
			SequenceNumber: seqNumber,
			Timestamp:      time.Now().UnixNano(),
		}
		binary.BigEndian.PutUint32(buffer[:4], packet.SequenceNumber)
		binary.BigEndian.PutUint64(buffer[4:12], uint64(packet.Timestamp))
		// fill with random data
		for i := 12; i < *bandwidthPacketSize; i++ {
			buffer[i] = byte(seqNumber)
		}

		_, err := c.conn.Write(buffer)
		if err != nil {
			c.logger.Errorf("error sending packet to %s: %v", c.conn.RemoteAddr().String(), err)
		}

		select {
		case err := <-c.errorChan:
			return fmt.Errorf("error during test: %v", err)
		default:
			// Continue if no error
		}

		// Wait until the next target time
		sleepTime := time.Until(targetTime)
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}
	}

	c.result.Duration = time.Since(startTime).Seconds()

	c.logger.Debugf("Test completed, client took %v seconds. Sent %d packets", c.result.Duration, seqNumber)

	var err error

retryLoop:
	// Wait for final stats
	for retry := 0; retry < *bandwidthMaxRetries; retry++ {
		// Send end of test notification
		if err = c.sendEndOfTest(); err != nil {
			return fmt.Errorf("error sending end of test to %s: %v", c.conn.RemoteAddr().String(), err)
		}
		// Wait for final stats
		select {
		case finalStats := <-c.statsChan:
			err = c.parseStats(finalStats)
			if err != nil {
				return fmt.Errorf("error parsing final stats from %s: %v", c.conn.RemoteAddr().String(), err)
			}
			break retryLoop
		case err = <-c.errorChan:
		case <-time.After(*bandwidthRetryDelay):
			c.logger.Debugf("no acknowledgment received from %s (attempt %d), retrying...", c.conn.RemoteAddr().String(), retry+1)
		}

		if retry == *bandwidthMaxRetries-1 {
			if err != nil {
				return fmt.Errorf("failed to complete test with %s after %d attempts: %v", c.conn.RemoteAddr().String(), *bandwidthMaxRetries, err)
			}
			return fmt.Errorf("failed to complete test with %s after %d attempts", c.conn.RemoteAddr().String(), *bandwidthMaxRetries)
		}
	}

	// Print client-side summary
	c.logger.Debugf("Client-side summary:")
	c.logger.Debugf("Total packets sent: %d", seqNumber)
	c.logger.Debugf("Test duration: %v", time.Since(startTime))
	c.logger.Debugf("Target bandwidth: %d Mbps", c.result.TargetBandwidth)
	c.logger.Debugf("Actual bandwidth: %d Mbps", c.result.Bandwidth)

	return nil
}

func (c *client) sendEndOfTest() error {
	endPacket := Packet{
		SequenceNumber: math.MaxUint32,
		Timestamp:      math.MaxInt64,
	}
	buffer := make([]byte, *bandwidthPacketSize) // Use the same packet size as in the test
	binary.BigEndian.PutUint32(buffer[:4], endPacket.SequenceNumber)
	binary.BigEndian.PutUint64(buffer[4:12], uint64(endPacket.Timestamp))

	_, err := c.conn.Write(buffer)
	if err != nil {
		return err
	}

	return nil
}

func (c *client) receiveMessage() {
	buffer := make([]byte, 1024)
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		// Set read deadline to 3x the stats interval
		// Acts as a heartbeat, ensuring the test is still alive
		err := c.conn.SetReadDeadline(time.Now().Add(*bandwidthStatsInterval * 3))
		if err != nil {
			c.errorChan <- fmt.Errorf("error setting read deadline: %v", err)
			return
		}
		n, err := c.conn.Read(buffer)
		if err != nil {
			c.errorChan <- fmt.Errorf("error receiving message from %s: %v", c.conn.RemoteAddr().String(), err)
			return
		}

		// Parse the message
		if strings.HasPrefix(string(buffer[:n]), "STATS") {
			// Interim Stats/Heartbeat
		} else if strings.HasPrefix(string(buffer[:n]), "FINAL") {
			// Final stats
			c.statsChan <- string(buffer[:n])
		} else {
			c.errorChan <- fmt.Errorf("unknown message type from %s: %s", c.conn.RemoteAddr().String(), string(buffer[:n]))
		}
	}

}

func (c *client) parseStats(statString string) error {
	parts := strings.Split(statString, "|")
	if len(parts) < 5 {
		return fmt.Errorf("invalid stat string format")
	}

	switch parts[0] {
	case "STATS", "FINAL":
		totalPacketsReceived, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("error parsing total packets: %v", err)
		}
		if totalPacketsReceived <= 10 {
			return fmt.Errorf("invalid total packets received: %d", totalPacketsReceived)
		}
		outOfOrder, err := strconv.Atoi(parts[2])
		if err != nil {
			return fmt.Errorf("error parsing out of order packets: %v", err)
		}
		c.result.OutOfOrder = 100 * (float64(outOfOrder) / float64(totalPacketsReceived))
		droppedPackets, err := strconv.Atoi(parts[3])
		if err != nil {
			return fmt.Errorf("error parsing lost packets: %v", err)
		}
		c.result.Loss = 100 * (float64(int(c.packetsCount)-totalPacketsReceived-droppedPackets) / float64(c.packetsCount))

		jitter, err := strconv.ParseFloat(parts[4], 64)
		if err != nil {
			return fmt.Errorf("error parsing jitter: %v", err)
		}
		c.result.Jitter = math.Sqrt(jitter)

		if parts[0] == "FINAL" {
			if len(parts) != 7 {
				return fmt.Errorf("invalid final stat string format")
			}
			c.result.Status = 1
			testDuration, err := strconv.Atoi(parts[6])
			if err != nil {
				return fmt.Errorf("error parsing duration: %v", err)
			}
			c.result.Bandwidth = (*bandwidthPacketSize * 8 * totalPacketsReceived) / (testDuration / 1000000)
		} else {
			c.result.Status = 0
		}
	default:
		return fmt.Errorf("unknown stat type")
	}

	return nil
}
