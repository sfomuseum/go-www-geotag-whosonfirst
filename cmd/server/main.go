package main

import (
	_ "github.com/sfomuseum/go-www-geotag-whosonfirst/writer"
)

import (
	"context"
	"github.com/sfomuseum/go-flags"
	wof_app "github.com/sfomuseum/go-www-geotag-whosonfirst/app"
	"github.com/sfomuseum/go-www-geotag/app"
	"log"
	"net/http"
)

func main() {

	ctx := context.Background()

	fs, err := app.CommonFlags()

	if err != nil {
		log.Fatalf("Failed to instantiate common flags, %v", err)
	}

	err = wof_app.AppendWhosOnFirstFlags(fs)

	if err != nil {
		log.Fatalf("Failed to append Who's On First flags, %v", err)
	}

	flags.Parse(fs)

	err = flags.SetFlagsFromEnvVars(fs, "GEOTAG")

	if err != nil {
		log.Fatalf("Failed to set flags from env vars, %v", err)
	}

	err = wof_app.AssignWhosOnFirstFlags(fs)

	if err != nil {
		log.Fatalf("Failed to assign Who's On First flags, %v", err)
	}

	mux := http.NewServeMux()

	err = app.AppendAssetHandlers(ctx, fs, mux)

	if err != nil {
		log.Fatalf("Failed to append asset handlers, %v", err)
	}

	err = app.AppendEditorHandler(ctx, fs, mux)

	if err != nil {
		log.Fatalf("Failed to append editor handler, %v", err)
	}

	err = app.AppendProxyTilesHandlerIfEnabled(ctx, fs, mux)

	if err != nil {
		log.Fatalf("Failed to append proxy tiles handler, %v", err)
	}

	err = app.AppendWriterHandlerIfEnabled(ctx, fs, mux)

	if err != nil {
		log.Fatalf("Failed to append writer handler, %v", err)
	}

	s, err := app.NewServer(ctx, fs)

	if err != nil {
		log.Fatalf("Failed to create application server, %v", err)
	}

	log.Printf("Listening on %s", s.Address())

	err = s.ListenAndServe(ctx, mux)

	if err != nil {
		log.Fatalf("Failed to start server, %v", err)
	}
}
