package main

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
)

func loadDeployment(path string) (dep lbdeploy.Deployment, err error) {
	if path == "" {
		return dep, errors.New("missing deployment configuraiton file path")
	}
	if !strings.HasSuffix(path, "deploy.json") {
		return dep, errors.New("the provided deployment file path must end in deploy.json")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return dep, err
	}
	err = json.Unmarshal(data, &dep)
	return
}
