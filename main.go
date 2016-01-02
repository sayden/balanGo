package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sayden/go-reverse-proxy/proxy"
	"github.com/sayden/go-reverse-proxy/registry"
	"github.com/sayden/go-reverse-proxy/types"
)

var hostCh = make(chan *types.HostPayload, 1)

func main() {
	startBalancer(os.Args)
}

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
	registry.StartRegistryServer(hostCh)

	for proxy.GetTargetsLengthWithChannel(hostCh) == 0 {
		time.Sleep(time.Second)
	}
	proxy := proxy.NewMultipleHostReverseProxy(hostCh)
	log.Fatal(http.ListenAndServe(":"+listeningPort, proxy))
}
