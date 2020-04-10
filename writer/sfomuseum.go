package writer

import (
	"context"
	"github.com/sfomuseum/go-www-geotag/geojson"	
	geotag "github.com/sfomuseum/go-www-geotag/writer"
	"encoding/json"
	"io"
	"bytes"
	"os"
)

func init() {
	ctx := context.Background()
	geotag.RegisterWriter(ctx, "sfomuseum", NewSFOMuseumWriter)
}

type SFOMuseumWriter struct {
	geotag.Writer
}

func NewSFOMuseumWriter(ctx context.Context, uri string) (geotag.Writer, error) {

	wr := &SFOMuseumWriter{}
	return wr, nil
}

func (wr *SFOMuseumWriter) WriteFeature(ctx context.Context, uri string, f *geojson.GeotagFeature) error {

	body, err := json.Marshal(f)

	if err != nil {
		return err
	}

	br := bytes.NewReader(body)
	_, err = io.Copy(os.Stdout, br)

	if err != nil {
		return err
	}

	return nil
}
