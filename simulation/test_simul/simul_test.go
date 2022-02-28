package main

import (
	"testing"

	simul "github.com/basedfs/simulation"
)

func TestSimulation(t *testing.T) {
	simul.Start("count.toml")
}
