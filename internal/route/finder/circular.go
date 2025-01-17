package finder

import (
	"container/heap"
	"fmt"
	"math"
	"sort"
)

// CircularPath represents a single circular path with its length
type CircularPath struct {
	nodes        []int
	length       int
	foundSubPath bool         // Track if we've used the required direct edge
	visited      map[int]bool // Track visited nodes to prevent duplicates
}

// PathHeap implements heap.Interface for CircularPath
type PathHeap []CircularPath

func (h PathHeap) Len() int           { return len(h) }
func (h PathHeap) Less(i, j int) bool { return h[i].length < h[j].length }
func (h PathHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *PathHeap) Push(x interface{}) {
	*h = append(*h, x.(CircularPath))
}

func (h *PathHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// FindNCircularPathsWithDirectEdge finds N unique circular paths that must use a specific direct edge
// where each node (except start/end) appears at most once
func (g *Graph) FindNCircularPathsWithDirectEdge(edgeStart, edgeEnd, n int) ([]CircularPath, error) {
	if edgeStart >= g.size || edgeEnd >= g.size {
		return nil, fmt.Errorf("invalid edge nodes")
	}

	// Verify that the required edge exists
	if g.matrix[edgeStart][edgeEnd] == math.Inf(1) {
		return nil, fmt.Errorf("specified edge does not exist in graph")
	}

	result := make([]CircularPath, 0, n)
	uniquePaths := make(map[string]bool)

	// Try each possible start node
	for startNode := 0; startNode < g.size; startNode++ {
		candidates := &PathHeap{}
		heap.Init(candidates)

		// Initialize visited nodes map for starting path
		visited := make(map[int]bool)
		visited[startNode] = true

		// Start the initial path
		heap.Push(candidates, CircularPath{
			nodes:        []int{startNode},
			length:       0,
			foundSubPath: false,
			visited:      visited,
		})

		for candidates.Len() > 0 && len(result) < n {
			current := heap.Pop(candidates).(CircularPath)
			lastNode := current.nodes[len(current.nodes)-1]

			// Check if we've completed a valid circular path
			if lastNode == startNode && len(current.nodes) > 1 && current.foundSubPath {
				pathKey := fmt.Sprintf("%v", current.nodes)
				if !uniquePaths[pathKey] {
					result = append(result, current)
					uniquePaths[pathKey] = true
					continue
				}
			}

			// Limit path length to prevent excessive searching
			if len(current.nodes) > g.size*2 {
				continue
			}

			// Explore neighbors
			for neighbor := 0; neighbor < g.size; neighbor++ {
				// Skip if edge doesn't exist
				if g.matrix[lastNode][neighbor] == math.Inf(1) {
					continue
				}

				// Skip if this is the reverse of our required edge
				if lastNode == edgeEnd && neighbor == edgeStart {
					continue
				}

				// Skip if we've already visited this node (unless it's the starting node and we've used required edge)
				if current.visited[neighbor] && !(neighbor == startNode && current.foundSubPath) {
					continue
				}

				// Check if this step uses the required edge
				usesRequiredEdge := lastNode == edgeStart && neighbor == edgeEnd
				newFoundSubPath := current.foundSubPath || usesRequiredEdge

				// Create new visited map
				newVisited := make(map[int]bool)
				for k, v := range current.visited {
					newVisited[k] = v
				}
				newVisited[neighbor] = true

				// Create new path
				newNodes := make([]int, len(current.nodes))
				copy(newNodes, current.nodes)
				newNodes = append(newNodes, neighbor)

				// Skip if we've seen this exact path before
				pathKey := fmt.Sprintf("%v", newNodes)
				if uniquePaths[pathKey] {
					continue
				}

				// Add new path to candidates
				heap.Push(candidates, CircularPath{
					nodes:        newNodes,
					length:       current.length + 1,
					foundSubPath: newFoundSubPath,
					visited:      newVisited,
				})
			}
		}

		if len(result) >= n {
			break
		}
	}

	// Sort results by length
	sort.Slice(result, func(i, j int) bool {
		return result[i].length < result[j].length
	})

	return result, nil
}

// Helper function to print a circular path
func (p CircularPath) String() string {
	return fmt.Sprintf("Length: %d edges, Path: %v", p.length, p.nodes)
}

func (c *CircularPath) Path() []int { return c.nodes }
