package path

import (
	stderr "errors"
	"fmt"

	"github.com/gonum/stat/combin"
	"github.com/gsmcwhirter/go-util/v7/errors"
)

type Finder struct {
	graph     [][]int
	graphTags [][]bool
}

func NewFinder(graph [][]int, graphTags [][]bool) *Finder {
	return &Finder{
		graph:     graph,
		graphTags: graphTags,
	}
}

var ErrNoRoute = errors.New("no route")

func hasZeros(vals []int, except int) bool {
	for i, v := range vals {
		if i == except {
			continue
		}

		if v == 0 {
			return true
		}
	}

	return false
}

func (f *Finder) hasTag(i, tag int) bool {
	return f.graphTags[i][tag]
}

func (f *Finder) hasAnyTag(i int, tags []int) bool {
	for _, tag := range tags {
		if f.hasTag(i, tag) {
			return true
		}
	}

	return false
}

func (f *Finder) FindShortestRoutes(start, end int, avoids, avoidTags, preferNotTags []int) ([][]int, error) {
	routes, err := f.findShortestRoutes(start, end, avoids, append(avoidTags, preferNotTags...))
	if err == nil {
		return routes, err
	}

	if !stderr.Is(err, ErrNoRoute) {
		return routes, err
	}

	numPrefer := len(preferNotTags)
	for i := 1; i <= numPrefer; i++ { //omit this many preferNotTags to try and find a route
		numLooser := combin.Binomial(numPrefer, numPrefer-i)
		looserRoutes := make([][][]int, 0, numLooser)
		looserLengths := make([]int, 0, numLooser)
		minLength := -1

		combos := combin.Combinations(numPrefer, numPrefer-i)
		avoidTagsPlus := make([]int, 0, len(avoidTags)+numPrefer-i)
		avoidTagsPlus = append(avoidTagsPlus, avoidTags...)

		for _, combo := range combos {
			for _, idx := range combo {
				avoidTagsPlus = append(avoidTagsPlus, preferNotTags[idx])
			}

			routes, err := f.findShortestRoutes(start, end, avoids, avoidTagsPlus)
			if stderr.Is(err, ErrNoRoute) {
				continue
			}

			if err != nil {
				return nil, err
			}

			looserRoutes = append(looserRoutes, routes)
			routeLength := len(routes[0])
			looserLengths = append(looserLengths, routeLength)
			if minLength == -1 || routeLength < minLength {
				minLength = routeLength
			}
		}

		if len(looserRoutes) == 0 {
			continue
		}

		retRoutes := make([][]int, 0, numLooser)
		for i, l := range looserLengths {
			if l == minLength {
				retRoutes = append(retRoutes, looserRoutes[i]...)
			}
		}

		return retRoutes, nil
	}

	return nil, errors.Wrap(ErrNoRoute, "could not find route")
}

