package data

import (
	"storage/internal/conf"

	"github.com/elastic/go-elasticsearch/v9"
)

type ESClient struct {
	*elasticsearch.Client
}

func NewESClient(conf *conf.Data) (*ESClient, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{
			conf.Elasticsearch.Addr,
		},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &ESClient{es}, nil
}
