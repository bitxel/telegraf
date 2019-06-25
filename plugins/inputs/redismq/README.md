# Telegraf Plugin: Redismq

### Configuration:

```
# Read Redis message queue length info
[[inputs.redismq]]
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
```

### Measurements & Fields:

- redismq
    - length(int, number)

### Tags:

- All measurements have the following tags:
    - key
    - db

### Example Output:

Using this configuration:
```
[[inputs.redismq]]
  [[inputs.redismq.queues]]
     server = "tcp://localhost:6379"
     db = 0
     keys = ["redismq::queue_name1", "redismq::queue_name2"]
  [[inputs.redismq.queues]]
     server = "tcp://localhost:6379"
     db = 5
     keys = ["redismq::queue_name3"]
```

When run with:
```
./telegraf --config telegraf.conf --input-filter redismq --test
```

It produces:
```
* Plugin: inputs.redismq, Collection 1
> redis_mq,key=redismq::queue_name1,db=0,host=localhost length=1594i 1504667263000000000
> redis_mq,key=redismq::queue_name2,db=0,host=localhost length=21569i 1504667263000000000
> redis_mq,key=redismq::queue_name3,db=5,host=localhost length=11030i 1504667263000000000
```

### Sample Query:

Get the mean,max,min for length of each queue in the last minute

```
select mean(length),max(length),min(length) from redis_mq where time> now()- 1m group by "key" 
```