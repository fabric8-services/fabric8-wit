package encoding

import (
	"bytes"
	gjson "encoding/json"
	"io"

	"github.com/goadesign/goa"
)

// CustomJSONEncoder creates a std Go JSON Encoder with SetEscapeHTML false
func CustomJSONEncoder(w io.Writer) goa.Encoder {
	e := gjson.NewEncoder(unicodeDecoder{target: w})
	//e.SetEscapeHTML(false) // Require go 1.7
	return e
}

type unicodeDecoder struct {
	target io.Writer
}

func (u unicodeDecoder) Write(p []byte) (n int, err error) {
	b := p
	b = bytes.Replace(b, []byte("\\u003c"), []byte("<"), -1)
	b = bytes.Replace(b, []byte("\\u003e"), []byte(">"), -1)
	b = bytes.Replace(b, []byte("\\u0026"), []byte("&"), -1)
	return u.target.Write(b)
}
