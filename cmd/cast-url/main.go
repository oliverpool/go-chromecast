package main

import (
	"context"
	"fmt"
	"os"

	"github.com/oliverpool/go-chromecast/command/media/tvnow_dash"
	"github.com/oliverpool/go-chromecast/command/media/youtube"

	"github.com/oliverpool/go-chromecast/cli"
	"github.com/oliverpool/go-chromecast/command/media"
)

func fatalf(format string, a ...interface{}) {
	fmt.Printf(format, a...)
	fmt.Println()
	os.Exit(1)
}

func main() {
	ctx := context.Background()

	rawurl := "https://youtu.be/b-GIBLX3nAk"
	if len(os.Args) > 1 {
		rawurl = os.Args[1]
	}

	logger := cli.NewLogger(os.Stdout)

	client, status, err := cli.FirstClientWithStatus(ctx, logger)
	if err != nil {
		fatalf(err.Error())
	}

	loaders := map[string]media.URLLoader{
		"tvnow":   tvnow_dash.URLLoader,
		"youtube": youtube.URLLoader,
	}

	for name, l := range loaders {
		loader, err := l(rawurl)
		if err != nil {
			logger.Log("loader", name, "err", err)
			continue
		}
		_, err = loader(client, status)
		if err != nil {
			logger.Log("loader", name, "unable to load", err)
			continue
		}
		return
	}
	fatalf("No supported loader found")
}