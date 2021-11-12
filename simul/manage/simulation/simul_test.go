package main

import (
	"testing"

	"github.com/basedfs/log"
	"github.com/basedfs/simul"
)

func TestSimulation(t *testing.T) {
	log.SetDebugVisible(2)
	simul.Start("BaseDFS.toml")
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
