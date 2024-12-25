package measure

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/measure/bandwidth"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	bandwidthStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_bandwidth_status",
		Help: "outcome of network_bandwidth",
	}, []string{"type", "source", "target", "version", "bandwidth", "packet_size", "duration", "path"})
	bandwidthPacketLoss = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_bandwidth_packet_loss",
		Help: "percentage of packet loss for network_bandwidth",
	}, []string{"type", "source", "target", "version", "bandwidth", "packet_size", "duration", "path"})
	bandwidthJitter = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_bandwidth_jitter",
		Help: "jitter in microseconds for network_bandwidth",
	}, []string{"type", "source", "target", "version", "bandwidth", "packet_size", "duration", "path"})
	bandwidthOutOfOrder = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_bandwidth_out_of_order",
		Help: "number of out-of-order packets for network_bandwidth",
	}, []string{"type", "source", "target", "version", "bandwidth", "packet_size", "duration", "path"})
	bandwidthResult = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_bandwidth_result",
		Help: "result of network_bandwidth in bits per second",
	}, []string{"type", "source", "target", "version", "bandwidth", "packet_size", "duration", "path"})
)

func startBandwidthWorker(worker *Worker) {

	key := fmt.Sprintf("iface=%s, sourceIP=%s, targetIP=%s, targetPort=%d",
		worker.iface.Name, worker.sourceIP, worker.targetIP, worker.targetPort)

	hash := sha256.Sum256([]byte(key))
	h := binary.BigEndian.Uint64(hash[:8])

	randSleep := time.Duration(float64(50*time.Second) * (float64(h) / (1 << 64)))
	time.Sleep(randSleep)

	// log.Printf("Starting bandwidth measurement on %s\n", worker.iface.Name)
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			go func() {
				// measure
				data, err := bandwidth.MeasureUDP(ctx, worker.sourceIP, worker.targetIP)
				if err != nil {
					logging.Errorf("%s: error measuring UDP bandwidth: %v", worker.iface.Name, err)
				}

				// generate metrics
				generateBandwidthMetrics(data, worker.iface)
			}()

		case <-worker.stopCh:
			log.Printf("Stopping bandwidth measurement on %s", worker.iface.Name)

			// Set metrics to NaN
			generateBandwidthMetrics(bandwidth.Result{}, worker.iface)
			return
		}
	}
}

func generateBandwidthMetrics(data bandwidth.Result, iface netctl.WGInterface) {
	var (
		jitter     float64
		outOfOrder float64
		packetLoss float64
		bandwidth  float64
	)

	if data.Status == 1 {
		jitter = data.Jitter
		outOfOrder = data.OutOfOrder
		packetLoss = data.Loss
		bandwidth = float64(data.Bandwidth)
	} else {
		jitter = math.NaN()
		outOfOrder = math.NaN()
		packetLoss = math.NaN()
		bandwidth = math.NaN()
	}

	pathName := generatePathName(iface.LocalID, iface.RemoteID)

	bandwidthStatus.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		iface.IPVersion,
		strconv.Itoa(data.TargetBandwidth),
		strconv.Itoa(data.PacketSize),
		strconv.Itoa(data.TargetDuration),
		pathName,
	).Set(float64(data.Status))

	bandwidthJitter.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		iface.IPVersion,
		strconv.Itoa(data.TargetBandwidth),
		strconv.Itoa(data.PacketSize),
		strconv.Itoa(data.TargetDuration),
		pathName,
	).Set(jitter)

	bandwidthPacketLoss.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		iface.IPVersion,
		strconv.Itoa(data.TargetBandwidth),
		strconv.Itoa(data.PacketSize),
		strconv.Itoa(data.TargetDuration),
		pathName,
	).Set(packetLoss)

	bandwidthResult.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		iface.IPVersion,
		strconv.Itoa(data.TargetBandwidth),
		strconv.Itoa(data.PacketSize),
		strconv.Itoa(data.TargetDuration),
		pathName,
	).Set(bandwidth)

	bandwidthOutOfOrder.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		iface.IPVersion,
		strconv.Itoa(data.TargetBandwidth),
		strconv.Itoa(data.PacketSize),
		strconv.Itoa(data.TargetDuration),
		pathName,
	).Set(outOfOrder)

}
