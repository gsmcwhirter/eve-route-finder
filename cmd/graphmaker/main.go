package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

func run() error {
	app := NewApp()

	pflag.StringVarP(&app.sourceDir, "source-dir", "s", "", "eve source file directory")
	pflag.StringVarP(&app.dataDir, "data-dir", "d", "", "local data directory")
	pflag.StringVarP(&app.outFile, "out-file", "o", "", "output file")

	pflag.Parse()

	return app.Run()
}

func main() {
	if err := run(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
