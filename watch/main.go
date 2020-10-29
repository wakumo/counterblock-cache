package main

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/kelseyhightower/envconfig"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

var nodes []string
var pool *redis.Pool

type Config struct {
	TIMEOUT   int    `default:"5"`
	CBNODES   string `required:"true"`
	REDIS     string `default:"localhost:6379"`
	REDIS_URL string `default:"redis://localhost:6379"`
}

var config Config

func newPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		MaxActive:   0,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr) },
	}
}

func getDb() redis.Conn {
	db := pool.Get()
	db.Do("SELECT", 0)
	return db
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / 1e6
}

func checkAvailableNodes() {
	var wg sync.WaitGroup
	path := "/"

	for _, node := range nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			db := getDb()
			defer db.Close()

			url := node + path
			begin := makeTimestamp()
			client := http.Client{
				Timeout: time.Duration(config.TIMEOUT) * time.Second,
			}
			res, _ := client.Get(url)
			end := makeTimestamp()
			score := end - begin
			if res != nil {
				log.Printf("%s %d %d", node, res.StatusCode, score)
				if res.StatusCode == 200 {
					db.Do("ZADD", "available_nodes", score, node)
				} else {
					db.Do("ZREM", "available_nodes", node)
				}
			} else {
				log.Printf("%s timeout > %d", node, config.TIMEOUT)
				db.Do("ZREM", "available_nodes", node)
			}
		}(node)
	}
	wg.Wait()
}

func showAvailableNodes() {
	db := getDb()
	defer db.Close()

	res, err := redis.Strings(db.Do("ZRANGE", "available_nodes", 0, 3))
	if err != nil {
		panic(err)
	}
	log.Println("avails: " + fmt.Sprint(res))
}

func initAvailableNodes() {
	// remove all scores
	db := getDb()
	defer db.Close()
	db.Do("DEL", "available_nodes")

}

func main() {
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalf("Failed to process env: %s", err.Error())
	}

	nodes = strings.Split(config.CBNODES, " ")
	log.Println("nodes: " + fmt.Sprint(nodes))

	pool = newPool(config.REDIS)
	initAvailableNodes()
	for {
		checkAvailableNodes()
		showAvailableNodes()
		time.Sleep(10 * time.Second)
	}
}
