package redismq

import (
	"bufio"
	"strings"
	"testing"

	"fmt"

	"github.com/influxdata/telegraf/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRedisMQ_SendCmd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	addr := fmt.Sprintf(testutil.GetLocalHost() + ":6379")

	r := &RedisMQ{
		Queues: []Queue{
			{
				Server: addr,
				DB:     0,
				Keys:   []string{"redismq::queue_0"},
			},
			{
				Server: addr,
				DB:     5,
				Keys:   []string{"redismq::queue_1}"},
			},
		},
	}

	var acc testutil.Accumulator

	err := acc.GatherError(r.Gather)
	assert.NoError(t, err)
	assert.Equal(t, len(r.Queues), acc.NFields())
}

func TestRedisMQ_gatherQueue(t *testing.T) {
	var err error
	var acc testutil.Accumulator
	tags := map[string]string{"host": "localhost", "key": "testkey"}

	rdr := bufio.NewReader(strings.NewReader(testCorrectOutput))
	err = gatherQueueOutput(rdr, &acc, tags)
	assert.NoError(t, err)

	rdr = bufio.NewReader(strings.NewReader(testIncorrectOutput))
	err = gatherQueueOutput(rdr, &acc, tags)
	assert.Error(t, err)
}

const (
	testCorrectOutput   = ":21561"
	testIncorrectOutput = "-ERR unknown command 'eof'"
)
