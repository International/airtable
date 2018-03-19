// Package airtable provides a high-level client to the Airtable API
// that allows the consumer to drop to a low-level request client when
// needed.
package airtable

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
)

const (
	defaultRootURL = "https://api.airtable.com"
	defaultVersion = "v0"
)

func makeURL(path string) {

}

// Client represents an interface to communicate with the Airtable API
type Client struct {
	APIKey  string
	BaseID  string
	Version string
	RootURL string
}

// ErrClientSetupError is returned when the client is missing APIKey
// or BaseID
type ErrClientSetupError struct {
	msg string
}

func (e ErrClientSetupError) Error() string {
	return e.msg
}

// ErrClientRequestError is returned when the client runs into
// problems with a request
type ErrClientRequestError struct {
	msg string
}

func (e ErrClientRequestError) Error() string {
	return e.msg
}

func (c *Client) checkSetup() error {
	if c.BaseID == "" {
		return ErrClientSetupError{"Client missing BaseID"}
	}
	if c.APIKey == "" {
		return ErrClientSetupError{"Client missing APIKey"}
	}
	if c.Version == "" {
		c.Version = defaultVersion
	}
	if c.RootURL == "" {
		c.RootURL = defaultRootURL
	}
	return nil
}

func (c *Client) makeURL(resource string, options QueryEncoder) string {
	q := options.Encode()
	url := fmt.Sprintf("%s/%s/%s/%s?%s",
		c.RootURL, c.Version, c.BaseID, resource, q)
	return url
}

// QueryEncoder encodes options to a query string
type QueryEncoder interface {
	Encode() string
}

type errorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func checkErrorResponse(b []byte) error {
	var reqerr errorResponse
	if jsonerr := json.Unmarshal(b, &reqerr); jsonerr != nil {
		return jsonerr
	}
	if reqerr.Error.Type != "" {
		return ErrClientRequestError{reqerr.Error.Message}
	}
	return nil
}

/* Field Types */

// Rating ...
type Rating int

// Text ...
type Text string

// LongText ...
type LongText string

// AttachmentThumbnail ...
type AttachmentThumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Attachment ...
type Attachment struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Filename   string `json:"filename"`
	Size       int    `json:"size"`
	Type       string `json:"type"`
	Thumbnails struct {
		Small AttachmentThumbnail `json:"small"`
		Large AttachmentThumbnail `json:"large"`
	} `json:"thumnbnails"`
}

// Checkbox ...
type Checkbox bool

// MultipleSelect ...
type MultipleSelect []string

// Date ...
type Date string

// FormulaResult ...
type FormulaResult interface {
	// TODO: represent string/int for successful formula, some error
	// sentinel for unsuccessful
}

// RecordLink ...
type RecordLink []string

// SingleSelect ...
type SingleSelect string

// GetResponse contains the response from requesting a resource
type GetResponse struct {
	ID          string                 `json:"id"`
	Fields      map[string]interface{} `json:"fields"`
	CreatedTime string                 `json:"createdTime"`
}

// Get returns information about a resource
func (r *Resource) Get(id string, options QueryEncoder) (*GetResponse, error) {
	fullid := r.name + "/" + id
	bytes, err := r.client.RequestBytes(fullid, options)
	if err != nil {
		return nil, err
	}

	var resp GetResponse
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return nil, err
	}

	// record comes in as an `interface {}` so let's get a pointer for
	// it and unwrap until we can get a value for the underlying struct
	refPtrToStruct := reflect.ValueOf(&r.record).Elem()
	structAsInterface := refPtrToStruct.Interface()
	refStruct := reflect.ValueOf(structAsInterface).Elem()
	refStructType := refStruct.Type()

	for i := 0; i < refStruct.NumField(); i++ {
		f := refStruct.Field(i)
		fType := refStructType.Field(i)

		key := fType.Name
		if from, ok := fType.Tag.Lookup("from"); ok {
			key = from
		}

		// TODO: confirm it fits
		if value := resp.Fields[key]; value != nil {
			switch f.Kind() {
			case reflect.Slice:
				fmt.Println(reflect.TypeOf(value))
				// s, ok := value.([]string)
				// if !ok {
				// 	panic("could not assert value as slice")
				// }
				// f.Set()
				fmt.Printf("%v (slice)", key)
			case reflect.String:
				s, ok := value.(string)
				if !ok {
					panic("could not assert value as string")
				}
				f.SetString(s)
				fmt.Printf("%v (string)", key)
			case reflect.Int:
				fmt.Printf("%v (int)", key)
			case reflect.Struct:
				fmt.Printf("%v (struct)", key)
			case reflect.Bool:
				fmt.Printf("%v (bool)", key)
			case reflect.Interface:
				fmt.Printf("%v (interface)", key)
			default:
				fmt.Printf("%v (unknown)", key)
			}
			fmt.Println()
		}
	}
	return &resp, nil
}

// Resource ...
type Resource struct {
	name   string
	client *Client
	record interface{}
}

// NewResource returns a new resource manipulator
func (c *Client) NewResource(name string, record interface{}) Resource {
	// TODO: panic early if record is not a pointer
	return Resource{name, c, record}
}

// RequestBytes makes a raw request to the Airtable API
func (c *Client) RequestBytes(resource string, options QueryEncoder) ([]byte, error) {
	var err error

	if err = c.checkSetup(); err != nil {
		return nil, err
	}

	if options == nil {
		options = url.Values{}
	}

	url := c.makeURL(resource, options)

	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}
	h := make(http.Header)
	h.Add("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	req.Header = h

	var httpclient http.Client
	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err = checkErrorResponse(bytes); err != nil {
		return bytes, err
	}

	return bytes, nil
}
