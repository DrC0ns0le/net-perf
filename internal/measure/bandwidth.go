package measure

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
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

		wg                      = &sync.WaitGroup{}
		test                    = bandwidth.NewMeasureClient(worker.sourceIP, worker.targetIP, worker.logger)
		workerCtx, workerCancel = context.WithCancel(context.Background())

		doMeasure = func() {
			ctx, cancel := context.WithTimeout(workerCtx, 5*time.Second)

			wg.Add(1)
			go func() {
				defer wg.Done()
				// measure
				data, err = test.MeasureUDP(ctx)
				if err != nil {
					worker.logger.Errorf("error measuring UDP bandwidth: %v", err)
				}

				// generate metrics
				generateBandwidthMetrics(data, worker.iface)
			}()

			wg.Wait()
			cancel()
		}
	)

	worker.logger.Debugf("starting bandwidth measurement on %s\n", worker.iface.Name)

	// start ticker
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// initial measurement
	doMeasure()
	for {
		select {
		case <-ticker.C:
			doMeasure()
		case <-worker.stopCh:
			worker.logger.Infof("stopping bandwidth measurement on %s", worker.iface.Name)

			// stop all measurements
			ticker.Stop()
			workerCancel()
			wg.Wait()

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

	generateBandwidthMetrics(bandwidth.Result{Status: 0, Protocol: data.Protocol, TargetBandwidth: data.TargetBandwidth, PacketSize: data.PacketSize, TargetDuration: data.TargetDuration}, iface)
	bandwidthStatus.WithLabelValues(
		data.Protocol,
		iface.LocalID,
		iface.RemoteID,
		iface.IPVersion,
		strconv.Itoa(data.TargetBandwidth),
		strconv.Itoa(data.PacketSize),
		strconv.Itoa(data.TargetDuration),
		pathName,
	).Set(math.NaN())

	// metrics := []*prometheus.GaugeVec{
	// 	bandwidthStatus,
	// 	bandwidthJitter,
	// 	bandwidthPacketLoss,
	// 	bandwidthResult,
	// 	bandwidthOutOfOrder,
	// }

	// for _, metric := range metrics {
	// 	ok := metric.DeleteLabelValues(
	// 		data.Protocol,
	// 		iface.LocalID,
	// 		iface.RemoteID,
	// 		iface.IPVersion,
	// 		strconv.Itoa(data.TargetBandwidth),
	// 		strconv.Itoa(data.PacketSize),
	// 		strconv.Itoa(data.TargetDuration),
	// 		pathName,
	// 	)

	// 	if !ok {
	// 		return fmt.Errorf("failed to delete %s bandwidth metrics for %s", data.Protocol, iface.Name)
	// 	}
	// }

	return nil
}
