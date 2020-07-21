package main

import (
	"bytes"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	//	"regexp"
	"time"
)

var pool *redis.Pool

type HttpHandler struct{}

func newPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		MaxActive:   0,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr) },
	}
}

func getAvailableNodes() []string {
	db := pool.Get()
	defer db.Close()

	res, err := redis.Strings(db.Do("ZRANGE", "available_nodes", 0, 3))
	if err != nil {
		panic(err)
	}
	log.Println("avails: " + fmt.Sprint(res))
	return res
}

func requestBroker(method string, path string, headers http.Header, bodyReader io.Reader) ([]byte, int, http.Header, error) {
	var nodes []string
	var res *http.Response
	body, _ := ioutil.ReadAll(bodyReader)
	contentType := headers.Get("Content-type")
	var url string

	retryWait := 10
	retry := 10
	for i := 0; i < retry; i++ {
		nodes = getAvailableNodes()
		if len(nodes) > 0 {
			break
		}
		log.Printf("No avails, retry %d/%d after waiting for %d sec", i+1, retry, retryWait)
		time.Sleep(time.Second * 10)
	}

	for _, node := range nodes {
		url = node + path
		switch method {
		case "GET":
			res, _ = http.Get(url)
		case "POST":
			res, _ = http.Post(url, contentType, bytes.NewReader(body))
		default:
			return nil, 500, nil, errors.New("Unsupported method " + method)
		}
		if res == nil {
			continue
		}
		if res.StatusCode < 500 {
			break
		}
	}
	if res != nil && res.StatusCode < 500 {
		defer res.Body.Close()
		byteArray, _ := ioutil.ReadAll(res.Body)
		log.Printf("%d %s %s %s req:%s res:%s", res.StatusCode, method, url, contentType, body, byteArray)
		return byteArray, res.StatusCode, res.Header, nil
	}

	// todo: resolve by cache
	return nil, 500, nil, errors.New("Cannot resolved")
}

func ProxyServer(res http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method
	headers := req.Header
	// preflight
	if method == "OPTIONS" {
		res.WriteHeader(http.StatusOK)
		return
	}

	body, code, h, err := requestBroker(method, path, headers, req.Body)
	if err != nil {
		// log error
		log.Println(err)
	}
	res.Header().Set("content-type", h.Get("content-type"))
	res.WriteHeader(code)
	res.Write(body)
}

type Config struct {
	LISTEN string `default:":8080"`
	REDIS  string `default:":6379"`
}

func main() {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalf("Failed to process env: %s", err.Error())
	}

	pool = newPool(config.REDIS)

	http.HandleFunc("/", ProxyServer)
	log.Printf("Listen  %s", config.LISTEN)
	log.Fatal(http.ListenAndServe(config.LISTEN, nil))
}

func shuffle(data []string) {
	n := len(data)
	for i := n - 1; i >= 0; i-- {
		j := rand.Intn(i + 1)
		data[i], data[j] = data[j], data[i]
	}
}
