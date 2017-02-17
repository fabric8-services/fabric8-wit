package rest

import (
	"fmt"

	"github.com/goadesign/goa"
)

// AbsoluteURL prefixes a relative URL with absolute address
func AbsoluteURL(req *goa.RequestData, relative string) string {
	scheme := "http"
	if req.TLS != nil { // isHTTPS
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, req.Host, relative)
}
