package esx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/tidwall/sjson"
)

type ESTyped struct {
	c *elasticsearch.TypedClient
}

func NewTyped(cfg elasticsearch.Config) (*ESTyped, error) {
	c, err := elasticsearch.NewTypedClient(cfg)
	if err != nil {
		return nil, err
	}

	resp, err := c.Info().Do(context.Background())
	if err != nil {
		return nil, err
	}

	slog.Info("es cluster info", "info", JsonMarshalString(resp))

	return &ESTyped{c: c}, nil
}

func (e *ESTyped) SearchWithRaw(index string, jsonBody string, from, size int, handler func(fields map[string]json.RawMessage)) error {
	if handler == nil {
		return errors.New("handler is nil")
	}

	if from < 0 {
		from = 0
	}
	if size < 0 {
		size = 10
	}

	for {
		jsonBody, err := sjson.Set(jsonBody, "from", from)
		if err != nil {
			return err
		}
		jsonBody, err = sjson.Set(jsonBody, "size", size)
		if err != nil {
			return err
		}

		res, err := e.c.Search().Index(index).
			Raw(strings.NewReader(jsonBody)).Do(context.Background())
		if err != nil {
			return err
		}

		if err != nil {
			slog.Error("Search Do: " + err.Error())
			return err
		}

		if len(res.Hits.Hits) == 0 {
			slog.Info("no more hits")
			break
		}

		slog.Info(fmt.Sprintf("hits: %d", len(res.Hits.Hits)))

		for _, hit := range res.Hits.Hits {
			handler(hit.Fields)
		}

		from += size
	}

	return nil
}

func JsonMarshalString(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}

	return string(data)
}
