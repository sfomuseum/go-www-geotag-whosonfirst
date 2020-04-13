package writer

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sfomuseum/go-geojson-geotag"
	geotag_writer "github.com/sfomuseum/go-www-geotag/writer"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/tomtaylor/go-whosonfirst-format"
	"github.com/whosonfirst/go-reader"
	"github.com/whosonfirst/go-whosonfirst-export"
	export_options "github.com/whosonfirst/go-whosonfirst-export/options"
	wof_uri "github.com/whosonfirst/go-whosonfirst-uri"
	"github.com/whosonfirst/go-writer"
	"io/ioutil"
	_ "log"
	"net/url"
	"regexp"
)

const GEOTAG_NS string = "geotag"
const GEOTAG_SRC string = "geotag"
const GEOTAG_LABEL string = "geotag"

func init() {
	ctx := context.Background()
	geotag_writer.RegisterWriter(ctx, "whosonfirst", NewWhosOnFirstGeotagWriter)
}

type WhosOnFirstGeotagWriter struct {
	geotag_writer.Writer
	writer      writer.Writer
	reader      reader.Reader
	update      bool
	geom_source string
}

func NewWhosOnFirstGeotagWriter(ctx context.Context, uri string) (geotag_writer.Writer, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	q := u.Query()

	writer_uri := q.Get("writer")
	reader_uri := q.Get("reader")

	if writer_uri == "" {
		return nil, errors.New("Missing writer parameter")
	}

	wof_wr, err := writer.NewWriter(ctx, writer_uri)

	if err != nil {
		return nil, err
	}

	if reader_uri == "" {
		return nil, errors.New("Missing reader parameter")
	}

	wof_rd, err := reader.NewReader(ctx, reader_uri)

	if err != nil {
		return nil, err
	}

	update := false

	if q.Get("update") == "1" {
		update = true
	}

	geom_source := GEOTAG_SRC

	q_source := q.Get("source")

	if q_source != "" {

		re, err := regexp.Compile(`^[a-zA-Z0-9_\-]+$`)

		if err != nil {
			return nil, err
		}

		if !re.MatchString(q_source) {
			return nil, errors.New("Invalid source")
		}

		geom_source = q_source
	}

	wr := &WhosOnFirstGeotagWriter{
		writer:      wof_wr,
		reader:      wof_rd,
		update:      update,
		geom_source: geom_source,
	}

	return wr, nil
}

