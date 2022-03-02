package main

import (
	"testing"

	simul "github.com/ChainBoost/simulation"
)

func TestSimulation(t *testing.T) {
	simul.Start("count.toml")
}
