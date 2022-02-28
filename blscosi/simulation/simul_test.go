package main

import (
	"testing"

	simul "github.com/basedfs/simulation"
)

func TestSimulation(t *testing.T) {
	raiseLimit()
	simul.Start("local.toml")
}
