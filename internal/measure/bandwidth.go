package measure

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/measure/bandwidth"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/cespare/xxhash"
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

	h := xxhash.Sum64String(key)

	randSleep := time.Duration(float64(50*time.Second) * (float64(h) / (1 << 64)))
	time.Sleep(randSleep)

	var (
		data bandwidth.Result
		err  error

		test = bandwidth.NewMeasureClient(worker.sourceIP, worker.targetIP, worker.logger)
	)

	worker.logger.Debugf("starting bandwidth measurement on %s\n", worker.iface.Name)
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			go func() {
				// measure
				data, err = test.MeasureUDP(ctx)
				if err != nil {
					worker.logger.Errorf("error measuring UDP bandwidth: %v", err)
				}

				// generate metrics
				generateBandwidthMetrics(data, worker.iface)
			}()

		case <-worker.stopCh:
			worker.logger.Infof("stopping bandwidth measurement on %s", worker.iface.Name)

			// remove worker metrics
			err = unregisterBandwidthMetrics(data, worker.iface)
			if err != nil {
				worker.logger.Errorf("error unregistering bandwidth metrics: %v", err)
			}
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

func unregisterBandwidthMetrics(data bandwidth.Result, iface netctl.WGInterface) error {
	pathName := generatePathName(iface.LocalID, iface.RemoteID)

	metrics := []*prometheus.GaugeVec{
		bandwidthStatus,
		bandwidthJitter,
		bandwidthPacketLoss,
		bandwidthResult,
		bandwidthOutOfOrder,
	}

	for _, metric := range metrics {
		ok := metric.DeleteLabelValues(
			data.Protocol,
			iface.LocalID,
			iface.RemoteID,
			iface.IPVersion,
			strconv.Itoa(data.TargetBandwidth),
			strconv.Itoa(data.PacketSize),
			strconv.Itoa(data.TargetDuration),
			pathName,
		)

		if !ok {
			fmt.Errorf("failed to delete bandwidth metrics")
		}
	}

	return nil
}
