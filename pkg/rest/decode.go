package rest

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strings"

	"github.com/clbanning/mxj/v2"
)

// decodeResponse decodes the body of a successful response into
// response, based on the content-type. response must be a non-nil
// pointer. A response without content (204 and 205) leaves response
// untouched, as does an empty body that has no content-type, but an
// empty body that claims to be JSON or XML is an error.
func decodeResponse(resp *http.Response, response any) error {
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusResetContent:
		return nil
	}

	ct := mediaType(resp.Header.Get("Content-Type"))
	if ct == "" && resp.ContentLength == 0 {
		return nil
	}

	switch {
	case isJSON(ct):
		return decodeJSON(resp.Body, ct, response)
	case isXML(ct):
		return decodeXML(resp.Body, ct, response)
	case strings.HasPrefix(ct, "text/"):
		return decodeText(resp.Body, response)
	}
	return fmt.Errorf("%w: response content-type %q", errors.ErrUnsupported, ct)
}

// mediaType returns the lower-case media type from a Content-Type
// header value, without any parameters such as the charset
func mediaType(header string) string {
	mt, _, err := mime.ParseMediaType(header)
	if err != nil {
		mt, _, _ = strings.Cut(header, ";")
		mt = strings.ToLower(strings.TrimSpace(mt))
	}
	return mt
}

func isJSON(ct string) bool {
	return ct == "application/json" || ct == "text/json" || strings.HasSuffix(ct, "+json")
}

func isXML(ct string) bool {
	return ct == "application/xml" || ct == "text/xml" || strings.HasSuffix(ct, "+xml")
}

func decodeJSON(body io.Reader, ct string, response any) error {
	if err := checkPointer(response); err != nil {
		return err
	}
	err := json.NewDecoder(body).Decode(response)
	if err == io.EOF {
		return fmt.Errorf("rest: empty %s response body: %w", ct, io.EOF)
	}
	return err
}

func decodeXML(body io.Reader, ct string, response any) error {
	if err := checkPointer(response); err != nil {
		return err
	}
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return fmt.Errorf("rest: empty %s response body: %w", ct, io.EOF)
	}
	switch rt := response.(type) {
	case *mxj.Map:
		mv, err := mxj.NewMapXml(b)
		if err != nil {
			return err
		}
		*rt = mv
		return nil
	case *map[string]any:
		mv, err := mxj.NewMapXml(b)
		if err != nil {
			return err
		}
		*rt = mv
		return nil
	}
	return xml.Unmarshal(b, response)
}

func decodeText(body io.Reader, response any) error {
	switch rt := response.(type) {
	case *[]byte:
		b, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		*rt = b
	case *string:
		b, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		*rt = string(b)
	default:
		return http.ErrNotSupported
	}
	return nil
}

// checkPointer returns an error unless response is a non-nil pointer,
// as decoding into anything else can never be seen by the caller
func checkPointer(response any) error {
	if rv := reflect.ValueOf(response); rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("rest: response must be a non-nil pointer, not %T", response)
	}
	return nil
}
