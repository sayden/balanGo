package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sayden/go-reverse-proxy/proxy"
	"github.com/sayden/go-reverse-proxy/types"
)

var hostCh = make(chan *types.HostPayload, 1)

func startBalancer(args []string) {
	if len(args) < 3 {
		log.Fatalf("Usage: %s <port to listen> <host to route to> <host to route to>...", os.Args[0])
	}

	listeningPort := args[1]
	args = args[2:]

	//GoRoutine with the host setter handler
	go proxy.HostsHandler(hostCh)

	for _, host := range args {
		proxy.AddTarget(host, hostCh)
	}

	// Create the server that adds new hosts to the balancer
	adderServer := http.NewServeMux()
	adderServer.HandleFunc("/api/v1/host", addHostPostHandler)

	go func() {
		http.ListenAndServe("localhost:49521", adderServer)
	}()

	for len(proxy.GetTargets()) == 0 {
		time.Sleep(time.Millisecond)
	}
	proxy := proxy.NewMultipleHostReverseProxy(hostCh)
	log.Fatal(http.ListenAndServe(":"+listeningPort, proxy))
}

func main() {
	startBalancer(os.Args)
}

func removeHost(seconds int, addr *string) {
	fmt.Println(*addr)
}

// addHostPostHandler is the POST handler that will add a new host to the
// balancer's pool. The JSON format is the following:
// { host: "a_host:a_port" }
func addHostPostHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Add("Content-Type", "application/json")

	if r.Method != "POST" {
		log.Println("Error: Only POST allowed")
		w.WriteHeader(500)
		w.Write([]byte("{\"status\":-1, \"result\":\"Error: Only POST is allowed\"}"))
		return
	}

	defer r.Body.Close()
	bodyByte, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Error trying to parse body")
		w.WriteHeader(500)
		w.Write([]byte("{\"status\":-1, \"result\":\"Error trying to parse body\"}"))
		return
	}
	var body map[string]string

	if err := json.Unmarshal(bodyByte, &body); err != nil {
		log.Println("Error parsing body")
		w.WriteHeader(500)
		w.Write([]byte("{\"status\":-1, \"result\":\"Error parsing body\"}"))
	}

	host := body["host"]
	proxy.AddTarget(host, hostCh)

	msg := fmt.Sprintf("Host %s added if not existing previously", body["host"])
	log.Println(msg)

	w.Write([]byte("{\"status\":0, \"result\":\"" + msg + "\"}"))
}
