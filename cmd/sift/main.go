package main

import (
	"context"
	"log"
	"os"

	"github.com/batu3384/sift/internal/cli"
)

func main() {
	ctx := context.Background()
	if err := cli.NewRootCommand().ExecuteContext(ctx); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
