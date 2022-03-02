package main

import (
	"testing"

	simul "github.com/chainBoostScale/ChainBoost/simulation"
)

func TestSimulation(t *testing.T) {
	raiseLimit()
	simul.Start("local.toml")
}
