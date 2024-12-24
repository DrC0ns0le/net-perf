package bandwidth

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	// UDP connection
	conn net.Conn

	packetsCount uint32

	// Statistics
	result Result

	statsChan chan string
	errorChan chan error
	ackChan   chan struct{}
	stopCh    chan struct{}
}

func MeasureUDP(ctx context.Context, sourceIP, serverAddr string) (Result, error) {
	dialer := net.Dialer{
		LocalAddr: &net.UDPAddr{IP: net.ParseIP(sourceIP)},
	}

	conn, err := dialer.Dial("udp4", net.JoinHostPort(serverAddr, strconv.Itoa(defaultPort)))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := newClient(conn)

	// log.Printf("Connected to server %s at %s", conn.RemoteAddr().String(), conn.LocalAddr().String())

	result, err := client.runTest(conn)
	if err != nil {
		return result, err
	}

	return result, nil
}

func newClient(conn net.Conn) *Client {

	client := &Client{
		conn:      conn,
		statsChan: make(chan string),
		errorChan: make(chan error),
		ackChan:   make(chan struct{}),
		stopCh:    make(chan struct{}),

		result: Result{
			Protocol:        "udp",
			TargetBandwidth: bandwidth,
			PacketSize:      packetSize,
			TargetDuration:  int(duration.Seconds()),
		},

		packetsCount: (uint32(bandwidth) * 1000000 * uint32(duration.Seconds())) / (uint32(packetSize) * 8),
	}

	// goroutine to receive statistics
	go client.receiveMessage()

	return client
}

func (c *Client) runTest(conn net.Conn) (Result, error) {
	interval := duration / time.Duration(c.packetsCount)
	var seqNumber uint32
	startTime := time.Now()

	for seqNumber = 1; seqNumber < c.packetsCount; seqNumber++ {
		targetTime := startTime.Add(time.Duration(seqNumber) * interval)

		packet := Packet{
			SequenceNumber: seqNumber,
			Timestamp:      time.Now().UnixNano(),
		}
		buffer := make([]byte, packetSize)
		binary.BigEndian.PutUint32(buffer[:4], packet.SequenceNumber)
		binary.BigEndian.PutUint64(buffer[4:12], uint64(packet.Timestamp))
		// fill with random data
		for i := 12; i < packetSize; i++ {
			buffer[i] = byte(seqNumber)
		}

		_, err := conn.Write(buffer)
		if err != nil {
			log.Printf("Error sending packet to %s: %v", conn.RemoteAddr().String(), err)
		}

		select {
		case err := <-c.errorChan:
			return c.result, fmt.Errorf("error during test: %v", err)
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

	// log.Printf("Test completed, client took %v seconds. Sent %d packets", c.result.Duration, seqNumber)

	// Send end of test notification
	if err := c.sendEndOfTest(conn, seqNumber); err != nil {
		return c.result, fmt.Errorf("error sending end of test to %s: %v", conn.RemoteAddr().String(), err)
	}

	// Wait for final statistics
	select {
	case finalStats := <-c.statsChan:
		err := c.parseStats(finalStats)
		if err != nil {
			return c.result, fmt.Errorf("error parsing final stats from %s: %v", conn.RemoteAddr().String(), err)
		}
		// log.Printf("Final stats from server: %+v", c.result)
	case err := <-c.errorChan:
		return c.result, fmt.Errorf("error receiving final stats from %s: %v", conn.RemoteAddr().String(), err)
	case <-time.After(10 * time.Second):
		return c.result, fmt.Errorf("timeout waiting for final stats from %s", conn.RemoteAddr().String())
	}

	// // Print client-side summary
	// log.Printf("Client-side summary:")
	// log.Printf("Total packets sent: %d", seqNumber)
	// log.Printf("Test duration: %v", time.Since(startTime))
	// log.Printf("Target bandwidth: %d Mbps", bandwidth)

	return c.result, nil
}

func (c *Client) sendEndOfTest(conn net.Conn, seqNumber uint32) error {
	endPacket := Packet{
		SequenceNumber: seqNumber,
		// Timestamp:      int64(binary.BigEndian.Uint64(padToEight("TESTEND"))),
		Timestamp: 101,
	}
	buffer := make([]byte, packetSize) // Use the same packet size as in the test
	binary.BigEndian.PutUint32(buffer[:4], endPacket.SequenceNumber)
	binary.BigEndian.PutUint64(buffer[4:12], uint64(endPacket.Timestamp))

	for retry := 0; retry < maxRetries; retry++ {
		_, err := conn.Write(buffer)
		if err != nil {
			log.Printf("Error sending end packet to %s (attempt %d): %v", conn.RemoteAddr().String(), retry+1, err)
			// } else {
			// log.Printf("Sent end of test notification (attempt %d)", retry+1)
		}

		// Wait for acknowledgment
		select {
		case <-c.ackChan:
			// log.Println("Received acknowledgment from server")
			return nil
		case <-time.After(retryDelay):
			log.Printf("No acknowledgment received from %s (attempt %d), retrying...", conn.RemoteAddr().String(), retry+1)
		}
	}

	return fmt.Errorf("failed to receive acknowledgment from %s after %d attempts", conn.RemoteAddr().String(), maxRetries)
}

func (c *Client) receiveMessage() {
	buffer := make([]byte, 1024)
	for {
		n, err := c.conn.Read(buffer)
		if err != nil {
			c.errorChan <- fmt.Errorf("error receiving message from %s: %v", c.conn.RemoteAddr().String(), err)
			return
		}
		parts := strings.Split(string(buffer[:n]), "|")
		switch parts[0] {
		case "ACK":
			c.ackChan <- struct{}{}
		case "STATS":
			err := c.parseStats(string(buffer[:n]))
			if err != nil {
				log.Println("Error parsing stats:", err)
				// } else {
				// 	log.Printf("Received %s interim stats: %+v", c.conn.RemoteAddr(), c.result)
			}
		case "FINAL":
			c.statsChan <- string(buffer[:n])
		default:
			c.errorChan <- fmt.Errorf("unknown message type from %s: %s", c.conn.RemoteAddr().String(), parts[0])
		}
	}
}

func (c *Client) parseStats(statString string) error {
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
			c.result.Bandwidth = (packetSize * 8 * totalPacketsReceived) / (testDuration / 1000000)
		} else {
			c.result.Status = 0
		}
	default:
		return fmt.Errorf("unknown stat type")
	}

	return nil
}
