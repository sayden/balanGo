package types

import "net/url"

//HostPayload is used to group and action to perform to a host (add, delete...)
type HostPayload struct {
	Action string
	Host   string
	Ch     chan []*url.URL
}
