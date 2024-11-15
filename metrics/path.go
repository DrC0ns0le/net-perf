package metrics

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
)

type PathMetrics struct {
	Availability float64
	Latency      float64
	Jitter       float64
	PacketLoss   float64
}

/*************  ✨ Codeium Command ⭐  *************/
// GetPreferredPath takes a remote endpoint string and returns the preferred
// IP version to use for communication with that endpoint. It does this by
// comparing the latency and packet loss metrics for both IPv4 and IPv6
// connections to the remote endpoint. If the metrics are equal, it returns an
// empty string. If the metrics for one version are significantly better than
// the other, it returns that version. Otherwise, it returns the version with the
// lowest latency.
//
// The returned string is the preferred IP version, the cost of using that version,
// and a string explaining why that version was chosen.
/******  63b20803-472b-41ad-b127-165a85a05065  *******/
func GetPreferredPath(ctx context.Context, origin, remote int) (string, float64, string, error) {

	metrics, err := GetPathMetrics(ctx, origin, remote)
	if err != nil {
		return "", 0, "", err
	}

	versionScoreMap := map[string]float64{
		"4": 0.0,
		"6": 0.0,
	}

	for v, m := range metrics {
		if m.Latency == 0 {
			continue
		}
		versionScoreMap[v] = 1 / ((m.Latency / 1e6) * (math.Sqrt(m.PacketLoss / 100)))
	}

	if int(metrics["4"].Availability) != 1 && int(metrics["6"].Availability) != 1 {
		if metrics["4"].Availability > metrics["6"].Availability {
			return "4", calculatePathCost(metrics["4"]), "higher availability", nil
		} else {
			return "6", calculatePathCost(metrics["6"]), "higher availability", nil
		}
	} else if versionScoreMap["4"] == math.Inf(1) && versionScoreMap["6"] == math.Inf(1) {
		if metrics["4"].Latency < metrics["6"].Latency {
			return "4", calculatePathCost(metrics["4"]), "lower latency", nil
		} else {
			return "6", calculatePathCost(metrics["6"]), "lower latency", nil
		}
	} else {
		if versionScoreMap["4"] > versionScoreMap["6"] {
			return "4", calculatePathCost(metrics["4"]), "higher score", nil
		} else {
			return "6", calculatePathCost(metrics["6"]), "higher score", nil
		}
	}
}

// GetPathMetrics queries prometheus for the average latency and packet loss for both
// IPv4 and IPv6 paths to the given remote. It returns a map where the keys are the
// IP versions and the values are PathMetrics objects.
//
// The metrics queried are:
//   - network_latency_duration: average latency in microseconds
//   - network_latency_loss and network_bandwidth_packet_loss: average packet loss
//     as a percentage.
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
func calculatePathCost(pathMetrics *PathMetrics) float64 {
	if pathMetrics == nil || pathMetrics.Availability == 0 || pathMetrics.Latency == 0 {
		return math.Inf(1)
	}

	const (
		K1 = 1.0 // Latency weight
		K2 = 1.0 // Load/Loss weight
		K3 = 0.5 // Jitter weight
	)

	latencyMs := pathMetrics.Latency / 1e6
	normalizedLoss := pathMetrics.PacketLoss / 100

	if normalizedLoss >= 1 {
		return math.Inf(1)
	}

	jitterMs := pathMetrics.Jitter / 1e6

	cost := K1*latencyMs +
		K2*(latencyMs*normalizedLoss/(1-normalizedLoss)) +
		K3*jitterMs

	if pathMetrics.Availability < 1 {
		cost /= pathMetrics.Availability
	}

	return cost * 1000
}
