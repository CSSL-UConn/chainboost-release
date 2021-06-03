package main

import (
	"testing"

	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/simul"
)

func TestSimulation(t *testing.T) {
	log.SetDebugVisible(1)
	simul.Start("OpinionGathering.toml")
}

/*
func TestSimulation_IndividualStats(t *testing.T) {
	simul.Start("individualstats.toml")
	csv, err := ioutil.ReadFile("test_data/individualstats.csv")
	log.ErrFatal(err)
	// header + 5 rounds + final newline
	assert.Equal(t, 7, len(strings.Split(string(csv), "\n")))

	simul.Start("csv1.toml")
	csv, err = ioutil.ReadFile("test_data/csv1.csv")
	log.ErrFatal(err)
	// header + 2 experiments + final newline
	assert.Equal(t, 4, len(strings.Split(string(csv), "\n")))
}
*/
