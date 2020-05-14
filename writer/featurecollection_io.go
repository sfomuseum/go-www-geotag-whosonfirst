package writer

import (
	"context"
	"errors"
	wof_writer "github.com/whosonfirst/go-writer"
	"io"
	"net/url"
	"strconv"
	"sync"
)

type FeatureCollectionIOWriter struct {
	wof_writer.Writer
	mu             *sync.RWMutex
	count_features int
	seen           int
}

func init() {

	ctx := context.Background()
	err := wof_writer.RegisterWriter(ctx, "featurecollection-io", NewFeatureCollectionIOWriter)

	if err != nil {
		panic(err)
	}
}

func NewFeatureCollectionIOWriter(ctx context.Context, uri string) (wof_writer.Writer, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	q := u.Query()

	str_count_features := q.Get("count_features")

	if str_count_features == "" {
		return nil, errors.New("Missing count_features parameter")
	}

	count_features, err := strconv.Atoi(str_count_features)

	if err != nil {
		return nil, err
	}

	mu := new(sync.RWMutex)

	wr := &FeatureCollectionIOWriter{
		count_features: count_features,
		seen:           0,
		mu:             mu,
	}

	return wr, nil
}

func (wr *FeatureCollectionIOWriter) Write(ctx context.Context, uri string, fh io.ReadCloser) error {

	if wr.count_features >= 1 && wr.seen == wr.count_features {
		return errors.New("Exceeded expected number of features")
	}

	wr.mu.Lock()
	defer wr.mu.Unlock()

	target, err := wof_writer.GetIOWriterFromContext(ctx)

	if err != nil {
		return err
	}

	if wr.seen == 0 {
		target.Write([]byte(`{"type":"FeatureCollection","features":[`))
	} else {
		target.Write([]byte(`,`))
	}

	_, err = io.Copy(target, fh)

	if err != nil {
		return err
	}

	wr.seen += 1

	if wr.count_features >= 1 && wr.seen == wr.count_features {
		target.Write([]byte(`]}`))
		wr.seen = 0
	}

	return nil
}

func (wr *FeatureCollectionIOWriter) URI(uri string) string {
	return uri
}
