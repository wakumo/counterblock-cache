package main

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/kelseyhightower/envconfig"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

var nodes []string

type Config struct {
	CBNODES string `required:"true"`
	REDIS   string `default:"localhost:6379"`
}

func shuffle(data []string) {
	n := len(data)
	for i := n - 1; i >= 0; i-- {
		j := rand.Intn(i + 1)
		data[i], data[j] = data[j], data[i]
	}
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / 1e6
}

func checkAvailableNodes(db redis.Conn) {
	shuffle(nodes)
	path := "/"
	for _, node := range nodes {
		url := node + path
		begin := makeTimestamp()
		res, _ := http.Get(url)
		end := makeTimestamp()
		score := end - begin
		log.Printf("%s %d %d", node, res.StatusCode, score)
		if res.StatusCode == 200 {
			db.Do("ZADD", "available_nodes", score, node)
		} else {
			db.Do("ZREM", "available_nodes", node)
		}
	}
}

func showAvailableNodes(db redis.Conn) {
	res, err := redis.Strings(db.Do("ZRANGE", "available_nodes", 0, 3))
	if err != nil {
		panic(err)
	}
	log.Println("avails: " + fmt.Sprint(res))
}

func main() {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalf("Failed to process env: %s", err.Error())
	}

	nodes = strings.Split(config.CBNODES, " ")
	log.Println("nodes: " + fmt.Sprint(nodes))

	db, err := redis.Dial("tcp", config.REDIS)
	if err != nil {
		panic(err)
	}

	for {
		checkAvailableNodes(db)
		showAvailableNodes(db)
		time.Sleep(10 * time.Second)
	}
}
