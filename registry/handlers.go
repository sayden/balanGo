package registry

import (
	"encoding/json"
	"fmt"
	"github.com/sayden/go-reverse-proxy/proxy"
	"github.com/sayden/go-reverse-proxy/types"
	"io/ioutil"
	"log"
	"net/http"
)

// HTTPResponse is an object holding the common information for the register
// service responses
type HTTPResponse struct {
	Status int
	Result string
}

func AddHostPostHandler(w http.ResponseWriter, r *http.Request, hostCh chan *types.HostPayload) {
	w.Header().Add("Content-Type", "application/json")

	if r.Method != "POST" {
		em := "Error: Only POST allowed"
		log.Println(em)
		w.WriteHeader(500)

		j := HTTPResponse{-1, em}
		m, _ := json.Marshal(j)
		w.Write(m)

		return
	}

	defer r.Body.Close()
	bodyByte, err := ioutil.ReadAll(r.Body)
	if err != nil {
		em := "Error trying to read contents of body"
		log.Println(em)
		w.WriteHeader(500)

		j := HTTPResponse{-1, em}
		m, _ := json.Marshal(j)
		w.Write(m)

		return
	}

	var body map[string]string
	if err := json.Unmarshal(bodyByte, &body); err != nil {
		em := "Error parsing body"
		log.Println(em)
		w.WriteHeader(500)

		//Create json
		j := HTTPResponse{-1, em}
		m, _ := json.Marshal(j)

		w.Write(m)
	}

	host := body["host"]
	proxy.AddTarget(host, hostCh)

	msg := fmt.Sprintf("Host %s added if not existing previously", body["host"])
	log.Println(msg)

	j := HTTPResponse{-1, msg}
	ok, _ := json.Marshal(j)

	w.Write(ok)
}
