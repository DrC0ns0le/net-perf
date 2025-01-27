package analyser

import (
	"container/heap"
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/measure/pathping"
	"github.com/DrC0ns0le/net-perf/internal/route/finder"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/networkanalysis"
)

type Path struct {
	Nodes    []int
	Segments [][2]int
	Cost     float64
}

type NetworkAnalyzer struct {
	numNodes     int
	targetStart  int
	targetEnd    int
	graph        [][]float64
	solverClient pb.NetworkAnalyzerClient
	ppClient     *pathping.Client
}

func NewNetworkAnalyzer() *NetworkAnalyzer {
	return &NetworkAnalyzer{
		ppClient: pathping.NewClient(15, 100*time.Millisecond),
	}
}

func (na *NetworkAnalyzer) CreateGraph() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	graph, err := finder.NewGraph(ctx)
	if err != nil {
		return fmt.Errorf("failed to create graph: %w", err)
	}

	na.graph = graph.GetMatrix()
	na.numNodes = len(na.graph)
	return nil
}

func (na *NetworkAnalyzer) findTargetContainingPaths(maxLength int) []Path {
	if maxLength == 0 {
		maxLength = na.numNodes * 2
	}

	paths := make([]Path, 0, 100)
	targetStart := na.targetStart
	targetEnd := na.targetEnd

	startNodes := []int{targetStart, targetEnd}
	for _, startNode := range startNodes {
		pq := make(PriorityQueue, 0)
		heap.Init(&pq)
		heap.Push(&pq, queueItem{0, []int{startNode}, false})

		for pq.Len() > 0 {
			current := heap.Pop(&pq).(queueItem)
			currentNode := current.path[len(current.path)-1]

			if len(current.path) > 2 {
				if na.graph[currentNode][startNode] > 0 {
					if current.usedTarget {
						cyclePath := append(current.path, startNode)
						paths = append(paths, Path{Nodes: cyclePath})
					}
					continue
				}
			}

			if len(current.path) < maxLength {
				for nextNode := 0; nextNode < na.numNodes; nextNode++ {
					if na.graph[currentNode][nextNode] <= 0 {
						continue
					}

					if containsNode(current.path, nextNode) {
						continue
					}

					isTargetEdge := (currentNode == targetStart && nextNode == targetEnd) ||
						(currentNode == targetEnd && nextNode == targetStart)

					heap.Push(&pq, queueItem{
						cost:       current.cost + na.graph[currentNode][nextNode],
						path:       append(current.path, nextNode),
						usedTarget: current.usedTarget || isTargetEdge,
					})
				}
			}
		}
	}

	return paths
}

func (na *NetworkAnalyzer) findShortCycles(node int, maxLength int) []Path {
	if maxLength == 0 {
		maxLength = calculateMaxLength(na.numNodes)
	}

	paths := make([]Path, 0)
	visitedStates := make(map[uint64]float64)
	queue := []queueItem{{path: []int{node}}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		currentNode := current.path[len(current.path)-1]

		if len(current.path) > 2 && currentNode == node {
			paths = append(paths, Path{Nodes: current.path})
			continue
		}

		if len(current.path) < maxLength {
			stateKey := generateStateKey(current.path)
			if existingCost, exists := visitedStates[stateKey]; exists && current.cost >= existingCost {
				continue
			}
			visitedStates[stateKey] = current.cost

			for nextNode := 0; nextNode < na.numNodes; nextNode++ {
				if na.graph[currentNode][nextNode] <= 0 || containsNode(current.path, nextNode) {
					continue
				}

				queue = append(queue, queueItem{
					cost: current.cost + na.graph[currentNode][nextNode],
					path: append(current.path, nextNode),
				})
			}
		}
	}

	return paths
}

