package main

type RawSystemData struct {
	Security  float64             `yaml:"security"`
	Stargates map[string]Stargate `yaml:"stargates"`
}

type Stargate struct {
	Destination string `yaml:"destination"`
}
