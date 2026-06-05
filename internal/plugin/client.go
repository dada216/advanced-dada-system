package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/advanced-dada-system/ads/internal/search"
	hashicorpplugin "github.com/hashicorp/go-plugin"
)

func CallPlugin(pluginName string, args map[string]string) (string, error) {
	// Try finding the plugin next to the current executable first
	exePath, err := os.Executable()
	var pluginPath string
	if err == nil {
		pluginPath = filepath.Join(filepath.Dir(exePath), pluginName)
		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			pluginPath = ""
		}
	}

	if pluginPath == "" {
		pluginPath, _ = exec.LookPath(pluginName)
	}
	if pluginPath == "" {
		// Fallback for development inside the project root
		pluginPath = "./bin/" + pluginName
	}

	client := hashicorpplugin.NewClient(&hashicorpplugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins: map[string]hashicorpplugin.Plugin{
			"service": &ServicePlugin{},
		},
		Cmd: exec.Command(pluginPath),
	})
	defer client.Kill()

	rpcClient, err := client.Client()
	if err != nil {
		return "", fmt.Errorf("failed to start plugin %s: %w", pluginName, err)
	}

	raw, err := rpcClient.Dispense("service")
	if err != nil {
		return "", err
	}

	service := raw.(Service)
	return service.RunTask(args)
}

func CallSearchPlugin(query string) ([]search.Result, error) {
	resStr, err := CallPlugin("ads-plugin-search", map[string]string{"query": query})
	if err != nil {
		return nil, err
	}

	var results []search.Result
	if resStr != "" {
		if err := json.Unmarshal([]byte(resStr), &results); err != nil {
			return nil, err
		}
	}
	return results, nil
}