func (f *Finder) FindAllShortestRoutes(starts []int, end int, avoids, avoidTags, preferNotTags []int) ([][]int, error) {
	allRoutes := make([][][]int, len(starts))
	routeLengths := make([]int, len(starts))
	minLength := -1
	routesCt := 0

	for i, start := range starts {
		routes, err := f.findShortestRoutes(start, end, avoids, append(avoidTags, preferNotTags...))
		if stderr.Is(err, ErrNoRoute) {
			continue
		}

		if err != nil {
			return nil, err
		}

		routesCt += len(routes)
		allRoutes[i] = routes
		routeLength := len(routes[0])
		routeLengths[i] = routeLength
		if minLength == -1 || routeLength < minLength {
			minLength = routeLength
		}
	}

	if minLength > 0 {
		minRoutes := make([][]int, 0, routesCt)

		for i, l := range routeLengths {
			if l == minLength {
				minRoutes = append(minRoutes, allRoutes[i]...)
			}
		}

		return minRoutes, nil
	}

	numPrefer := len(preferNotTags)
	for i := 1; i <= numPrefer; i++ { //omit this many preferNotTags to try and find a route
		numLooser := combin.Binomial(numPrefer, numPrefer-i)
		looserRoutes := make([][][]int, 0, numLooser*len(starts))
		looserLengths := make([]int, 0, numLooser*len(starts))
		minLength := -1

		combos := combin.Combinations(numPrefer, numPrefer-i)
		avoidTagsPlus := make([]int, 0, len(avoidTags)+numPrefer-i)
		avoidTagsPlus = append(avoidTagsPlus, avoidTags...)

		for _, combo := range combos {
			for _, idx := range combo {
				avoidTagsPlus = append(avoidTagsPlus, preferNotTags[idx])
			}

			for _, start := range starts {
				routes, err := f.findShortestRoutes(start, end, avoids, avoidTagsPlus)
				if stderr.Is(err, ErrNoRoute) {
					continue
				}

				if err != nil {
					return nil, err
				}

				looserRoutes = append(looserRoutes, routes)
				routeLength := len(routes[0])
				looserLengths = append(looserLengths, routeLength)
				if minLength == -1 || routeLength < minLength {
					minLength = routeLength
				}
			}
		}

		if len(looserRoutes) == 0 {
			continue
		}

		retRoutes := make([][]int, 0, numLooser)
		for i, l := range looserLengths {
			if l == minLength {
				retRoutes = append(retRoutes, looserRoutes[i]...)
			}
		}

		return retRoutes, nil
	}

	return nil, errors.Wrap(ErrNoRoute, "could not find route")
}

func (f *Finder) findShortestRoutes(start, end int, avoids, avoidTags []int) ([][]int, error) {
	distances := make([]int, len(f.graph))

	if start == end {
		return nil, errors.Wrap(ErrNoRoute, "start and end are identical")
	}

	avoidSet := map[int]bool{}
	for _, v := range avoids {
		avoidSet[v] = true
	}

	candidates := [][]int{{start}}
	found := false
	for {
		if found {
			// fmt.Printf("filter for accepted: %v\n", candidates)
			actualRoutes := make([][]int, 0, len(candidates))
			for _, c := range candidates {
				if c[len(c)-1] != end {
					// fmt.Printf("%v bad\n", c)
					continue
				}
				// fmt.Printf("%v ok\n", c)
				actualRoutes = append(actualRoutes, c)
			}
			return actualRoutes, nil
		}

		if !hasZeros(distances, start) || len(candidates) == 0 {
			return nil, errors.Wrap(ErrNoRoute, "route impossible 1")
		}

		// fmt.Printf("Candidates: %v\n", candidates)
		newCandidates := make([][]int, 0, len(candidates))

		for _, candidate := range candidates {
			curr := candidate[len(candidate)-1]
			// fmt.Printf("candidate %v, curr %v, neighbors %v\n", candidate, curr, f.graph[curr])

			for _, neighbor := range f.graph[curr] {
				if neighbor == start {
					// fmt.Println("back to start")
					continue
				}

				if neighbor == end {
					// fmt.Println("made it!")
					distances[neighbor] = len(candidate) - 1
					newCandidate := make([]int, 0, len(candidate)+1)
					newCandidate = append(newCandidate, candidate...)
					newCandidate = append(newCandidate, neighbor)
					newCandidates = append(newCandidates, newCandidate)
					found = true
					continue
				}

				if distances[neighbor] > 0 && distances[neighbor] < len(candidate)-1 { // backtracking
					// fmt.Println("already visited")
					continue
				}

				if avoidSet[neighbor] { //avoided
					// fmt.Println("avoided")
					continue
				}

				if f.hasAnyTag(neighbor, avoidTags) { // avoided
					// fmt.Println("tag avoided")
					continue
				}

				distances[neighbor] = len(candidate) - 1
				newCandidate := make([]int, 0, len(candidate)+1)
				newCandidate = append(newCandidate, candidate...)
				newCandidate = append(newCandidate, neighbor)
				// fmt.Printf("accepted: %v\n", newCandidate)
				newCandidates = append(newCandidates, newCandidate)
				// fmt.Printf("  new candidates: %v\n", newCandidates)
			}
		}

		candidates = newCandidates
	}
}

