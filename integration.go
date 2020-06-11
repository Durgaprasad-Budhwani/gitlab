package main

import (
	"github.com/pinpt/agent.next.gitlab/internal"
	"github.com/pinpt/agent.next/runner"
)

// Integration is used to export the integration
var Integration internal.GitlabIntegration

func main() {
	runner.Main(&Integration)
}
