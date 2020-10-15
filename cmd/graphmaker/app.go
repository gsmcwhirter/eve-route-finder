package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gsmcwhirter/eve-route-finder/pkg/system"
	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

type App struct {
	dataDir   string
	sourceDir string
	outFile   string

	ctx         context.Context
	cancel      context.CancelFunc
	workers     *errgroup.Group
	tokenpool   chan struct{}
	parsed      chan system.Data
	systemGates chan SystemGate

	trigFinal    map[string]struct{}
	trigMinor    map[string]struct{}
	edenMinor    map[string]struct{}
	edenFortress map[string]struct{}

	systemGatesData map[string]string
	SystemData      []system.Data
}

func NewApp() *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		ctx:         ctx,
		cancel:      cancel,
		workers:     &errgroup.Group{},
		tokenpool:   make(chan struct{}, 20),
		parsed:      make(chan system.Data, 20),
		systemGates: make(chan SystemGate, 20),

		trigFinal:    map[string]struct{}{},
		trigMinor:    map[string]struct{}{},
		edenMinor:    map[string]struct{}{},
		edenFortress: map[string]struct{}{},
	}
}

func (a *App) Run() error {
	if err := a.LoadData(); err != nil {
		return err
	}

	eg := &errgroup.Group{}

	eg.Go(a.HandleParsedResults)
	eg.Go(a.HandleSystemGates)

	if err := filepath.Walk(a.sourceDir, a.WalkFile); err != nil {
		a.cancel()
		return errors.Wrap(err, "walk failed")
	}

	if err := a.workers.Wait(); err != nil {
		a.cancel()
		return errors.Wrap(err, "workers wait failed")
	}

	close(a.parsed)
	close(a.systemGates)

	if err := eg.Wait(); err != nil {
		return errors.Wrap(err, "sidecar wait failed")
	}

	// Put destinations as real names
	for i := 0; i < len(a.SystemData); i++ {
		realDest := make([]string, 0, len(a.SystemData[i].Destinations))
		for _, gid := range a.SystemData[i].Destinations {
			if ds, ok := a.systemGatesData[gid]; ok {
				realDest = append(realDest, ds)
			}
		}
		a.SystemData[i].Destinations = realDest
	}

	f, err := os.Create(a.outFile)
	if err != nil {
		return errors.Wrap(err, "could not open file to write")
	}
	defer deferutil.CheckDefer(f.Close)

	encoder := yaml.NewEncoder(f)
	if err := encoder.Encode(a); err != nil {
		return errors.Wrap(err, "could not encode yaml")
	}

	return nil
}

type SystemGate struct {
	System string
	GateID string
}

var ErrParseFailed = errors.New("could not parse system data")

func loadFile(target map[string]struct{}, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "could not read file", "path", path)
	}
	defer deferutil.CheckDefer(f.Close)

	if err := loadData(target, f); err != nil {
		return errors.Wrap(err, "could not load file", "path", path)
	}

	return nil
}

func loadData(target map[string]struct{}, f io.Reader) error {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		target[line] = struct{}{}
	}

	return nil
}

func (a *App) LoadData() error {
	trigFinalFile := filepath.Join(a.dataDir, "trig-final-lim.txt")
	trigMinorFile := filepath.Join(a.dataDir, "trig-minor-victory.txt")
	edenMinorFile := filepath.Join(a.dataDir, "edencom-minor-victory.txt")
	edenFortressFile := filepath.Join(a.dataDir, "edencom-fortress.txt")

	if err := loadFile(a.trigFinal, trigFinalFile); err != nil {
		return errors.Wrap(err, "trigFinal")
	}

	if err := loadFile(a.trigMinor, trigMinorFile); err != nil {
		return errors.Wrap(err, "trigMinor")
	}

	if err := loadFile(a.edenMinor, edenMinorFile); err != nil {
		return errors.Wrap(err, "edenMinor")
	}

	if err := loadFile(a.edenFortress, edenFortressFile); err != nil {
		return errors.Wrap(err, "edenFortress")
	}

	return nil
}

