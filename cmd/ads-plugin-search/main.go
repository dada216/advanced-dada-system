package main

import (
	"encoding/json"
	"fmt"

	"github.com/advanced-dada-system/ads/internal/plugin"
	"github.com/advanced-dada-system/ads/internal/search"
	hashicorpplugin "github.com/hashicorp/go-plugin"
)

type SearchPlugin struct{}

func (s *SearchPlugin) RunTask(args map[string]string) (string, error) {
	query, ok := args["query"]
	if !ok {
		return "", fmt.Errorf("missing 'query' argument")
	}

	results, err := search.Query(query)
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(results)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func main() {
	hashicorpplugin.Serve(&hashicorpplugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig,
		Plugins: map[string]hashicorpplugin.Plugin{
			"service": &plugin.ServicePlugin{Impl: &SearchPlugin{}},
		},
	})
}
