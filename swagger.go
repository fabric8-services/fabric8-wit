package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/goadesign/goa"
)

// NewSwaggerController returns a filtered swagger.json file based on request host
func NewSwaggerController() goa.MuxHandler {
	return func(res http.ResponseWriter, req *http.Request, url url.Values) {
		b, err := Asset("swagger/swagger.json")
		if err != nil {
			res.WriteHeader(404)
			return
		}

		s := string(b)
		s = strings.Replace(s, `"host":"demo.api.almighty.io"`, `"host":""`, -1)

		res.Header().Set("Access-Control-Allow-Origin", "*")
		res.Header().Set("Access-Control-Allow-Methods", "GET")

		res.Write([]byte(s))
	}
}
