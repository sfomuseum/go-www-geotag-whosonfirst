package main

import (
	_ "github.com/sfomuseum/go-www-sfomuseum-geotag/writer"
)

import (
	"context"
	"github.com/sfomuseum/go-www-geotag/app"
	"github.com/sfomuseum/go-www-geotag/flags"
	"log"
	"net/http"
)

func main() {

	ctx := context.Background()

	fl, err := flags.CommonFlags()

	if err != nil {
		log.Fatalf("Failed to instantiate common flags, %v", err)
	}

	flags.Parse(fl)

	mux := http.NewServeMux()

	err = app.AppendAssetHandlers(ctx, fl, mux)

	if err != nil {
		log.Fatalf("Failed to append asset handlers, %v", err)
	}

	err = app.AppendEditorHandler(ctx, fl, mux)

	if err != nil {
		log.Fatalf("Failed to append editor handler, %v", err)
	}

	err = app.AppendProxyTilesHandlerIfEnabled(ctx, fl, mux)

	if err != nil {
		log.Fatalf("Failed to append proxy tiles handler, %v", err)
	}

	err = app.AppendWriterHandlerIfEnabled(ctx, fl, mux)

	if err != nil {
		log.Fatalf("Failed to append writer handler, %v", err)
	}

	s, err := app.NewServer(ctx, fl)

	if err != nil {
		log.Fatalf("Failed to create application server, %v", err)
	}

	log.Printf("Listening on %s", s.Address())

	err = s.ListenAndServe(mux)

	if err != nil {
		log.Fatalf("Failed to start server, %v", err)
	}
}
