package proxy

import (
	"fmt"
	"github.com/sayden/go-reverse-proxy/types"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

var addDelayedTime int = 30 //seconds

func GetTargetsLengthWithChannel(hostCh chan *types.HostPayload) int {
	ch := make(chan interface{})
	defer close(ch)
	hostCh <- &types.HostPayload{
		Action:      "length",
		ReceivingCh: ch,
	}

	l := <-ch
	length := l.(int)

	return length
}

func HostsHandler(hostCh chan *types.HostPayload) {
	targets := make([]*url.URL, 0)

	for {
		for p := range hostCh {
			switch p.Action {
			case "get":
				p.TargetsCh <- targets
			case "length":
				p.ReceivingCh <- len(targets)
			case "add":
				//Check if the route exists already
				if stringInSlice(&p.Host, &targets) == false {
					url := getURLFromString(&p.Host)
					targets = append(targets, url)
				}

				log.Printf("After adding, routes are: %s\n", targets)
			case "remove":
				i := -1
				for iter := 0; iter < len(targets); iter++ {
					t := targets[iter]
					if t.Host == p.Host {
						i = iter
					}
				}

				if i != -1 {
					targets = append(targets[:i], targets[i+1:]...)
				}
				log.Printf("After remove, routes are: %s\n", targets)

				//Now add it again after some time
				go addTargetDelayed(&p.Host, hostCh)
			}
		}
	}
}

func addTargetDelayed(t *string, tCh chan *types.HostPayload) {
	fmt.Printf("Adding target %s again after %d seconds\n", *t, addDelayedTime)
	time.Sleep(time.Second * time.Duration(addDelayedTime))
	AddTarget(*t, tCh)
}

// stringInSlice searches if the string 'a' exists in the slice 'list'
func stringInSlice(a *string, list *[]*url.URL) bool {
	for _, b := range *list {
		if b.Host == *a {
			return true
		}
	}
	return false
}

func AddTarget(h string, hostCh chan *types.HostPayload) {
	hostCh <- &types.HostPayload{
		Action: "add",
		Host:   h,
	}
}

func getURLFromString(addr *string) *url.URL {
	return &url.URL{
		Scheme: "http",
		Host:   *addr,
	}
}

// NewMultipleHostReverseProxy creates a reverse proxy that will randomly
// select a host from the passed `targets`
func NewMultipleHostReverseProxy(hostCh chan *types.HostPayload) *httputil.ReverseProxy {

	director := func(req *http.Request) {
		ch := make(chan []*url.URL)
		defer close(ch)
		hostCh <- &types.HostPayload{
			Action:    "get",
			TargetsCh: ch,
		}

		ts := <-ch

		directorHandler(req, ts)
	}

	transport := http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return http.ProxyFromEnvironment(req)
		},

		Dial: func(network, addr string) (net.Conn, error) {
			return dialHandler(network, addr, hostCh)
		},

		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &httputil.ReverseProxy{
		Director:  director,
		Transport: &transport,
	}
}

func directorHandler(req *http.Request, targets []*url.URL) {
	tLength := len(targets)
	ts := targets
	t := ts[rand.Int()%tLength]
	req.URL.Scheme = "http"
	req.URL.Host = t.Host
	req.URL.Path = t.Path
}

func dialHandler(network, addr string, hostCh chan *types.HostPayload) (net.Conn, error) {

	conn := getGoodTarget(hostCh, network)

	return conn, nil
}

func getGoodTarget(hostCh chan *types.HostPayload, network string) net.Conn {
	dial := (&net.Dialer{
		Timeout:   time.Duration(addDelayedTime) * time.Second,
		KeepAlive: time.Duration(addDelayedTime) * time.Second,
	})

	ch := make(chan []*url.URL)
	hostCh <- &types.HostPayload{
		Action:    "get",
		TargetsCh: ch,
	}
	ts := <-ch
	close(ch)

	for {
		i := rand.Int() % len(ts)
		addr := ts[i].Host
		conn, err := dial.Dial(network, addr)
		if err == nil {
			return conn
		}
		removeTarget(&addr, hostCh)
	}
}

func removeTarget(addr *string, hostCh chan *types.HostPayload) {
	fmt.Println("Removing target ", *addr)

	go func() {
		hostCh <- &types.HostPayload{
			Action: "remove",
			Host:   *addr,
		}
	}()
}
