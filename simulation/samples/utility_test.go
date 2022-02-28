package sample

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestNormalDistribution(t *testing.T) {
	const (
		n     = 10000
		bins  = 12
		sig   = 3
		scale = 100
	)
	var sum, sumSq float64
	h := make([]int, bins)
	for i, accum := 0, func(v float64) {
		sum += v
		sumSq += v * v
		b := int((v + sig) * bins / sig / 2)
		if b >= 0 && b < bins {
			h[b]++
		}
	}; i < n/2; i++ {
		v1, v2 := norm2()
		accum(v1)
		accum(v2)
	}
	m := sum / n
	fmt.Println("mean:", m)
	fmt.Println("stddev:", math.Sqrt(sumSq/float64(n)-m*m))
	for _, p := range h {
		fmt.Println(strings.Repeat("*", p/scale))
	}
	var samplearr []float64
	desiredStdDev := float64(2)
	desiredMean := float64(10)
	for i := 0; i < 20; i++ {
		samplearr = append(samplearr, rand.NormFloat64()*desiredStdDev+desiredMean)
	}
	sort.Float64s(samplearr)
	for i := 0; i < 20; i++ {
		fmt.Println(strings.Repeat("*", int(samplearr[i])))
	}
	// ------ from: https://rosettacode.org/wiki/Random_numbers#Go
	const mean = 1.0
	const derivation = .5
	const n2 = 1000

	var list [n2]float64
	rand.Seed(time.Now().UnixNano())
	for i := range list {
		list[i] = mean + derivation*rand.NormFloat64()
	}
}

func TestTime(t *testing.T) {
	// Creating channel using make keyword
	mychan1 := make(chan string, 2)
	// Calling Sleep function of go
	go func() {
		t := time.Now()
		fmt.Println("start:", t.String())
		time.Sleep(2 * time.Second)
		// Displayed after sleep overs
		mychan1 <- "output: after two seconds"
	}()
	// Select statement
	select {
	// Case statement
	case out := <-mychan1:
		t := time.Now()
		fmt.Println(t.String(), "---", out)
	// Calling After method
	case <-time.After(3 * time.Second):
		t := time.Now()
		fmt.Println(t.String(), "---", "timeout: after three seconds")
	}
	// Again Creating channel using make keyword
	mychan2 := make(chan string, 2)
	// Calling Sleep method of go
	go func() {
		time.Sleep(6 * time.Second)
		// Printed after sleep overs
		mychan2 <- "output2"
	}()
	// Select statement
	select {
	// Case statement
	case out := <-mychan2:
		t := time.Now()
		fmt.Println(t.String(), "-----", out)
		// Calling After method
	case <-time.After(3 * time.Second):
		t := time.Now()
		fmt.Println(t.String(), "---", "timeout: after three seconds")
	}
}