func (na *NetworkAnalyzer) findRelevantCycles() ([]Path, error) {
	targetPaths := na.findTargetContainingPaths(0)
	startCycles := na.findShortCycles(na.targetStart, 0)

	allPaths := append(targetPaths, startCycles...)
	uniquePaths := deduplicatePaths(allPaths)

	measuredPaths := make([]Path, 0, len(uniquePaths))
	wg := sync.WaitGroup{}
	var mu sync.Mutex
	for _, path := range uniquePaths {
		wg.Add(1)
		go func(path Path) {
			defer wg.Done()
			result, err := na.ppClient.Measure(path.Nodes)
			if err != nil {
				log.Printf("Skipping path %v due to measurement error: %v", path.Nodes, err)
				return
			}

			mu.Lock()
			defer mu.Unlock()
			measuredPaths = append(measuredPaths, Path{
				Nodes:    path.Nodes,
				Segments: generateSegments(path.Nodes),
				Cost:     float64(result.Duration.Microseconds()) / 1000.0,
			})
		}(path)
	}

	if len(measuredPaths) == 0 {
		return nil, fmt.Errorf("no measurable cycles found")
	}

	return measuredPaths[:min(len(measuredPaths), 100)], nil
}

func (na *NetworkAnalyzer) AnalyzeTargetPath(ctx context.Context, start, end int, solverClient pb.NetworkAnalyzerClient) (float64, error) {
	na.targetStart = start
	na.targetEnd = end
	na.solverClient = solverClient

	paths, err := na.findRelevantCycles()
	if err != nil {
		return 0, err
	}

	edgeMap, targetEdge := na.buildEdgeMap(paths)
	matrix, vector := na.buildMatrix(paths, edgeMap)

	response, err := na.solverClient.SolveMatrix(ctx, &pb.MatrixData{
		Matrix:     matrix,
		Vector:     vector,
		MatrixRows: int32(len(paths) + len(edgeMap)),
		MatrixCols: int32(len(edgeMap)),
		TargetIdx:  int32(edgeMap[targetEdge]),
	})

	if err != nil {
		return 0, fmt.Errorf("solver error: %w", err)
	}

	return response.Solution[edgeMap[targetEdge]], nil
}

// Helper functions
func generateSegments(nodes []int) [][2]int {
	segments := make([][2]int, len(nodes)-1)
	for i := 0; i < len(nodes)-1; i++ {
		segments[i] = [2]int{nodes[i], nodes[i+1]}
	}
	return segments
}

func containsNode(path []int, node int) bool {
	for _, n := range path {
		if n == node {
			return true
		}
	}
	return false
}

func calculateMaxLength(numNodes int) int {
	switch {
	case numNodes < 10:
		return numNodes + 3
	case numNodes < 24:
		return (numNodes / 2) + 1
	default:
		return 3
	}
}

func generateStateKey(path []int) uint64 {
	hash := uint64(0)
	for _, n := range path {
		hash = (hash << 5) ^ uint64(n)
	}
	return hash
}

func deduplicatePaths(paths []Path) []Path {
	seen := make(map[string]bool)
	unique := make([]Path, 0, len(paths))
	for _, p := range paths {
		key := fmt.Sprintf("%v", p.Nodes)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, p)
		}
	}
	sort.Slice(unique, func(i, j int) bool {
		return len(unique[i].Nodes) < len(unique[j].Nodes)
	})
	return unique
}

func (na *NetworkAnalyzer) buildEdgeMap(paths []Path) (map[[2]int]int, [2]int) {
	edgeSet := make(map[[2]int]bool)
	for _, path := range paths {
		for _, seg := range path.Segments {
			edgeSet[canonicalEdge(seg)] = true
		}
	}

	targetEdge := canonicalEdge([2]int{na.targetStart, na.targetEnd})
	edgeSet[targetEdge] = true

	edgeMap := make(map[[2]int]int)
	i := 0
	for edge := range edgeSet {
		edgeMap[edge] = i
		i++
	}
	return edgeMap, targetEdge
}

func (na *NetworkAnalyzer) buildMatrix(paths []Path, edgeMap map[[2]int]int) ([]float64, []float64) {
	numEdges := len(edgeMap)
	numPaths := len(paths)
	matrixSize := (numPaths + numEdges) * numEdges
	matrix := make([]float64, matrixSize)
	vector := make([]float64, numPaths+numEdges)

	for i, path := range paths {
		for _, seg := range path.Segments {
			idx := edgeMap[canonicalEdge(seg)]
			matrix[i*numEdges+idx] += 1
		}
		vector[i] = path.Cost
	}

	// Add regularization
	for i := 0; i < numEdges; i++ {
		matrix[(numPaths+i)*numEdges+i] = 1
	}
	return matrix, vector
}

func canonicalEdge(edge [2]int) [2]int {
	if edge[0] > edge[1] {
		return [2]int{edge[1], edge[0]}
	}
	return edge
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