func (f *Finder) FindShortestRoutesToTag(start, endTag int, avoids, avoidTags, preferNotTags []int) ([][]int, error) {
	routes, err := f.findShortestRoutesToTag(start, endTag, avoids, append(avoidTags, preferNotTags...))
	if err == nil {
		return routes, err
	}

	if !stderr.Is(err, ErrNoRoute) {
		return routes, err
	}

	numPrefer := len(preferNotTags)
	for i := 1; i <= numPrefer; i++ { //omit this many preferNotTags to try and find a route
		numLooser := combin.Binomial(numPrefer, numPrefer-i)
		looserRoutes := make([][][]int, 0, numLooser)
		looserLengths := make([]int, 0, numLooser)
		minLength := -1

		combos := combin.Combinations(numPrefer, numPrefer-i)

		for _, combo := range combos {
			avoidTagsPlus := make([]int, 0, len(avoidTags)+numPrefer-i)
			avoidTagsPlus = append(avoidTagsPlus, avoidTags...)
			for _, idx := range combo {
				avoidTagsPlus = append(avoidTagsPlus, preferNotTags[idx])
			}

			routes, err := f.findShortestRoutesToTag(start, endTag, avoids, avoidTagsPlus)
			if stderr.Is(err, ErrNoRoute) {
				continue
			}

			if err != nil {
				return nil, err
			}

			looserRoutes = append(looserRoutes, routes)
			routeLength := len(routes[0])
			looserLengths = append(looserLengths, routeLength)
			if minLength == -1 || routeLength < minLength {
				minLength = routeLength
			}
		}

		if len(looserRoutes) == 0 {
			continue
		}

		retRoutes := make([][]int, 0, numLooser)
		for i, l := range looserLengths {
			if l == minLength {
				retRoutes = append(retRoutes, looserRoutes[i]...)
			}
		}

		return retRoutes, nil
	}

	return nil, errors.Wrap(ErrNoRoute, "could not find route")
}

func (f *Finder) FindAllShortestRoutesToTag(starts []int, endTag int, avoids, avoidTags, preferNotTags []int) ([][]int, error) {
	allRoutes := make([][][]int, len(starts))
	routeLengths := make([]int, len(starts))
	minLength := -1
	routesCt := 0

	for i, start := range starts {
		routes, err := f.findShortestRoutesToTag(start, endTag, avoids, append(avoidTags, preferNotTags...))
		if stderr.Is(err, ErrNoRoute) {
			continue
		}

		if err != nil {
			return nil, err
		}

		routesCt += len(routes)
		allRoutes[i] = routes
		routeLength := len(routes[0])
		routeLengths[i] = routeLength
		if minLength == -1 || routeLength < minLength {
			minLength = routeLength
		}
	}

	if minLength > 0 {
		minRoutes := make([][]int, 0, routesCt)

		for i, l := range routeLengths {
			if l == minLength {
				minRoutes = append(minRoutes, allRoutes[i]...)
			}
		}

		return minRoutes, nil
	}

	numPrefer := len(preferNotTags)
	for i := 1; i <= numPrefer; i++ { //omit this many preferNotTags to try and find a route
		numLooser := combin.Binomial(numPrefer, numPrefer-i)
		looserRoutes := make([][][]int, 0, numLooser*len(starts))
		looserLengths := make([]int, 0, numLooser*len(starts))
		minLength := -1

		combos := combin.Combinations(numPrefer, numPrefer-i)

		fmt.Printf("avoid: %v\n", avoidTags)
		fmt.Printf("prefer not: %v\n", preferNotTags)

		for _, combo := range combos {
			avoidTagsPlus := make([]int, 0, len(avoidTags)+numPrefer-i)
			avoidTagsPlus = append(avoidTagsPlus, avoidTags...)
			for _, idx := range combo {
				avoidTagsPlus = append(avoidTagsPlus, preferNotTags[idx])
			}

			fmt.Printf("combo: %v\n", combo)
			fmt.Printf("full avoids: %v\n", avoidTagsPlus)
			fmt.Printf("minLength before: %v\n", minLength)

			for _, start := range starts {
				routes, err := f.findShortestRoutesToTag(start, endTag, avoids, avoidTagsPlus)
				if stderr.Is(err, ErrNoRoute) {
					continue
				}

				if err != nil {
					return nil, err
				}

				looserRoutes = append(looserRoutes, routes)
				routeLength := len(routes[0])
				looserLengths = append(looserLengths, routeLength)
				if minLength == -1 || routeLength < minLength {
					minLength = routeLength
				}
			}

			fmt.Printf("minLength after: %v\n", minLength)
		}

		if len(looserRoutes) == 0 {
			continue
		}

		retRoutes := make([][]int, 0, numLooser)
		for i, l := range looserLengths {
			if l == minLength {
				retRoutes = append(retRoutes, looserRoutes[i]...)
			}
		}

		return retRoutes, nil
	}

	return nil, errors.Wrap(ErrNoRoute, "could not find route")
}

