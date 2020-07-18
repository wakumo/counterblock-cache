/*
CBNODES='https://cb1 https://cb2 https://cb3' LISTEN='localhost:3333' REDIS='localhost:6789' ./counterblock-cache
*/
package main

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"regexp"
)

var db redis.Conn
var nodes []string

type HttpHandler struct{}

func getAvailableNodes() []string {
	res, err := redis.Strings(db.Do("ZRANGE", "available_nodes", 0, 3))
	if err != nil {
		panic(err)
	}
	log.Println("avails: " + fmt.Sprint(res))
	return res
}

func requestBroker(method string, path string, headers http.Header, body io.Reader) ([]byte, int, http.Header, error) {
	var res *http.Response
	contentType := headers.Get("Content-type")
	nodes = getAvailableNodes()

	for _, node := range nodes {
		url := node + path
		switch method {
		case "GET":
			res, _ = http.Get(url)
		case "POST":
			res, _ = http.Post(url, contentType, body)
		default:
			return nil, 500, nil, errors.New("Unsupported method " + method)
		}
		if res == nil {
			continue
		}
		log.Printf("%d %s %s", res.StatusCode, method, url)
		if res.StatusCode < 500 {
			break
		}
	}
	if res != nil && res.StatusCode < 500 {
		defer res.Body.Close()
		byteArray, _ := ioutil.ReadAll(res.Body)

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

	body, code, res_headers, err := requestBroker(method, path, headers, req.Body)
	if err != nil {
		// log error
		log.Println(err)
	}

	for k, h := range res_headers {
		// filter headers
		r := regexp.MustCompile(`^[Aa]ccess-[Cc]ontrol-.*`)
		if !r.MatchString(k) {
			res.Header().Set(k, h[0])
		}
	}
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

	c, err := redis.Dial("tcp", config.REDIS)
	if err != nil {
		panic(err)
	}
	db = c

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
