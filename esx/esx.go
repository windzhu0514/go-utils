package esx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/tidwall/gjson"
)

type ES struct {
	c *elasticsearch.Client
}

func New(cfg elasticsearch.Config) (*ES, error) {
	c, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	resp, err := c.Info()
	if err != nil {
		return nil, err
	}

	slog.Info("es cluster info: \n" + resp.String())

	return &ES{c: c}, nil
}

func (e *ES) SearchWithRaw(index string, jsonBody string, handler func(string)) error {
	if handler == nil {
		return errors.New("handler is nil")
	}

	from := 0
	size := 1
	for {
		res, err := e.c.Search(
			e.c.Search.WithContext(context.Background()),
			e.c.Search.WithIndex(index),
			e.c.Search.WithBody(strings.NewReader(jsonBody)),
			e.c.Search.WithTrackTotalHits(true),
			e.c.Search.WithFrom(from),
			e.c.Search.WithSize(size),
		)
		if err != nil {
			return err
		}

		defer res.Body.Close()

		if res.IsError() {
			slog.Error("search error: " + res.String())

			return errors.New("search error")
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		hits := gjson.GetBytes(body, "hits.hits.#").Int()
		fmt.Println("hits length:", hits)
		if hits == 0 {
			break
		}

		array := gjson.GetBytes(body, "hits.hits.#.fields").Array()
		for _, v := range array {
			handler(v.String())
		}

		time.Sleep(100 * time.Millisecond)

		from += size
	}

	return nil
}
