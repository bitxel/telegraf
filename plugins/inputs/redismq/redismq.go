package redismq

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"regexp"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/pkg/errors"
)

// RedisMQ defines the configuration for redis message queues
type RedisMQ struct {
	Queues []Queue
}

// Queue is a configuration for one message queue
type Queue struct {
	Server string
	DB     int
	Keys   []string
}

var sampleConfig = `
  ## specify a list of message queue
  ## [[inputs.redismq.queues]]
  ##    server = "tcp://localhost:6379"
  ##    db = 0
  ##    keys = ["queue_name1"]
  ## [[inputs.redismq.queues]]
  ##    server = "tcp://localhost:6379"
  ##    db = 5
  ##    keys = ["queue_name2"]
  ##
  ## specify servers via a url matching:
  ##  [protocol://][:password]@address[:port]
  ##  e.g.
  ##    tcp://localhost:6379
  ##    tcp://:password@192.168.99.100
  ##    unix:///var/run/redis.sock
  ##
  ## db refers to the index of redis db, default 0.
  ## key defines the queues to be measured.
  ##
  [[inputs.redismq.queues]]
     server = "tcp://localhost:6379"
     db = 0
     keys = ["queue_name1"]
  [[inputs.redismq.queues]]
     server = "tcp://localhost:6379"
     db = 5
     keys = ["queue_name2"]
`

const defaultPort = "6379"

var (
	defaultTimeout   = 5 * time.Second
	ErrInvalidKey    = errors.New("key name is invalid")
	ErrProtocolError = errors.New("redis protocol error")
)

func (r *RedisMQ) SampleConfig() string {
	return sampleConfig
}

func (r *RedisMQ) Description() string {
	return "Read metrics from one or many redis message queues"
}

// Gather stats from all configured message queues.
func (r *RedisMQ) Gather(acc telegraf.Accumulator) error {
	var wg sync.WaitGroup
	for _, queue := range r.Queues {
		wg.Add(1)
		go func(queue Queue) {
			defer wg.Done()
			if len(queue.Keys) == 0 {
				acc.AddError(ErrInvalidKey)
				return
			}
			acc.AddError(r.gatherQueue(queue, acc))
		}(queue)
	}

	wg.Wait()
	return nil
}

func (r *RedisMQ) gatherQueue(queue Queue, acc telegraf.Accumulator) error {
	var address string
	serv := queue.Server
	if !strings.HasPrefix(serv, "tcp://") && !strings.HasPrefix(serv, "unix://") {
		serv = "tcp://" + serv
	}

	u, err := url.Parse(serv)
	if err != nil {
		acc.AddError(fmt.Errorf("Unable to parse to address '%s': %s", serv, err))
		return err
	} else if u.Scheme == "" {
		// fallback to simple string based address (i.e. "10.0.0.1:10000")
		u.Scheme = "tcp"
		u.Host = serv
		u.Path = ""
	}
	if u.Scheme == "tcp" {
		_, _, err := net.SplitHostPort(u.Host)
		if err != nil {
			u.Host = u.Host + ":" + defaultPort
		}
		address = u.Host
	} else {
		address = u.Path
	}

	c, err := net.DialTimeout(u.Scheme, address, defaultTimeout)
	if err != nil {
		return fmt.Errorf("Unable to connect to redis server '%s': %s", address, err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(defaultTimeout))

	if u.User != nil {
		pwd, set := u.User.Password()
		if set && pwd != "" {
			c.Write([]byte(fmt.Sprintf("AUTH %s\r\n", pwd)))

			rdr := bufio.NewReader(c)

			line, err := rdr.ReadString('\n')
			if err != nil {
				return err
			}
			if line[0] != '+' {
				return fmt.Errorf("%s", strings.TrimSpace(line)[1:])
			}
		}
	}

	rdr := bufio.NewReader(c)
	c.Write([]byte(fmt.Sprintf("SELECT %d\r\n", queue.DB)))
	if _, err = rdr.ReadString('\n'); err != nil {
		return err
	}

	for _, key := range queue.Keys {
		c.Write([]byte(fmt.Sprintf("LLEN %s\r\n", key)))

		real_key := key
		re := regexp.MustCompile("_[0-9]+$")
		if re.FindStringIndex(key) != nil {
			idx := strings.LastIndex(key, "_")
			if idx != -1 {
				key = key[:idx]
			}
		}

		tags := map[string]string{"key": key, "db": strconv.Itoa(queue.DB), "real_key": real_key, "redis_addr": u.Host}
		gatherQueueOutput(rdr, acc, tags)
	}

	return nil
}

func gatherQueueOutput(
	rdr *bufio.Reader,
	acc telegraf.Accumulator,
	tags map[string]string,
) (err error) {
	scanner := bufio.NewScanner(rdr)
	fields := make(map[string]interface{})
	if !scanner.Scan() {
		return scanner.Err()
	}
	line := scanner.Text()
	if strings.Contains(line, "ERR") {
		return ErrProtocolError
	}

	if len(line) == 0 {
		return ErrProtocolError
	}
	if line[0] == ':' {
		if length, err := strconv.ParseInt(line[1:], 10, 64); err == nil {
			fields["length"] = length
		}
	}
	acc.AddFields("redis_mq", fields, tags)
	return
}

func init() {
	inputs.Add("redismq", func() telegraf.Input {
		return &RedisMQ{}
	})
}
