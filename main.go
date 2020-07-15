/*
  cbc
	[--redis-host=127.0.0.0]
	[--redis-port=6379]
	https://cb1.server [https://cb2.server [https://cb3.server...]]


  https://golang.org/pkg/net/http/#Request
  https://golang.org/pkg/net/http/#Response
*/
package main

import (
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	//"github.com/gomodule/redigo/redis"
)

var (
	address    = flag.String("l", "", "Listen address")
	port       = flag.Int("p", 8080, "Listen port")
	redis_host = flag.String("redis-host", "", "redis host")
	redis_port = flag.Int("redis-port", 6379, "redis port")
)

var nodes []string

type HttpHandler struct{}

func requestBroker(method string, path string, contentType string, body io.Reader) ([]byte, int, error) {
	var res *http.Response

	shuffle(nodes)

	for _, node := range nodes {
		url := node + path
		switch method {
		case "GET":
			res, _ = http.Get(url)
		case "POST":
			res, _ = http.Post(url, contentType, body)
		default:
			return nil, 500, errors.New("Unsupported method " + method)
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

		return byteArray, res.StatusCode, nil
	}

	// todo: resolve by cache
	return nil, 500, errors.New("Cannot resolved")
}

func (h HttpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method
	contentType := req.Header.Get("Content-type")
	body, code, err := requestBroker(method, path, contentType, req.Body)
	if err != nil {
		// log error
		log.Println(err)
	}
	res.WriteHeader(code)
	res.Write(body)
}

func ProxyServer(res http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method
	contentType := req.Header.Get("Content-type")
	body, code, err := requestBroker(method, path, contentType, req.Body)
	if err != nil {
		// log error
		log.Println(err)
	}
	res.WriteHeader(code)
	res.Write(body)
}

func main() {
	flag.Parse()
	nodes = flag.Args()
	log.Println("nodes: " + fmt.Sprint(nodes))
	listen := fmt.Sprintf("%s:%d", *address, *port)
	http.HandleFunc("/", ProxyServer)
	log.Printf("Listen  %s", listen)
	log.Fatal(http.ListenAndServe(listen, nil))
}

func shuffle(data []string) {
	n := len(data)
	for i := n - 1; i >= 0; i-- {
		j := rand.Intn(i + 1)
		data[i], data[j] = data[j], data[i]
	}
}
