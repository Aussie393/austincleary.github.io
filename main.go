package main

import (
	"container/heap"
	"fmt"
	"math"
	"os"
	"sort"
)

// ---------- Graph types ----------
type Edge struct {
	To     string
	Weight int
}

type Graph map[string][]Edge

// ---------- Priority Queue for Dijkstra ----------
type item struct {
	node string
	dist int
	idx  int // index in the heap
}

type priorityQueue []*item

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].dist < pq[j].dist }
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].idx = i
	pq[j].idx = j
}

func (pq *priorityQueue) Push(x interface{}) {
	it := x.(*item)
	it.idx = len(*pq)
	*pq = append(*pq, it)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	it := old[n-1]
	it.idx = -1
	*pq = old[:n-1]
	return it
}

func (pq *priorityQueue) update(it *item, dist int) {
	it.dist = dist
	heap.Fix(pq, it.idx)
}

// ---------- Dijkstra ----------
type DijkstraResult struct {
	Found         bool
	Distance      int
	Prev          map[string]string
	VisitedOrder  []string
	DistanceTable map[string]int
	Steps         []DijkstraStep
}

type DijkstraStep struct {
	Step         int
	Action       string
	CurrentNode  string
	Distance     int
	RelaxedNode  string
	NewDistance  int
	DistSnapshot map[string]int
}

func Dijkstra(g Graph, src, dst string) DijkstraResult {
	dist := make(map[string]int)
	prev := make(map[string]string)
	visited := make(map[string]bool)
	visitedOrder := []string{}
	steps := []DijkstraStep{}
	stepNum := 0

	// initialize
	for u := range g {
		dist[u] = math.MaxInt / 4
	}
	// It's possible some nodes only exist as targets; bring them in
	for _, edges := range g {
		for _, e := range edges {
			if _, ok := dist[e.To]; !ok {
				dist[e.To] = math.MaxInt / 4
			}
		}
	}
	if _, ok := dist[src]; !ok {
		dist[src] = 0
	}
	dist[src] = 0

	pq := &priorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &item{node: src, dist: 0})

	// Record initialization
	stepNum++
	snapshot := copyDistMap(dist)
	steps = append(steps, DijkstraStep{
		Step:         stepNum,
		Action:       fmt.Sprintf("Initialize: distance to %s = 0, all others = ∞", src),
		DistSnapshot: snapshot,
	})

	for pq.Len() > 0 {
		it := heap.Pop(pq).(*item)
		u := it.node
		if visited[u] {
			continue
		}
		visited[u] = true
		visitedOrder = append(visitedOrder, u)

		stepNum++
		snapshot := copyDistMap(dist)
		steps = append(steps, DijkstraStep{
			Step:         stepNum,
			Action:       fmt.Sprintf("Visit node %s with distance %d", u, dist[u]),
			CurrentNode:  u,
			Distance:     dist[u],
			DistSnapshot: snapshot,
		})

		if u == dst {
			// early exit: we popped the target with its final shortest distance
			return DijkstraResult{
				Found:         true,
				Distance:      dist[dst],
				Prev:          prev,
				VisitedOrder:  visitedOrder,
				DistanceTable: dist,
				Steps:         steps,
			}
		}

		for _, e := range g[u] {
			if visited[e.To] {
				continue
			}
			alt := dist[u] + e.Weight
			if alt < dist[e.To] {
				dist[e.To] = alt
				prev[e.To] = u
				heap.Push(pq, &item{node: e.To, dist: alt})

				stepNum++
				snapshot := copyDistMap(dist)
				steps = append(steps, DijkstraStep{
					Step:         stepNum,
					Action:       fmt.Sprintf("Relax edge %s → %s", u, e.To),
					CurrentNode:  u,
					RelaxedNode:  e.To,
					NewDistance:  alt,
					DistSnapshot: snapshot,
				})
			}
		}
	}

	// Not reachable
	return DijkstraResult{
		Found:         false,
		Distance:      math.MaxInt / 4,
		Prev:          prev,
		VisitedOrder:  visitedOrder,
		DistanceTable: dist,
		Steps:         steps,
	}
}

func copyDistMap(m map[string]int) map[string]int {
	copy := make(map[string]int)
	for k, v := range m {
		copy[k] = v
	}
	return copy
}

