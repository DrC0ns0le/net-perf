package finder

import (
	"container/heap"
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/DrC0ns0le/net-perf/internal/route/cost"
)

type Graph struct {
	size   int
	matrix [][]float64
	prev   []int
	mu     sync.RWMutex
}

// NewGraph creates a new graph with adjacency matrix representation
func NewGraph(ctx context.Context) (*Graph, error) {
	// Get all network paths
	paths, err := GetAllPaths(ctx)
	if err != nil {
		return nil, err
	}

	// Find the maximum node ID to determine matrix size
	maxNode := 0
	for _, path := range paths {
		if path.Source > maxNode {
			maxNode = path.Source
		}
		if path.Target > maxNode {
			maxNode = path.Target
		}
	}

	// Create graph with size + 1 (as nodes might be zero-indexed)
	size := maxNode + 1
	g := &Graph{
		size:   size,
		matrix: make([][]float64, size),
		prev:   make([]int, size),
	}

	// Initialize matrix with infinity
	for i := range g.matrix {
		g.matrix[i] = make([]float64, size)
		for j := range g.matrix[i] {
			g.matrix[i][j] = math.Inf(1)
		}
	}

	// Fill matrix with path costs
	for _, path := range paths {
		cost, err := cost.GetPathCost(ctx, path.Source+64512, path.Target+64512)
		if err != nil {
			return nil, err
		}
		g.matrix[path.Source][path.Target] = cost
	}

	return g, nil
}

func NewGraphFromMatrix(matrix [][]float64) *Graph {
	return &Graph{
		size:   len(matrix),
		matrix: matrix,
		prev:   make([]int, len(matrix)),
	}
}

func (g *Graph) GetMatrix() [][]float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.matrix
}

// RefreshWeights refreshes all the weights in the graph
func (g *Graph) RefreshWeights(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	paths, err := GetAllPaths(ctx)
	if err != nil {
		return fmt.Errorf("failed to get network paths: %w", err)
	}

	for _, path := range paths {
		newCost, err := cost.GetPathCost(ctx, path.Source+64512, path.Target+64512)
		if err != nil {
			return fmt.Errorf("failed to get path cost for %d -> %d: %w", path.Source, path.Target, err)
		}
		if newCost == 0 {
			return fmt.Errorf("unexpected cost of 0 for %d -> %d", path.Source, path.Target)
		}

		if math.IsInf(newCost, 1) {
			return fmt.Errorf("unexpected infinity cost for %d -> %d", path.Source, path.Target)
		}
		g.matrix[path.Source][path.Target] = newCost
	}

	return nil
}

// SetDirectedWeights sets the weight of a directed edge
func (g *Graph) SetDirectedWeights(src, dst int, weight float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.matrix[src][dst] = weight
}

// SetUndirectedWeights sets the weight of an undirected edge
func (g *Graph) SetUndirectedWeights(src, dst int, weight float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.matrix[src][dst] = weight
	g.matrix[dst][src] = weight
}

func (g *Graph) String() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return fmt.Sprintf("Graph:\nSize: %d\nMatrix: %v\n", g.size, g.matrix)
}

// Dijkstra implements Dijkstra's shortest path algorithm using a priority queue
func (g *Graph) Dijkstra(start int) []float64 {
	dist := make([]float64, g.size)
	for i := range dist {
		dist[i] = math.Inf(1)
		g.prev[i] = -1
	}
	dist[start] = 0

	// Initialize priority queue
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	// Add starting node
	startRoute := &Route{
		node:     start,
		distance: 0,
	}
	heap.Push(&pq, startRoute)

	// Process nodes
	for pq.Len() > 0 {
		// Get the node with minimum distance
		route := heap.Pop(&pq).(*Route)
		u := route.node

		// If we've found a longer path, skip
		if route.distance > dist[u] {
			continue
		}

		// Check all neighbors of u
		for v := 0; v < g.size; v++ {
			if g.matrix[u][v] != math.Inf(1) {
				newDist := dist[u] + g.matrix[u][v]

				// If we found a shorter path to v
				if newDist < dist[v] {
					dist[v] = newDist
					g.prev[v] = u

					// Add to priority queue
					heap.Push(&pq, &Route{
						node:     v,
						distance: newDist,
					})
				}
			}
		}
	}

	return dist
}

// GetShortestPath returns the shortest path from start to end
func (g *Graph) GetShortestPath(start, end int) ([]int, float64, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	dist := g.Dijkstra(start)

	if dist[end] == math.Inf(1) {
		return nil, math.Inf(1), fmt.Errorf("no path exists") // No path exists
	}

	// Reconstruct path
	path := []int{}
	for curr := end; curr != -1; curr = g.prev[curr] {
		path = append([]int{curr}, path...)
	}

	return path, dist[end], nil
}