func (wr *WhosOnFirstGeotagWriter) WriteFeature(ctx context.Context, uri string, geotag_f *geotag.GeotagFeature) error {

	// for local debugging
	// uri = "1511948897"

	id, uri_args, err := wof_uri.ParseURI(uri)

	if err != nil {
		return err
	}

	if uri_args.IsAlternate {
		return errors.New("Alt files are not supported yet")
	}

	rel_path, err := wof_uri.Id2RelPath(id)

	if err != nil {
		return err
	}

	main_fh, err := wr.reader.Read(ctx, rel_path)

	if err != nil {
		return err
	}

	main_body, err := ioutil.ReadAll(main_fh)

	if err != nil {
		return err
	}

	repo_rsp := gjson.GetBytes(main_body, "properties.wof:repo")

	if !repo_rsp.Exists() {
		return errors.New("Missing wof:repo")
	}

	main_repo := repo_rsp.String()

	//

	geotag_body, err := json.Marshal(geotag_f)

	if err != nil {
		return err
	}

	alt_body, err := GeotagFeatureToAltFeature(ctx, uri, geotag_body)

	if err != nil {
		return err
	}

	alt_body, err = sjson.SetBytes(alt_body, "properties.wof:repo", main_repo)

	if err != nil {
		return err
	}

	alt_body, err = sjson.SetBytes(alt_body, "properties.src:geom", wr.geom_source)

	if err != nil {
		return err
	}

	alt_body, err = Format(alt_body)

	if err != nil {
		return err
	}

	//

	alt_geom := &wof_uri.AltGeom{
		Source: GEOTAG_LABEL,
	}

	alt_args := &wof_uri.URIArgs{
		IsAlternate: true,
		AltGeom:     alt_geom,
	}

	alt_uri, err := wof_uri.Id2RelPath(id, alt_args)

	if err != nil {
		return err
	}

	alt_br := bytes.NewReader(alt_body)
	alt_fh := ioutil.NopCloser(alt_br)

	err = wr.writer.Write(ctx, alt_uri, alt_fh)

	if err != nil {
		return err
	}

	if wr.update {

		pov, err := geotag_f.PointOfView()

		if err != nil {
			return err
		}

		main_body, err = sjson.SetBytes(main_body, "geometry", pov)

		if err != nil {
			return err
		}

		main_body, err = sjson.SetBytes(main_body, "properties.lbl:longitude", pov.Coordinates[0])

		if err != nil {
			return err
		}

		main_body, err = sjson.SetBytes(main_body, "properties.lbl:latitude", pov.Coordinates[1])

		if err != nil {
			return err
		}

		geom_alt := []string{
			GEOTAG_SRC,
		}

		geom_alt_rsp := gjson.GetBytes(main_body, "properties.src:geom_alt")

		if geom_alt_rsp.Exists() {

			for _, r := range geom_alt_rsp.Array() {

				if r.String() == GEOTAG_SRC {
					continue
				}

				geom_alt = append(geom_alt, r.String())
			}
		}

		main_body, err = sjson.SetBytes(main_body, "properties.src:geom_alt", geom_alt)

		if err != nil {
			return err
		}

		main_body, err = sjson.SetBytes(main_body, "properties.src:geom", wr.geom_source)

		if err != nil {
			return err
		}

		// please refactor everything about whosonfirst/go-whosonfirst-export...
		// (20200410/thisisaaronland)

		ex_opts, err := export_options.NewDefaultOptions()

		if err != nil {
			return err
		}

		var main_buf bytes.Buffer
		main_wr := bufio.NewWriter(&main_buf)

		err = export.Export(main_body, ex_opts, main_wr)

		if err != nil {
			return err
		}

		main_wr.Flush()
		main_br := bytes.NewReader(main_buf.Bytes())
		main_fh := ioutil.NopCloser(main_br)

		err = wr.writer.Write(ctx, rel_path, main_fh)

		if err != nil {
			return err
		}

	}

	return nil
}

func GeotagFeatureToAltFeature(ctx context.Context, uri string, body []byte) ([]byte, error) {

	id, _, err := wof_uri.ParseURI(uri)

	if err != nil {
		return nil, err
	}

	to_ns := []string{
		"angle",
		"bearing",
		"distance",
	}

	for _, k := range to_ns {

		path_old := fmt.Sprintf("properties.%s", k)
		path_ns := fmt.Sprintf("properties.%s:%s", GEOTAG_NS, k)

		rsp := gjson.GetBytes(body, path_old)

		if !rsp.Exists() {
			msg := fmt.Sprintf("Missing '%s' property", path_old)
			return nil, errors.New(msg)
		}

		body, err = sjson.DeleteBytes(body, path_old)

		if err != nil {
			return nil, err
		}

		body, err = sjson.SetBytes(body, path_ns, rsp.Float())

		if err != nil {
			return nil, err
		}
	}

	body, err = sjson.SetBytes(body, "id", id)

	if err != nil {
		return nil, err
	}

	body, err = sjson.SetBytes(body, "properties.wof:id", id)

	if err != nil {
		return nil, err
	}

	body, err = sjson.SetBytes(body, "properties.src:alt_label", GEOTAG_LABEL)

	if err != nil {
		return nil, err
	}

	body, err = sjson.SetBytes(body, "properties.src:geom", GEOTAG_SRC)

	if err != nil {
		return nil, err
	}

	return body, nil
}

func Format(feature []byte) ([]byte, error) {

	// TODO: Add FormatBytes to go-whosonfirst-feature

	var f format.Feature
	json.Unmarshal(feature, &f)

	body, err := format.FormatFeature(&f)

	if err != nil {
		return nil, err
	}

	// TODO: Add omitempty hooks for bbox in go-whosonfirst-feature

	body, err = sjson.DeleteBytes(body, "bbox")

	if err != nil {
		return nil, err
	}

	return body, nil
}