func reconstructPath(prev map[string]string, target string) []string {
	path := []string{}
	cur := target
	for {
		path = append([]string{cur}, path...)
		p, ok := prev[cur]
		if !ok {
			break
		}
		cur = p
	}
	return path
}

// ---------- Demo graph with a long cheap dead-end ----------
// This creates a directed graph:
//
//	S -> d1 -> d2 -> ... -> dN  (each weight 1; very long dead-end; last dN has no outgoing)
//	S -> X (12), X -> T (12)    (target reachable on a different branch; total 24)
//	S -> Y (13), Y -> T (13)    (slightly worse alternative; total 26)
//
// The dead-end edges have the *least* per-edge weight, but the target is NOT on that path.
type Point struct{ X, Y float64 }

func buildGraph(deadEndLen int) (Graph, map[string]Point) {
	g := make(Graph)

	// Node positions for drawing (SVG)
	pos := make(map[string]Point)

	// Place S
	pos["S"] = Point{X: 60, Y: 160}

	// Build the long dead-end to the right of S
	prev := "S"
	for i := 1; i <= deadEndLen; i++ {
		di := fmt.Sprintf("d%d", i)
		g[prev] = append(g[prev], Edge{To: di, Weight: 1})
		// layout horizontally
		pos[di] = Point{X: 60 + float64(i)*44, Y: 160}
		prev = di
	}
	// no outgoing edges from the last dead-end node 'prev' (makes it a dead end)

	// Build the target branch downward (larger per-edge weights)
	pos["X"] = Point{X: 60, Y: 260}
	pos["Y"] = Point{X: 60, Y: 360}
	pos["T"] = Point{X: 240, Y: 260}

	g["S"] = append(g["S"], Edge{To: "X", Weight: 12})
	g["X"] = append(g["X"], Edge{To: "T", Weight: 12})

	g["S"] = append(g["S"], Edge{To: "Y", Weight: 13})
	g["Y"] = append(g["Y"], Edge{To: "T", Weight: 13})

	// Ensure all nodes appear in the graph map
	for n := range pos {
		if _, ok := g[n]; !ok {
			g[n] = []Edge{}
		}
	}
	return g, pos
}

