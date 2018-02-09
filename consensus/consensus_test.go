package consensus

import (
	"fmt"
	"time"
)

func ExampleStartProduction() {
	poll := doElection()

	for candidate := range winners {
		startProduction(poll, candidate)
	}
	for candidate := range losers {
		startProduction(poll, candidate)
	}
	time.Sleep(10500 * time.Millisecond)
	// Output:
	//winner
	//second
	//thirdd
	//forthh
	//fifthh
	//winner
	//second
	//thirdd
	//forthh
	//fifthh

}

func startProduction(poll *Poll, candidate string) {
	ttpChan := StartProduction(poll, candidate, 1)
	go func(ttpChan chan struct{}, id string) {
		for range ttpChan {
			fmt.Println(id)
		}
	}(ttpChan, candidate)
}
