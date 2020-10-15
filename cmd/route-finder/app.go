package main

import (
	"fmt"
	"os"

	"github.com/gsmcwhirter/eve-route-finder/pkg/path"
	"github.com/gsmcwhirter/eve-route-finder/pkg/system"
	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type App struct {
	systemDataFile string

	fromSystems   []string
	toSystem      string
	toTag         string
	avoidSystems  []string
	avoidTags     []string
	preferNotTags []string

	rawDataContents DataContents

	tags           map[string]int
	reverseTags    map[int]string
	systems        map[string]int
	reverseSystems map[int]string

	systemSec map[int]string

	pathfinder *path.Finder
}

func NewApp() *App {
	return &App{
		tags:           map[string]int{},
		reverseTags:    map[int]string{},
		systems:        map[string]int{},
		reverseSystems: map[int]string{},
		systemSec:      map[int]string{},
	}
}

type DataContents struct {
	SystemData []system.Data
}

func (a *App) Run() error {
	if a.toSystem != "" && a.toTag != "" {
		return errors.New("cannot provide both target system and tag")
	}

	if a.toSystem == "" && a.toTag == "" {
		return errors.New("must provide either target system or tag")
	}

	if err := a.loadSystemData(); err != nil {
		return errors.Wrap(err, "could not load system data")
	}

	if err := a.preparePathFinder(); err != nil {
		return errors.Wrap(err, "could not populate data")
	}

	fromIDs := make([]int, len(a.fromSystems))
	for i, sname := range a.fromSystems {
		fromIDs[i] = a.systems[sname]
	}
	avoidIDs := make([]int, len(a.avoidSystems))
	for i, aname := range a.avoidSystems {
		avoidIDs[i] = a.systems[aname]
	}
	avoidTagIDs := make([]int, len(a.avoidTags))
	for i, tname := range a.avoidTags {
		avoidTagIDs[i] = a.tags[tname]
	}
	preferNotTagIDs := make([]int, len(a.preferNotTags))
	for i, tname := range a.preferNotTags {
		preferNotTagIDs[i] = a.tags[tname]
	}

	var routes [][]int
	var err error

	switch {
	case a.toSystem != "":
		fmt.Printf("from: %v, to: %s, avoid: %v, avoid tags: %v, soft avoid tags: %v\n", a.fromSystems, a.toSystem, a.avoidSystems, a.avoidTags, a.preferNotTags)
		toID := a.systems[a.toSystem]
		if len(fromIDs) == 1 {
			routes, err = a.pathfinder.FindShortestRoutes(fromIDs[0], toID, avoidIDs, avoidTagIDs, preferNotTagIDs)
		} else {
			routes, err = a.pathfinder.FindAllShortestRoutes(fromIDs, toID, avoidIDs, avoidTagIDs, preferNotTagIDs)
		}
	case a.toTag != "":
		fmt.Printf("from: %v, to tag: %s, avoid: %v, avoid tags: %v, soft avoid tags: %v\n", a.fromSystems, a.toSystem, a.avoidSystems, a.avoidTags, a.preferNotTags)
		toID := a.tags[a.toTag]
		if len(fromIDs) == 1 {
			routes, err = a.pathfinder.FindShortestRoutesToTag(fromIDs[0], toID, avoidIDs, avoidTagIDs, preferNotTagIDs)
		} else {
			routes, err = a.pathfinder.FindAllShortestRoutesToTag(fromIDs, toID, avoidIDs, avoidTagIDs, preferNotTagIDs)
		}
	}

	if err != nil {
		return errors.Wrap(err, "could not find a viable route")
	}

	for _, route := range routes {
		fmt.Println(a.GetNiceRoute(route))
	}

	return nil
}

func (a *App) GetNiceRoute(route []int) []string {
	niceRoute := make([]string, len(route))
	for j, id := range route {
		niceRoute[j] = fmt.Sprintf("%s [%s]", a.reverseSystems[id], a.systemSec[id][0:1])
	}

	return niceRoute
}

func (a *App) loadSystemData() error {
	f, err := os.Open(a.systemDataFile)
	if err != nil {
		return errors.Wrap(err, "could not open system data")
	}
	defer deferutil.CheckDefer(f.Close)

	dataContents := DataContents{}

	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&dataContents); err != nil {
		return errors.Wrap(err, "could not yaml decode system data")
	}

	a.rawDataContents = dataContents

	return nil
}

func (a *App) preparePathFinder() error {
	numSystems := len(a.rawDataContents.SystemData)
	graph := make([][]int, numSystems)
	graphTags := make([][]bool, numSystems)

	// first pass
	for _, sd := range a.rawDataContents.SystemData {
		// populate system name lookups
		a.systems[sd.Name] = sd.ID
		a.reverseSystems[sd.ID] = sd.Name
		a.systemSec[sd.ID] = sd.SecStatus

		// populate tag name lookups
		for _, t := range sd.Tags {
			var tid int
			var ok bool
			if _, ok = a.tags[t]; !ok {
				tid = len(a.tags)
				a.tags[t] = tid
				a.reverseTags[tid] = t
			}
		}
	}

	// second pass
	for _, sd := range a.rawDataContents.SystemData {
		graph[sd.ID] = make([]int, len(sd.Destinations))
		graphTags[sd.ID] = make([]bool, len(a.tags))

		// set destinations
		for i, d := range sd.Destinations {
			graph[sd.ID][i] = a.systems[d]
		}

		// set tags
		for _, t := range sd.Tags {
			tid := a.tags[t]
			graphTags[sd.ID][tid] = true
		}
	}

	a.pathfinder = path.NewFinder(graph, graphTags)

	return nil
}