// ---------- SVG generation ----------
func writeSVG(filename string, g Graph, pos map[string]Point, shortestPath []string) error {
	// build edge set for highlighting shortest path
	pathEdges := make(map[[2]string]bool)
	for i := 0; i+1 < len(shortestPath); i++ {
		pathEdges[[2]string{shortestPath[i], shortestPath[i+1]}] = true
	}

	// bounds
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -1.0, -1.0
	for _, p := range pos {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	margin := 60.0
	w := maxX - minX + 2*margin
	h := maxY - minY + 2*margin

	shift := func(p Point) Point {
		return Point{X: p.X - minX + margin, Y: p.Y - minY + margin}
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	radius := 16.0
	arrowLen := 10.0
	arrowWidth := 6.0

	write := func(s string) {
		_, _ = f.WriteString(s)
	}

	write(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f">`, w, h, w, h))
	write("\n<style>\n")
	write(`.node { fill: #ffffff; stroke: #333; stroke-width: 1.5px; }` + "\n")
	write(`.node.path { fill: #ffe3e3; }` + "\n")
	write(`.edge { stroke: #888; stroke-width: 1.5px; }` + "\n")
	write(`.edge.path { stroke: #d33; stroke-width: 3px; }` + "\n")
	write(`.label { font: 12px sans-serif; fill: #222; }` + "\n")
	write(`.wlabel { font: 11px sans-serif; fill: #555; }` + "\n")
	write("</style>\n")

	// draw edges first
	for u, edges := range g {
		pu := shift(pos[u])
		for _, e := range edges {
			pv := shift(pos[e.To])

			// compute shortened endpoints (so arrows stop at node boundary)
			dx := pv.X - pu.X
			dy := pv.Y - pu.Y
			L := math.Hypot(dx, dy)
			if L == 0 {
				continue
			}
			ux, uy := dx/L, dy/L
			start := Point{X: pu.X + ux*radius, Y: pu.Y + uy*radius}
			end := Point{X: pv.X - ux*radius, Y: pv.Y - uy*radius}

			cls := "edge"
			if pathEdges[[2]string{u, e.To}] {
				cls = "edge path"
			}

			// edge line
			write(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" class="%s"/>`+"\n",
				start.X, start.Y, end.X, end.Y, cls))

			// arrowhead at end
			base := Point{X: end.X - ux*arrowLen, Y: end.Y - uy*arrowLen}
			nx, ny := -uy, ux // perpendicular
			p1 := end
			p2 := Point{X: base.X + nx*(arrowWidth/2), Y: base.Y + ny*(arrowWidth/2)}
			p3 := Point{X: base.X - nx*(arrowWidth/2), Y: base.Y - ny*(arrowWidth/2)}
			write(fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" />`+"\n",
				p1.X, p1.Y, p2.X, p2.Y, p3.X, p3.Y,
				map[bool]string{true: "#d33", false: "#888"}[cls == "edge path"]))

			// weight label at midpoint
			mx := (start.X + end.X) / 2
			my := (start.Y + end.Y) / 2
			write(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="middle" class="wlabel" dy="-4">%d</text>`+"\n",
				mx, my, e.Weight))
		}
	}

	// draw nodes
	// figure out which nodes are on the shortest path
	onPath := map[string]bool{}
	for _, n := range shortestPath {
		onPath[n] = true
	}

	for n, p := range pos {
		pp := shift(p)
		cls := "node"
		if onPath[n] {
			cls = "node path"
		}
		write(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="%.1f" class="%s"/>`+"\n", pp.X, pp.Y, radius, cls))
		write(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="middle" class="label" dy="4">%s</text>`+"\n",
			pp.X, pp.Y, n))
	}

	write("</svg>\n")
	return nil
}

func main() {
	// You can tweak these to experiment:
	deadEndLen := 25 // length of the very long dead-end (d1 -> d2 -> ... -> d25)
	src := "S"
	dst := "T"

	g, pos := buildGraph(deadEndLen)

	// Run Dijkstra
	res := Dijkstra(g, src, dst)

	fmt.Println("=== Dijkstra run ===")
	fmt.Printf("Source: %s, Target: %s\n\n", src, dst)

	// Display all steps
	fmt.Println("--- ALGORITHM STEPS ---\n")
	for _, step := range res.Steps {
		fmt.Printf("Step %d: %s\n", step.Step, step.Action)
		if step.CurrentNode != "" {
			fmt.Printf("  Node: %s\n", step.CurrentNode)
		}
		if step.RelaxedNode != "" {
			fmt.Printf("  Updated %s: → %d\n", step.RelaxedNode, step.NewDistance)
		}
		fmt.Printf("  Distances: ")
		// Print in sorted order for clarity
		var nodes []string
		for n := range step.DistSnapshot {
			nodes = append(nodes, n)
		}
		sort.Strings(nodes)
		for i, n := range nodes {
			if i > 0 {
				fmt.Printf(", ")
			}
			d := step.DistSnapshot[n]
			if d == math.MaxInt/4 {
				fmt.Printf("%s:∞", n)
			} else {
				fmt.Printf("%s:%d", n, d)
			}
		}
		fmt.Println("\n")
	}

	// Display final results
	fmt.Println("\n=== FINAL RESULT ===\n")
	fmt.Printf("Visited order: %v\n", res.VisitedOrder)
	if res.Found {
		path := reconstructPath(res.Prev, dst)
		fmt.Printf("Shortest distance: %d\n", res.Distance)
		fmt.Printf("Shortest path: %v\n\n", path)

		// Write an SVG of the graph highlighting the shortest path
		if err := writeSVG("graph.svg", g, pos, path); err != nil {
			fmt.Printf("Error writing SVG: %v\n", err)
		} else {
			fmt.Println("Graph image written to graph.svg (shortest path highlighted in red).")
		}
	} else {
		fmt.Println("No path found to the target.")
		// Still write the SVG (without a highlighted path)
		if err := writeSVG("graph.svg", g, pos, []string{}); err != nil {
			fmt.Printf("Error writing SVG: %v\n", err)
		} else {
			fmt.Println("Graph image written to graph.svg.")
		}
	}

	// Helpful note to observe behavior:
	fmt.Println("\nNotes:")
	fmt.Println("- The dead-end consists of many edges with weight 1 (very cheap) and the target is NOT on that chain.")
	fmt.Println("- Dijkstra explores nodes in order of increasing cumulative distance from S.")
	fmt.Println("- Because dead-end edges are so cheap, you'll see many d1, d2, ... visited early.")
	fmt.Println("- Despite that, Dijkstra remains correct and will return the shortest path to T if one exists (it does here).")
}
