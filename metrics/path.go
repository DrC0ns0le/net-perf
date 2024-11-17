package metrics

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
)

type PathMetrics struct {
	Availability float64 // in percentage (0-1)
	Latency      float64 // in microseconds
	Jitter       float64 // in microseconds
	PacketLoss   float64 // in percentage (0-1)
	Cost         float64
}

// GetPreferredPath returns the preferred IP version to use for communication with the given remote endpoint
// along with the associated cost. If no valid paths are available, it returns an empty string and an error.
func GetPreferredPath(ctx context.Context, origin, remote int) (string, float64, error) {
	metrics, err := GetPathMetrics(ctx, origin, remote)
	if err != nil {
		return "", 0, err
	}

	var bestVersion string
	var bestCost float64 = math.Inf(1)

	for version, path := range metrics {
		if !math.IsInf(path.Cost, 1) && (math.IsInf(bestCost, 1) || path.Cost < bestCost) {
			bestVersion = version
			bestCost = path.Cost
		}
	}

	if bestVersion == "" {
		return "", 0, errors.New("no valid paths available")
	}

	return bestVersion, bestCost, nil
}

// GetPathMetrics queries prometheus for the average latency, packet loss, and availability for both
// IPv4 and IPv6 paths to the given remote. It returns a map where the keys are the
// IP versions and the values are PathMetrics objects.
//
// The metrics queried are:
//   - network_latency_duration: average latency in microseconds
//   - network_latency_loss and network_bandwidth_packet_loss: average packet loss
//     as a percentage.
//   - network_latency_status: average availability as a percentage.
//
// The values are averaged over a 5 minute time window.
func GetPathMetrics(ctx context.Context, origin, remote int) (map[string]*PathMetrics, error) {

	var metrics map[string]*PathMetrics

	// assumes dual-stack
	path := GetPathLabel(origin, remote)
	if path == "" {
		return metrics, errors.New("could not generate path label")
	}
	queries := []string{
		fmt.Sprintf(`avg(avg_over_time(network_latency_duration{path=~"%s"}[5m])) by (version)`, path),
		fmt.Sprintf(`avg(avg_over_time(network_latency_loss{path=~"%s"}[5m]),avg_over_time(network_bandwidth_packet_loss{path=~"%s"}[5m])) by (version)`, path, path),
		fmt.Sprintf(`avg(avg_over_time(network_latency_status{path=~"%s"}[5m])) by (version)`, path),
	}

	metrics = map[string]*PathMetrics{
		"4": {},
		"6": {},
	}

	for i, query := range queries {
		response, err := Query(ctx, query)
		if err != nil {
			return metrics, err
		}

		for _, result := range response.Data.Result {
			switch i {
			case 0:
				latency, err := strconv.ParseFloat(result.Value[1].(string), 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing latency: %v", err)
				}
				metrics[result.Metric.Version].Latency = latency
			case 1:
				loss, err := strconv.ParseFloat(result.Value[1].(string), 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing packet loss: %v", err)
				}
				metrics[result.Metric.Version].PacketLoss = loss
			case 2:
				availability, err := strconv.ParseFloat(result.Value[1].(string), 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing availability: %v", err)
				}
				metrics[result.Metric.Version].Availability = availability
			}
		}
	}

	for _, metric := range metrics {
		metric.calculatePathCost()
	}

	return metrics, nil
}

// GetPathLabel generates a label string for a path between two nodes. The string is
// in the format "X-Y", where X and Y are the node IDs. The IDs are sorted so that
// the smaller one is first, to ensure that the same path is always labelled with
// the same string, regardless of the order in which the nodes are given. If the
// IDs can't be parsed as integers, an empty string is returned.
func GetPathLabel(origin, remote int) string {
	if origin > remote {
		origin, remote = remote, origin
	}

	return strconv.Itoa(origin) + "-" + strconv.Itoa(remote)
}

// calculatePathCost calculates a cost for a path based on its latency, packet loss and jitter metrics.
// The cost is a weighted sum of the latency, packet loss and jitter metrics, where the weights are
// K1, K2 and K3 respectively. If the packet loss is 100% or the availability is 0, the cost is set to
// positive infinity. Otherwise, the cost is multiplied by 1000 and returned.
func (pm *PathMetrics) calculatePathCost() {
	if pm == nil || pm.Availability == 0 || pm.Latency == 0 {
		pm.Cost = math.Inf(1)
		return
	}

	const (
		K1 = 1.0 // Latency weight
		K2 = 1.0 // Load/Loss weight
		K3 = 0.5 // Jitter weight
	)

	latencyMs := pm.Latency / 1e3
	jitterMs := pm.Jitter / 1e3

	normalizedLoss := pm.PacketLoss
	if normalizedLoss >= 1 {
		pm.Cost = math.Inf(1)
		return
	}

	pm.Cost = (K1*latencyMs + K2*(latencyMs*normalizedLoss/(1-normalizedLoss)) + K3*jitterMs) / pm.Availability
}
