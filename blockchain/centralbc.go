package blockchain

import (
	"bufio"
	"fmt"
	"os"
	//"strings"
)

func Testfileaccess(RoundDuration string) {
	/*_, err := ioutil.ReadFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.csv")
	check(err)
	log.LLvl2(RoundDuration)
	//assert.Equal(t, 7, len(strings.Split(string(csv), "\n")))*/
	f, err := os.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.csv", os.O_APPEND|os.O_WRONLY, 0600)
	check(err)
	defer f.Close()
	_, err = f.WriteString("new data that wasn't there originally\n")
	check(err)

	/*
	data, err := ioutil.ReadFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.csv")
	check(err)
	fmt.Print(string(data))
	*/
	//--------------------
	//f, err = os.Create("/Users/raha/Documents/GitHub/basedfs/simul/platform/testfileaccess.csv")
	//check(err)
	//defer f.Close()
	//--------------------
	w := bufio.NewWriter(f)
	//choose random number for recipe
	//r := rand.New(rand.NewSource(time.Now().UnixNano()))
	//i := r.Perm(5)

	_, err = fmt.Fprintf(w, "%v\n", RoundDuration)
	check(err)
	w.Flush()

	// read and look up (by all virtual nodes in a single host)
	// update a value in that file
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}