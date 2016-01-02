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
	"sync"
	"time"
)

var targets = make([]*url.URL, 0)
var addDelayedTime int = 5 //seconds
var mutex sync.Mutex

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
					mutex.Lock()
					targets = append(targets, url)
					mutex.Unlock()
				}

				log.Printf("Current routes are: %s\n", targets)
			case "remove":
				newTargets := make([]*url.URL, len(targets)-1)
				i := 0
				for _, t := range targets {
					if t.Host != p.Host {
						newTargets[i] = t
						i++
					}
				}

				mutex.Lock()
				targets = newTargets
				mutex.Unlock()

				//Now add it again after some time
				go addTargetDelayed(&p.Host, hostCh)
			}
		}
	}
}

func addTargetDelayed(t *string, tCh chan *types.HostPayload) {
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
	go func() {
		hostCh <- &types.HostPayload{
			Action: "add",
			Host:   h,
		}
	}()
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
			ch := make(chan []*url.URL)
			defer close(ch)
			hostCh <- &types.HostPayload{
				Action:    "get",
				TargetsCh: ch,
			}
			ts := <-ch
			return dialHandler(network, addr, hostCh, ts)
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
	//	fmt.Printf("CALLING DIRECTOR WITH %d targets\n", tLength)
	ts := targets
	t := ts[rand.Int()%tLength]
	req.URL.Scheme = t.Scheme
	req.URL.Host = t.Host
	req.URL.Path = t.Path
}

func dialHandler(network, addr string, hostCh chan *types.HostPayload, ts []*url.URL) (net.Conn, error) {
	dial := (&net.Dialer{
		Timeout:   time.Duration(addDelayedTime) * time.Second,
		KeepAlive: time.Duration(addDelayedTime) * time.Second,
	})

	conn, err := dial.Dial(network, addr)
	if err != nil {
		println("Error during DIAL:", err.Error())
		removeTarget(&addr, hostCh)
	}

	return conn, err
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