func (f *Finder) findShortestRoutesToTag(start, endTag int, avoids, avoidTags []int) ([][]int, error) {
	distances := make([]int, len(f.graph))

	if f.hasTag(start, endTag) {
		return nil, errors.Wrap(ErrNoRoute, "start has the tag")
	}

	for _, atag := range avoidTags {
		if atag == endTag {
			return nil, errors.Wrap(ErrNoRoute, "end tag is avoided")
		}
	}

	avoidSet := map[int]bool{}
	for _, v := range avoids {
		avoidSet[v] = true
	}

	candidates := [][]int{{start}}
	found := false
	for {
		if found {
			// fmt.Printf("filter for accepted: %v\n", candidates)
			actualRoutes := make([][]int, 0, len(candidates))
			for _, c := range candidates {
				if !f.hasTag(c[len(c)-1], endTag) {
					// fmt.Printf("%v bad\n", c)
					continue
				}
				// fmt.Printf("%v ok\n", c)
				actualRoutes = append(actualRoutes, c)
			}
			return actualRoutes, nil
		}

		if !hasZeros(distances, start) || len(candidates) == 0 {
			return nil, errors.Wrap(ErrNoRoute, "route impossible 1")
		}

		// fmt.Printf("Candidates: %v\n", candidates)
		newCandidates := make([][]int, 0, len(candidates))

		for _, candidate := range candidates {
			curr := candidate[len(candidate)-1]
			// fmt.Printf("candidate %v, curr %v, neighbors %v\n", candidate, curr, f.graph[curr])

			for _, neighbor := range f.graph[curr] {
				if neighbor == start {
					// fmt.Println("back to start")
					continue
				}

				if distances[neighbor] > 0 && distances[neighbor] < len(candidate)-1 { // backtracking
					// fmt.Println("already visited")
					continue
				}

				if avoidSet[neighbor] { //avoided
					// fmt.Println("avoided")
					continue
				}

				if f.hasAnyTag(neighbor, avoidTags) { // avoided
					// fmt.Println("tag avoided")
					continue
				}

				if f.hasTag(neighbor, endTag) {
					// fmt.Println("made it!")
					distances[neighbor] = len(candidate) - 1
					newCandidate := make([]int, 0, len(candidate)+1)
					newCandidate = append(newCandidate, candidate...)
					newCandidate = append(newCandidate, neighbor)
					newCandidates = append(newCandidates, newCandidate)
					found = true
					continue
				}

				distances[neighbor] = len(candidate) - 1
				newCandidate := make([]int, 0, len(candidate)+1)
				newCandidate = append(newCandidate, candidate...)
				newCandidate = append(newCandidate, neighbor)
				// fmt.Printf("accepted: %v\n", newCandidate)
				newCandidates = append(newCandidates, newCandidate)
				// fmt.Printf("  new candidates: %v\n", newCandidates)
			}
		}

		candidates = newCandidates
	}
}
