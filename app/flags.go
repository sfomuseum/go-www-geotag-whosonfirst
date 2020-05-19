package flags

import (
	"flag"
	"github.com/sfomuseum/go-flags/lookup"
	"net/url"
	"strings"
)

func AppendWhosOnFirstFlags(fs *flag.FlagSet) error {

	fs.String("whosonfirst-writer-uri", "", "A valid whosonfirst/go-writer.Writer URI. If present it will be encoded and used to replace the '{whosonfirst_writer}' string in the -writer-uri flag.")
	fs.String("whosonfirst-reader-uri", "", "A valid whosonfirst/go-reader.Reader URI. If present it will be encoded and used to replace the '{whosonfirst_reader}' string in the -writer-uri flag.")

	return nil
}

func AssignWhosOnFirstFlags(fs *flag.FlagSet) error {

	wof_writer, err := lookup.StringVar(fs, "whosonfirst-writer-uri")

	if err != nil {
		return err
	}

	wof_reader, err := lookup.StringVar(fs, "whosonfirst-reader-uri")

	if err != nil {
		return err
	}

	if wof_writer != "" || wof_writer != "" {

		writer_uri, err := lookup.StringVar(fs, "writer-uri")

		if err != nil {
			return err
		}

		if wof_writer != "" {
			enc_wof_writer := url.QueryEscape(wof_writer)
			writer_uri = strings.Replace(writer_uri, "{whosonfirst_writer}", enc_wof_writer, 1)
		}

		if wof_reader != "" {
			enc_wof_reader := url.QueryEscape(wof_reader)
			writer_uri = strings.Replace(writer_uri, "{whosonfirst_reader}", enc_wof_reader, 1)
		}

		fs.Set("writer-uri", writer_uri)
	}

	return nil
}
