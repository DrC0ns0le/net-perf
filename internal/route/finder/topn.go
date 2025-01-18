package finder

import (
	"container/heap"
	"fmt"
	"math"
)

// PathResult represents a single path result with its total distance
type PathResult struct {
	Path     []int
	Distance float64
}

// PathQueue is a priority queue for paths
type PathQueue []*PathResult

func (pq PathQueue) Len() int { return len(pq) }

func (pq PathQueue) Less(i, j int) bool {
	return pq[i].Distance < pq[j].Distance
}

func (pq PathQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PathQueue) Push(x interface{}) {
	item := x.(*PathResult)
	*pq = append(*pq, item)
}

func (pq *PathQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

// GetTopNShortestPaths returns the N shortest paths from start to end
func (g *Graph) GetTopNShortestPaths(start, end int, n int) ([]PathResult, error) {
	if n <= 0 {
		return nil, fmt.Errorf("n must be positive")
	}

	results := make([]PathResult, 0, n)
	visited := make(map[string]bool)

	// Initialize priority queue for paths
	pq := make(PathQueue, 0)
	heap.Init(&pq)

	// Add starting path
	startPath := &PathResult{
		Path:     []int{start},
		Distance: 0,
	}
	heap.Push(&pq, startPath)

	for pq.Len() > 0 && len(results) < n {
		current := heap.Pop(&pq).(*PathResult)
		lastNode := current.Path[len(current.Path)-1]

		// If we've reached the end node, add this path to results
		if lastNode == end {
			results = append(results, *current)
			continue
		}

		// Explore neighbors
		for v := 0; v < g.size; v++ {
			if g.matrix[lastNode][v] != math.Inf(1) {
				// Check if this would create a cycle
				if containsNode(current.Path, v) {
					continue
				}

				// Create new path
				newPath := make([]int, len(current.Path))
				copy(newPath, current.Path)
				newPath = append(newPath, v)

				// Calculate new distance
				newDistance := current.Distance + g.matrix[lastNode][v]

				// Create path key for visited check
				pathKey := fmt.Sprintf("%v", newPath)
				if !visited[pathKey] {
					visited[pathKey] = true
					heap.Push(&pq, &PathResult{
						Path:     newPath,
						Distance: newDistance,
					})
				}
			}
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no paths found")
	}

	return results, nil
}

// containsNode checks if a path contains a specific node
func containsNode(path []int, node int) bool {
	for _, n := range path {
		if n == node {
			return true
		}
	}
	return false
}
