package consensus

import (
	"fmt"
	"github.com/spf13/viper"
	"testing"
	"time"
)

func ExampleStartProduction() {
	viper.Set("poll.period", int64(time.Second))
	poll := doElection()

	for candidate := range winners {
		startProduction(poll, candidate)
	}
	for candidate := range losers {
		startProduction(poll, candidate)
	}

	time.Sleep(10100 * time.Millisecond)
	// Output:
	//forthh
	//fifthh
	//winner
	//second
	//thirdd
	//forthh
	//fifthh
	//winner
	//second
	//thirdd
}

func startProduction(poll *Poll, candidate string) {
	ttpChan := StartProduction(poll, candidate)
	go func(ttpChan chan struct{}, id string) {
		for range ttpChan {
			fmt.Println(id)
		}
	}(ttpChan, candidate)
}

func TestPoll_GetCurrentSlot(t *testing.T) {
	startTime := getTestStartTime()
	poll := NewPoll(5, 1, 0, startTime)
	poll.period = int64(1 * time.Second)

	expected := []int{2, 3, 4, 4, 0, 1, 1, 2, 3}

	for i := 0; i < 9; i++ {
		slot := poll.getSlotAt(time.Now().UTC().UnixNano())
		if slot != expected[i] {
			t.Errorf("Got %d slot, expected: %d", slot, expected[i])
		}
		time.Sleep(700 * time.Millisecond)
	}

}
