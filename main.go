package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/SuperInstance/gh-dungeons/game"
)

func main() {
	// Parse flags
	mergeMode := flag.Bool("merge", false, "Enable merge conflict display mode")
	platoURL := flag.String("plato-url", "", "PLATO server URL (enables PLATO mode)")
	platoRoom := flag.String("plato-room", "", "Specific PLATO room to dungeonify (optional)")
	flag.Parse()

	// Build game options
	opts := []game.GameOption{}
	if *mergeMode {
		opts = append(opts, game.WithMergeMode(true))
	}
	if *platoURL != "" {
		opts = append(opts, game.WithPLATOURL(*platoURL))
		if *platoRoom != "" {
			opts = append(opts, game.WithPLATORoom(*platoRoom))
		}
	}

	g, err := game.New(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing game: %v\n", err)
		os.Exit(1)
	}
	defer g.Close()

	if err := g.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running game: %v\n", err)
		os.Exit(1)
	}
}
