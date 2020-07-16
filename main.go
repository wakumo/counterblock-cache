/*
CBNODES='https://cb1 https://cb2 https://cb3' LISTEN='localhost:3333' REDIS='localhost:6789' ./counterblock-cache
*/
package main

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	//"github.com/gomodule/redigo/redis"
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

type Config struct {
	CBNODES string `required:"true"`
	LISTEN  string `default:":8080"`
	REDIS   string `default:"localhost:6789"`
}

func main() {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalf("Failed to process env: %s", err.Error())
	}
	nodes = strings.Split(config.CBNODES, " ")
	log.Println("nodes: " + fmt.Sprint(nodes))
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
