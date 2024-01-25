package esx

import (
	"fmt"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
)

func TestESNew(t *testing.T) {
	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://10.177.42.239:9200",
		},
		Username: "elastic",
		Password: "riYbTNQWTFawDgDzFjt2",
	}

	es, err := New(cfg)
	if err != nil {
		panic(err)
	}

	err = es.SearchWithRaw("gsplane*", jsonBody, func(v string) {
		fmt.Println(v)
	})
	if err != nil {
		panic(err)
	}
}

var jsonBody = `{
	"track_total_hits": false,
	"sort": [
	  {
		"@timestamp": {
		  "order": "desc",
		  "unmapped_type": "boolean"
		}
	  }
	],
	"fields": [
	  {
		"field": "*",
		"include_unmapped": "true"
	  },
	  {
		"field": "@timestamp",
		"format": "strict_date_optional_time"
	  },
	  {
		"field": "LogEndDate",
		"format": "strict_date_optional_time"
	  },
	  {
		"field": "LogStartDate",
		"format": "strict_date_optional_time"
	  },
	  {
		"field": "ts",
		"format": "strict_date_optional_time"
	  },
	  {
		"field": "user.created_at",
		"format": "strict_date_optional_time"
	  }
	],
	"size": 500,
	"version": true,
	"script_fields": {},
	"stored_fields": [
	  "*"
	],
	"runtime_mappings": {},
	"_source": false,
	"query": {
	  "bool": {
		"must": [],
		"filter": [
		  {
			"range": {
			  "@timestamp": {
				"format": "strict_date_optional_time",
				"gte": "2024-01-25T01:22:14.566Z",
				"lte": "2024-01-25T02:22:14.566Z"
			  }
			}
		  },
		  {
			"match_phrase": {
			  "facility": "dianping"
			}
		  }
		],
		"should": [],
		"must_not": []
	  }
	},
	"highlight": {
	  "pre_tags": [
		"@kibana-highlighted-field@"
	  ],
	  "post_tags": [
		"@/kibana-highlighted-field@"
	  ],
	  "fields": {
		"*": {}
	  },
	  "fragment_size": 2147483647
	}
  }`
