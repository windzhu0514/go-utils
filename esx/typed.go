// 使用 ElasticSearch 前端页面的 Request 进行数据拉取，减少构造请求的工作量
package esx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/closepointintime"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type ESTyped struct {
	c *elasticsearch.TypedClient
}

func NewTypedClient(cfg elasticsearch.Config) (*ESTyped, error) {
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

// SearchWithRaw 使用 Elasticsearch 的 Inspect 里 Request 数据进行数据拉取，可自定义起始位置和每次拉取的数据量
// 最多只能拉取 10000 条数据
func (e *ESTyped) SearchWithRaw(index string, jsonBody string, from, size int, handler func(id string, fields map[string]json.RawMessage) bool) error {
	if index == "" {
		return errors.New("index is empty")
	}
	if jsonBody == "" {
		return errors.New("jsonBody is empty")
	}
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
		var err error
		jsonBody, err = sjson.Set(jsonBody, "from", from)
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

		for _, failure := range res.Shards_.Failures {
			slog.Error("Search Do: " + failure.Reason.Type + ": " + *failure.Reason.Reason)
			return errors.New(failure.Reason.Type)
		}

		if len(res.Hits.Hits) == 0 {
			slog.Info("no more hits")
			break
		}

		slog.Info(fmt.Sprintf("hits: %d", len(res.Hits.Hits)))

		for _, hit := range res.Hits.Hits {
			handler(*hit.Id_, hit.Fields)
		}

		from += size
	}

	return nil
}

// SearchWithRawByPIT 使用 Elasticsearch 的 Inspect 里 Request 数据进行数据拉取，可自定义起始位置、每次拉取的数据量和时间间隔
// 时间间隔参数用于指定每次搜索的时间范围，单位为小时，避免一次搜索数据量过大，导致es服务内存占用过高
// 使用 Elasticsearch 的 PointInTime API，https://www.elastic.co/guide/en/elasticsearch/reference/current/point-in-time-api.html
// 可遍历超过10000条数据
func (e *ESTyped) SearchWithRawByPIT(index string, jsonBody string, size, intervalhours int, handler func(id string, fields map[string]json.RawMessage) bool) error {
	if index == "" {
		return errors.New("index is empty")
	}
	if jsonBody == "" {
		return errors.New("jsonBody is empty")
	}
	if handler == nil {
		return errors.New("handler is nil")
	}

	if size < 0 {
		size = 10
	}
	if intervalhours < 0 {
		intervalhours = 1 // 每次搜索1小时的数据
	}

	pitResp, err := e.c.OpenPointInTime(index).KeepAlive("5m").Do(context.Background())
	if err != nil {
		return err
	}

	defer func() {
		_, err = e.c.ClosePointInTime().Request(&closepointintime.Request{Id: pitResp.Id}).Do(context.Background())
		if err != nil {
			slog.Error("ClosePointInTime Do: " + err.Error())
		}
	}()

	gte := gjson.Get(jsonBody, "query.bool.filter.#.range.logtime.gte").Array()[0].String()
	lte := gjson.Get(jsonBody, "query.bool.filter.#.range.logtime.lte").Array()[0].String()
	start, err := time.Parse(time.RFC3339Nano, gte)
	if err != nil {
		return err
	}
	end, err := time.Parse(time.RFC3339Nano, lte)
	if err != nil {
		return err
	}

	jsonBody, err = sjson.Set(jsonBody, "size", size)
	if err != nil {
		return err
	}

	bFirstTime := true
	var lastSortValue []types.FieldValue
	for start.Before(end) {
		endtime := start.Add(time.Hour * time.Duration(intervalhours))
		if endtime.After(end) {
			endtime = end
		}

		slog.Info(fmt.Sprintf("search from %s to %s", start.Format(time.RFC3339Nano), endtime.Format(time.RFC3339Nano)))

		body, err := sjson.Set(jsonBody, "query.bool.filter.#.range.logtime.gte", start.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
		body, err = sjson.Set(body, "query.bool.filter.#.range.logtime.lte", endtime.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}

		for {
			body, err = sjson.Set(body, "pit.id", pitResp.Id)
			if err != nil {
				return err
			}
			body, err = sjson.Set(body, "pit.keep_alive", "5m")
			if err != nil {
				return err
			}
			if bFirstTime {
				bFirstTime = false
			} else {
				body, err = sjson.Set(body, "search_after", lastSortValue)
				if err != nil {
					return err
				}
			}

			resp, err := e.c.Search().Raw(strings.NewReader(body)).Do(context.Background())
			if err != nil {
				slog.Error("Search Do: " + err.Error())
				return err
			}

			for _, failure := range resp.Shards_.Failures {
				slog.Error("Search Do: " + failure.Reason.Type + ": " + *failure.Reason.Reason)
				return errors.New(failure.Reason.Type)
			}

			if len(resp.Hits.Hits) == 0 {
				slog.Info("no more hits")
				break
			}

			slog.Info(fmt.Sprintf("hits: %d", len(resp.Hits.Hits)))

			for _, hit := range resp.Hits.Hits {
				handler(*hit.Id_, hit.Fields)
			}

			lastSortValue = resp.Hits.Hits[len(resp.Hits.Hits)-1].Sort
		}

		start = endtime
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
