package main

import (
	"github.com/pinpt/agent/v4/runner"
	"github.com/pinpt/gitlab/internal"
)

// Integration is used to export the integration
var Integration internal.GitlabIntegration

func main() {
	runner.Main(&Integration)
}