func (a *App) HandleSystemGates() error {
	systemGates := map[string]string{}

	for {
		select {
		case <-a.ctx.Done():
			a.systemGatesData = systemGates
			return a.ctx.Err()
		case v, ok := <-a.systemGates:
			if !ok { // channel closed
				a.systemGatesData = systemGates
				return nil
			}

			systemGates[v.GateID] = v.System
		}
	}
}

func (a *App) HandleParsedResults() error {
	systemData := make([]system.Data, 0, 1000)

	i := 0

	for {
		select {
		case <-a.ctx.Done():
			a.SystemData = systemData
			return a.ctx.Err()
		case v, ok := <-a.parsed:
			if !ok {
				a.SystemData = systemData
				return nil
			}

			v.ID = i
			i++
			systemData = append(systemData, v)
		}
	}
}

func (a *App) WalkFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Printf("Error parsing %v: %v\n", path, err)
		return err
	}

	if info.IsDir() {
		return nil
	}

	if !strings.HasPrefix(path, a.sourceDir) {
		return errors.Wrap(ErrParseFailed, "file not within source directory", "path", path)
	}

	a.tokenpool <- struct{}{} // wait until we can work; ParseSystemData should pull a token back off
	a.workers.Go(a.ParseSystemData(path))

	return nil
}

func (a *App) ParseSystemData(path string) func() error {
	return func() error {
		defer func() {
			<-a.tokenpool // release our token
		}()

		relpath := strings.TrimPrefix(path, a.sourceDir)
		reldir, basename := filepath.Split(relpath)
		if basename != "solarsystem.staticdata" {
			return nil
		}
		reldir = strings.Trim(reldir, "/")

		parts := strings.Split(reldir, "/")
		if len(parts) != 3 {
			return errors.Wrap(ErrParseFailed, "could not parse file path", "path", path)
		}

		region := parts[0]
		constellation := parts[1]
		systemName := parts[2]

		f, err := os.Open(path)
		if err != nil {
			return errors.Wrap(err, "could not open file to read", "path", path)
		}
		defer deferutil.CheckDefer(f.Close)

		decoder := yaml.NewDecoder(f)
		rawInfo := RawSystemData{}
		if err := decoder.Decode(&rawInfo); err != nil {
			return errors.Wrap(err, "could not decode yaml", "path", path)
		}

		secStatus := classifySecurity(rawInfo.Security)

		tags := []string{
			region,
			constellation,
			secStatus,
		}

		if _, ok := a.trigFinal[systemName]; ok {
			tags = append(tags, "trig-final")
			secStatus = "trig"
		} else if _, ok := a.trigMinor[systemName]; ok {
			tags = append(tags, "trig-minor")
		} else if _, ok := a.edenMinor[systemName]; ok {
			tags = append(tags, "eden-minor")
		} else if _, ok := a.edenFortress[systemName]; ok {
			tags = append(tags, "eden-fortress")
		}

		destinations := make([]string, 0, len(rawInfo.Stargates))

		if _, ok := a.trigFinal[systemName]; !ok {
			// we're not final-lim, so we have gates
			for k, v := range rawInfo.Stargates {
				destinations = append(destinations, v.Destination)

				a.systemGates <- SystemGate{
					System: systemName,
					GateID: k,
				}
			}
		}

		systemData := system.Data{
			Name:          systemName,
			Constellation: constellation,
			Region:        region,
			Destinations:  destinations,
			SecStatus:     secStatus,
			Tags:          tags,
		}

		fmt.Printf("%s: %#v\n", reldir, systemData)

		a.parsed <- systemData
		return nil
	}
}

func classifySecurity(sec float64) string {
	if sec >= 0.5 {
		return "high"
	}

	if sec > 0.0 {
		return "low"
	}

	return "null"
}
