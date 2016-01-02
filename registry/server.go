package registry

import (
	"github.com/sayden/go-reverse-proxy/types"
	"net/http"
)

func StartRegistryServer(hostCh chan *types.HostPayload) {
	adderServer := http.NewServeMux()
	adderServer.HandleFunc("/api/v1/host", func(w http.ResponseWriter, r *http.Request) {
		AddHostPostHandler(w, r, hostCh)
	})

	go func() {
		http.ListenAndServe("localhost:49521", adderServer)
	}()
}
