package main

import (
	"github.com/pinpt/gitlab/internal"
	"github.com/pinpt/agent/runner"
)

// Integration is used to export the integration
var Integration internal.GitlabIntegration

func main() {
	runner.Main(&Integration)
}
