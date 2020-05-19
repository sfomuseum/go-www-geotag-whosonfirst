package writer

import (
	"bufio"
	"bytes"
	"context"
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
const GEOTAG_LABEL string = "geotag-fov" // field of view

func init() {
	ctx := context.Background()
	geotag_writer.RegisterWriter(ctx, "whosonfirst", NewWhosOnFirstGeotagWriter)
}

// please put this in a common whosonfirst geojson/feature package
// (20200413/thisisaaronland)

type WhosOnFirstAltFeature struct {
	Id         int64                  `json:"id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Geometry   interface{}            `json:"geometry"`
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

	if reader_uri == "" {
		return nil, errors.New("Missing reader parameter")
	}

	writer_uri, err = url.QueryUnescape(writer_uri)

	if err != nil {
		return nil, err
	}

	reader_uri, err = url.QueryUnescape(reader_uri)

	if err != nil {
		return nil, err
	}

	wof_wr, err := writer.NewWriter(ctx, writer_uri)

	if err != nil {
		return nil, err
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

	io_wr, err := geotag_writer.GetIOWriterFromContext(ctx)

	if err == nil {

		ctx, err = writer.SetIOWriterWithContext(ctx, io_wr)

		if err != nil {
			return err
		}

	}

	// for local debugging
	// uri = "1511948897"

	wof_id, uri_args, err := wof_uri.ParseURI(uri)

	if err != nil {
		return err
	}

	if uri_args.IsAlternate {
		return errors.New("Alt files are not supported yet")
	}

	rel_path, err := wof_uri.Id2RelPath(wof_id)

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

	pov, err := geotag_f.PointOfView()

	if err != nil {
		return err
	}

	tgt, err := geotag_f.Target()

	if err != nil {
		return err
	}

	pov_coords := pov.Coordinates
	tgt_coords := tgt.Coordinates

	geotag_props := geotag_f.Properties

	alt_props := map[string]interface{}{
		"wof:id":                  wof_id,
		"wof:repo":                main_repo,
		"src:alt_label":           GEOTAG_LABEL,
		"src:geom":                wr.geom_source,
		"geotag:angle":            geotag_props.Angle,
		"geotag:bearing":          geotag_props.Bearing,
		"geotag:distance":         geotag_props.Distance,
		"geotag:camera_longitude": pov_coords[0],
		"geotag:camera_latitude":  pov_coords[1],
		"geotag:target_longitude": tgt_coords[0],
		"geotag:target_latitude":  tgt_coords[1],
	}

	alt_geom, err := geotag_f.FieldOfView()

	if err != nil {
		return err
	}

	alt_feature := &WhosOnFirstAltFeature{
		Type:       "Feature",
		Id:         wof_id,
		Properties: alt_props,
		Geometry:   alt_geom,
	}

	//

	alt_body, err := FormatAltFeature(alt_feature)

	if err != nil {
		return err
	}

	alt_uri_geom := &wof_uri.AltGeom{
		Source: GEOTAG_LABEL,
	}

	alt_uri_args := &wof_uri.URIArgs{
		IsAlternate: true,
		AltGeom:     alt_uri_geom,
	}

	alt_uri, err := wof_uri.Id2RelPath(wof_id, alt_uri_args)

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

		main_body, err = sjson.SetBytes(main_body, "geometry", pov)

		if err != nil {
			return err
		}

		to_update := map[string]interface{}{
			"lbl:longitude":           pov_coords[0],
			"lbl:latitude":            pov_coords[1],
			"geotag:camera_longitude": pov_coords[0],
			"geotag:camera_latitude":  pov_coords[1],
			"geotag:target_longitude": tgt_coords[0],
			"geotag:target_latitude":  tgt_coords[1],
			"geotag:angle":            geotag_props.Angle,
			"src:geom":                wr.geom_source,
		}

		geom_alt := []string{
			GEOTAG_LABEL,
		}

		geom_alt_rsp := gjson.GetBytes(main_body, "properties.src:geom_alt")

		if geom_alt_rsp.Exists() {

			for _, r := range geom_alt_rsp.Array() {

				if r.String() == GEOTAG_LABEL {
					continue
				}

				geom_alt = append(geom_alt, r.String())
			}
		}

		to_update["src:geom_alt"] = geom_alt

		for k, v := range to_update {

			path := fmt.Sprintf("properties.%s", k)
			main_body, err = sjson.SetBytes(main_body, path, v)

			if err != nil {
				return err
			}
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

func (wr *WhosOnFirstGeotagWriter) Close(ctx context.Context) error {
	return nil
}

func FormatAltFeature(f *WhosOnFirstAltFeature) ([]byte, error) {

	// please standardize on a common whosonfirst geojson/feature package
	// (20200413/thisisaaronland)

	ff := &format.Feature{
		Type:       f.Type,
		ID:         f.Id,
		Properties: f.Properties,
		Geometry:   f.Geometry,
	}

	body, err := format.FormatFeature(ff)

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
