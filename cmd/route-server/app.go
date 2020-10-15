package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gsmcwhirter/eve-route-finder/pkg/path"
	"github.com/gsmcwhirter/eve-route-finder/pkg/system"
	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type App struct {
	systemDataFile string
	listen         string

	rawDataContents DataContents

	tags              map[string]int
	reverseTags       map[int]string
	systems           map[string]int
	reverseSystems    map[int]string
	reverseSystemData map[int]system.Data

	systemSec map[int]string

	pathfinder *path.Finder
}

func NewApp() *App {
	return &App{
		tags:              map[string]int{},
		reverseTags:       map[int]string{},
		systems:           map[string]int{},
		reverseSystems:    map[int]string{},
		reverseSystemData: map[int]system.Data{},
		systemSec:         map[int]string{},
	}
}

type DataContents struct {
	SystemData []system.Data
}

func (a *App) Prep() error {
	if err := a.loadSystemData(); err != nil {
		return errors.Wrap(err, "could not load system data")
	}

	if err := a.preparePathFinder(); err != nil {
		return errors.Wrap(err, "could not populate data")
	}

	return nil
}

type RouteRequest struct {
	FromSystems   []string `json:"from_systems"`
	ToSystem      string   `json:"to_system"`
	ToTag         string   `json:"to_tag"`
	AvoidSystems  []string `json:"avoid_systems"`
	AvoidTags     []string `json:"avoid_tags"`
	PreferNotTags []string `json:"prefer_not_tags"`
}

type RouteResponse struct {
	Error  string
	Routes [][]system.Data
}

type ListResponse struct {
	Error string
	Items []string
}

func (a *App) writeError(w http.ResponseWriter, estr string, code int) {
	resp := RouteResponse{
		Error: estr,
	}

	encoder := json.NewEncoder(w)

	w.WriteHeader(code)
	if err := encoder.Encode(resp); err != nil {
		panic(err)
	}
}

func (a *App) handleGetRoute(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.Header().Add("Content-type", "application/json")

	req := RouteRequest{}
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		a.writeError(w, err.Error(), 400)
		return
	}

	if len(req.FromSystems) < 1 {
		a.writeError(w, "must specify at least one source system", 400)
		return
	}

	if req.ToSystem != "" && req.ToTag != "" {
		a.writeError(w, "cannot provide both target system and tag", 400)
		return
	}

	if req.ToSystem == "" && req.ToTag == "" {
		a.writeError(w, "must provide either target system or tag", 400)
		return
	}

	fromIDs := make([]int, len(req.FromSystems))
	for i, sname := range req.FromSystems {
		fromIDs[i] = a.systems[sname]
	}
	avoidIDs := make([]int, len(req.AvoidSystems))
	for i, aname := range req.AvoidSystems {
		avoidIDs[i] = a.systems[aname]
	}
	avoidTagIDs := make([]int, len(req.AvoidTags))
	for i, tname := range req.AvoidTags {
		avoidTagIDs[i] = a.tags[tname]
	}
	preferNotTagIDs := make([]int, len(req.PreferNotTags))
	for i, tname := range req.PreferNotTags {
		preferNotTagIDs[i] = a.tags[tname]
	}

	var routes [][]int
	var err error

	switch {
	case req.ToSystem != "":
		toID := a.systems[req.ToSystem]
		if len(fromIDs) == 1 {
			routes, err = a.pathfinder.FindShortestRoutes(fromIDs[0], toID, avoidIDs, avoidTagIDs, preferNotTagIDs)
		} else {
			routes, err = a.pathfinder.FindAllShortestRoutes(fromIDs, toID, avoidIDs, avoidTagIDs, preferNotTagIDs)
		}
	case req.ToTag != "":
		toID := a.tags[req.ToTag]
		if len(fromIDs) == 1 {
			routes, err = a.pathfinder.FindShortestRoutesToTag(fromIDs[0], toID, avoidIDs, avoidTagIDs, preferNotTagIDs)
		} else {
			routes, err = a.pathfinder.FindAllShortestRoutesToTag(fromIDs, toID, avoidIDs, avoidTagIDs, preferNotTagIDs)
		}
	}

	if err != nil {
		a.writeError(w, errors.Wrap(err, "could not find a viable route").Error(), 404)
		return
	}

	resp := RouteResponse{
		Routes: make([][]system.Data, len(routes)),
	}

	for i, route := range routes {
		resp.Routes[i] = a.GetNiceRoute(route)
	}

	encoder := json.NewEncoder(w)
	w.WriteHeader(200)
	if err := encoder.Encode(resp); err != nil {
		panic(err)
	}
}

func (a *App) handleListTags(w http.ResponseWriter, r *http.Request) {
	resp := ListResponse{
		Items: make([]string, 0, len(a.tags)),
	}

	for k := range a.tags {
		resp.Items = append(resp.Items, k)
	}

	w.Header().Add("Content-type", "application/json")

	encoder := json.NewEncoder(w)

	w.WriteHeader(200)
	if err := encoder.Encode(resp); err != nil {
		panic(err)
	}
}

func (a *App) handleListSystems(w http.ResponseWriter, r *http.Request) {
	resp := ListResponse{
		Items: make([]string, 0, len(a.systems)),
	}

	for k := range a.systems {
		resp.Items = append(resp.Items, k)
	}

	w.Header().Add("Content-type", "application/json")

	encoder := json.NewEncoder(w)

	w.WriteHeader(200)
	if err := encoder.Encode(resp); err != nil {
		panic(err)
	}
}

func (a *App) Serve() error {
	http.HandleFunc("/get_routes", a.handleGetRoute)
	http.HandleFunc("/list_tags", a.handleListTags)
	http.HandleFunc("/list_systems", a.handleListSystems)

	return http.ListenAndServe(a.listen, nil)
}

func (a *App) GetNiceRoute(route []int) []system.Data {
	niceRoute := make([]system.Data, len(route))
	for j, id := range route {
		niceRoute[j] = a.reverseSystemData[id]
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
		a.reverseSystemData[sd.ID] = sd
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
