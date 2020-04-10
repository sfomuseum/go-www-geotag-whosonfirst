package writer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/sfomuseum/go-www-geotag/geojson"
	geotag_writer "github.com/sfomuseum/go-www-geotag/writer"
	"github.com/tidwall/sjson"
	wof_uri "github.com/whosonfirst/go-whosonfirst-uri"
	"github.com/whosonfirst/go-writer"	
	"io/ioutil"
	"net/url"
)

const ALT_LABEL string = "geotag"

func init() {
	ctx := context.Background()
	geotag_writer.RegisterWriter(ctx, "whosonfirst", NewWhosOnFirstWriter)
}

type WhosOnFirstWriter struct {
	geotag_writer.Writer
	writer writer.Writer
}

func NewWhosOnFirstWriter(ctx context.Context, uri string) (geotag_writer.Writer, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	q := u.Query()

	wof_wr_uri := q.Get("writer")

	if wof_wr_uri == "" {
		return nil, errors.New("Missing writer parameter")
	}
	
	wof_wr, err := writer.NewWriter(ctx, wof_wr_uri)

	if err != nil {
		return nil, err
	}
	
	wr := &WhosOnFirstWriter{
		writer: wof_wr,
	}
	
	return wr, nil
}

func (wr *WhosOnFirstWriter) WriteFeature(ctx context.Context, uri string, f *geojson.GeotagFeature) error {

	id, _, err := wof_uri.ParseURI(uri)

	if err != nil {
		return err
	}
	
	body, err := json.Marshal(f)

	if err != nil {
		return err
	}

	alt_body, err := GeotagFeatureToAltFeature(ctx, uri, body)

	if err != nil {
		return err
	}

	alt_geom := &wof_uri.AltGeom{
		Source:   ALT_LABEL,
	}

	uri_args := &wof_uri.URIArgs{
		IsAlternate: true,
		AltGeom:     alt_geom,
	}
	
	alt_uri, err := wof_uri.Id2Fname(id, uri_args)

	if err != nil {
		return err
	}
	
	br := bytes.NewReader(alt_body)
	fh := ioutil.NopCloser(br)
	
	return wr.writer.Write(ctx, alt_uri, fh)
}

func GeotagFeatureToAltFeature(ctx context.Context, uri string, body []byte) ([]byte, error) {

	id, _, err := wof_uri.ParseURI(uri)

	if err != nil {
		return nil, err
	}

	body, err = sjson.SetBytes(body, "id", id)

	if err != nil {
		return nil, err
	}

	body, err = sjson.SetBytes(body, "properties.wof:id", id)

	if err != nil {
		return nil, err
	}

	body, err = sjson.SetBytes(body, "properties.src:alt_label", ALT_LABEL)

	if err != nil {
		return nil, err
	}

	body, err = sjson.SetBytes(body, "properties.src:geom", "geotag")

	if err != nil {
		return nil, err
	}

	return body, nil
}
