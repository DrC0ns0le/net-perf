package finder

import (
	"container/heap"
	"context"
	"fmt"
	"math"

	"github.com/DrC0ns0le/net-perf/internal/route/cost"
)

type Graph struct {
	size   int
	matrix [][]float64
	prev   []int
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
		if path.source > maxNode {
			maxNode = path.source
		}
		if path.target > maxNode {
			maxNode = path.target
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
		a, b := path.source, path.target
		if a > b {
			a, b = b, a
		}
		cost, err := cost.GetPathCost(ctx, a+64512, b+64512)
		if err != nil {
			return nil, err
		}
		g.matrix[path.source][path.target] = cost
	}

	return g, nil
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
