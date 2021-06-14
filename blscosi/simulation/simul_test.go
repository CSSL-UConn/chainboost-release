package main

import (
	"testing"

	"github.com/basedfs/simul"
)

func TestSimulation(t *testing.T) {
	raiseLimit()
	simul.Start("local.toml")
}
