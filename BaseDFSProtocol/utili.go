package BaseDFSProtocol
//from:  https://rosettacode.org/wiki/Statistics/Normal_distribution#Go
import (
	"math"
	"math/rand"
)

// Box-Muller
func norm2() (s, c float64) {
	r := math.Sqrt(-2 * math.Log(rand.Float64()))
	s, c = math.Sincos(2 * math.Pi * rand.Float64())
	return s * r, c * r
}