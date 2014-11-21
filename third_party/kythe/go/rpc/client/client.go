// Package client implements client interfaces to K-RPC servers (either HTTP
// or streaming).
//
// For normal Clients, there are two ways to call a service method:
//
//   // Call a method, expecting a single result.
//   (*Client) Call(string, interface{}, interface{}) error
//
//   // Stream a method's zero or more results, using a Reader to handle each
//   (*Client) Stream(string, interface{}) *Reader
//
// There are also the disjoint-pipe clients which can (*PipeWriter) Send
// requests without handling the results or can (*PipeReader) Receive responses
// without sending a request. These are useful when piping commands in normal
// POSIX fashion.
//
// Example:
//   c := client.NewHTTPClient("localhost:8888")
//
//   req := &apb.AnalysisRequest{/* ... */}
//   rd := c.Stream("/CompilationAnalyzer/Analyze", req)
//   defer rd.Close()
//   for {
//     var entry spb.Entry
//     if err := rd.NextResult(&req); err == io.EOF {
//       break
//     } else if err != nil {
//       log.Fatal(err)
//     }
//     // handle entry
//   }
//
// Example:
//   c := client.NewPipeWriter(os.Stdout)
//   if err := c.Send("/ServerInfo/List", nil); err != nil :
//     log.Fatal(err)
//   }
//
// Example:
//   rd := client.NewPipeReader(os.Stdin)
//   var entry spb.Entry
//   err := rd.Receive(&entry, func(id []byte, err error, success bool) {
//     // handle entry/err/success
//   })
//   if err != nil {
//     log.Fatal(err)
//   }
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sync/atomic"
	"time"

	"third_party/kythe/go/rpc/protocol"
	"third_party/kythe/go/util/httpencoding"
)

const httpContentType = "application/json"

var httpClient = &http.Client{
	Transport: &http.Transport{
		// MaxIdleConnsPerHost increase from default (2) to allow higher volumes of
		// KRPC calls
		MaxIdleConnsPerHost: 128,

		// From http.DefaultTransport
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

// httpTransport is a handle for a K-RPC HTTP server.
type httpTransport struct {
	url *url.URL

	// atomically incremented id per request sent
	id uint64
}

// A Transport is a K-RPC protocol implementation
type Transport interface {
	SendRequest(version string, serviceMethod string, params interface{}) (io.ReadCloser, error)
}

// A Client to a handle to a K-RPC server
type Client struct {
	Transport
}

var addrPattern = regexp.MustCompile("^.*:[[:digit:]]+$")

// ValidHTTPAddr returns true if the given string is a valid address
// (<host>:<port>) for an HTTPClient.
func ValidHTTPAddr(addr string) bool {
	return addrPattern.MatchString(addr)
}

// NewHTTPClient creates a client connected to the HTTP K-RPC address given
func NewHTTPClient(addr string) *Client {
	u := &url.URL{Scheme: "http", Host: addr, Path: "/"}
	return &Client{&httpTransport{url: u}}
}

func discardAndClose(r io.ReadCloser) error {
	io.Copy(ioutil.Discard, r) // Ignore errors
	if err := r.Close(); err != nil {
		return fmt.Errorf("error closing rpc/client response body: %v", err)
	}
	return nil
}

func logDiscardAndClose(r io.ReadCloser) {
	if err := discardAndClose(r); err != nil {
		log.Println(err)
	}
}

func encodeRequest(version string, id *uint64, serviceMethod string, params interface{}) ([]byte, error) {
	data, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("error encoding params: %v", err)
	}
	idData, err := json.Marshal(atomic.AddUint64(id, 1))
	if err != nil {
		return nil, fmt.Errorf("error encoding id: %v", err)
	}
	req, err := json.Marshal(&protocol.Request{
		Version: version,
		ID:      idData,
		Method:  serviceMethod,
		Params:  data,
	})
	if err != nil {
		return nil, fmt.Errorf("error encoding protocol request: %v", err)
	}
	return req, nil
}

// SendRequest implements the Transport interface over HTTP
func (c *httpTransport) SendRequest(version string, serviceMethod string, params interface{}) (io.ReadCloser, error) {
	req, err := encodeRequest(version, &c.id, serviceMethod, params)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(&http.Request{
		Method: "POST",
		URL:    c.url,
		Header: map[string][]string{
			"Content-Type":    []string{httpContentType},
			"Accept-Encoding": []string{"gzip", "deflate"},
		},
		Body:          ioutil.NopCloser(bytes.NewBuffer(req)),
		ContentLength: int64(len(req)),
	})
	if err != nil {
		return nil, fmt.Errorf("HTTP failure: %v", err)
	}

	if resp.StatusCode != 200 {
		message := new(bytes.Buffer)
		if _, err := io.Copy(message, resp.Body); err != nil {
			return nil, fmt.Errorf("error reading rpc/client.Request error response body: %v", err)
		}

		// Ensure response is fully read/closed to allow connection reuse
		discardAndClose(resp.Body)
		return nil, fmt.Errorf("RPC error: %q", message.String())
	}

	return httpencoding.UncompressData(resp)
}

func unmarshalResult(dec *json.Decoder, result interface{}) (*protocol.Response, error) {
	var resp protocol.Response
	if err := dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("error decoding JSON-RPC response: %v", err)
	}

	switch {
	case resp.Error != nil:
		return nil, resp.Error
	case resp.Success:
		return &resp, nil
	default:
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return nil, fmt.Errorf("error unmarshalling result: %v", err)
		}
		return &resp, nil
	}
}

// Call calls the given method, expecting a single result that will be
// unmarshalled into the output parameter.
func (c *Client) Call(serviceMethod string, params interface{}, result interface{}) error {
	resp, err := c.SendRequest(protocol.Version2, serviceMethod, params)
	if err != nil {
		return err
	}
	// Ensure response is fully read/closed to allow connection reuse
	defer logDiscardAndClose(resp)

	_, err = unmarshalResult(json.NewDecoder(resp), result)
	return err
}

// Reader provides sequential access to a streaming RPC call's results. When no
// longer used, Readers must be Closed to ensure resources are not leaked.
type Reader struct {
	resp io.ReadCloser
	dec  *json.Decoder
	err  error
}

// NextResult decodes the next available value into result. io.EOF is returned
// if there are no further results.
func (r *Reader) NextResult(result interface{}) error {
	if r.err != nil {
		return r.err
	}

	resp, err := unmarshalResult(r.dec, result)
	if err != nil {
		r.err = err
		return err
	}

	if resp.Version == protocol.Version2 {
		// If the server fell back to V2, we don't expect any more results
		r.err = io.EOF
	} else if resp.Success {
		r.err = io.EOF
		return r.err
	}
	return nil
}

// Close discards the rest of the results and releases the underlying resources.
func (r *Reader) Close() error {
	if r.resp == nil {
		// may happen if there was an initial Reader error set
		return nil
	}
	return discardAndClose(r.resp)
}

// Stream calls the given method, expecting multiple results which can be
// accessed through the returned Reader.
func (c *Client) Stream(serviceMethod string, params interface{}) *Reader {
	resp, err := c.SendRequest(protocol.Version2Streaming, serviceMethod, params)
	return &Reader{resp, json.NewDecoder(resp), err}
}

// WriteStream calls the given method and writes each JSON response to w.
func (c *Client) WriteStream(w io.Writer, serviceMethod string, params interface{}) error {
	resp, err := c.SendRequest(protocol.Version2Streaming, serviceMethod, params)
	if err != nil {
		return err
	}
	// Ensure response is fully read/closed to allow connection reuse
	defer logDiscardAndClose(resp)
	_, err = io.Copy(w, resp)
	return err
}

// WaitUntilReady is a helper that repeatedly attempts to call the ServerInfo
// List method on the given client until either the timeout expires or a
// response is received.  If timeout == 0, it will retry forever.  Returns
// ErrTimedOut if the service did not respond before the timeout, otherwise
// returns the status from the response.
//
// Example:
//   if err := c.WaitUntilReady(1*time.Minute); err != nil {
//     log.Fatal("Connection did not succeed: %v", err)
//   }
//   log.Println("The server is ready to receive requests")
//
func (c *Client) WaitUntilReady(timeout time.Duration) error {
	const maxInterval = 25 * time.Second

	start := time.Now()
	ival := 800 * time.Microsecond
	var lastErr error
	for timeout == 0 || time.Since(start) < timeout {
		var res json.RawMessage
		lastErr = c.Call("/ServerInfo/List", nil, &res)
		if lastErr == nil {
			return nil
		}

		time.Sleep(ival)
		if ival <= maxInterval {
			ival *= 2
		}
	}
	return fmt.Errorf("service did not respond before timeout: %v", lastErr)
}

// PipeReader reads a KRPC result stream for a client.
type PipeReader struct {
	dec *json.Decoder
}

// NewPipeReader returns a new PipeReader decoding r
func NewPipeReader(r io.Reader) *PipeReader {
	return &PipeReader{json.NewDecoder(r)}
}

// Receive decodes each response result into result and calls f with the result's
// id. If a response error was received, result will not be touched and err will
// be populated for f. Likewise, on the end of a streaming call, result will not
// be touched and success will be true for the call to f.
func (r *PipeReader) Receive(result interface{}, f func(id []byte, err error, success bool) bool) error {
	for {
		var resp protocol.Response
		if err := r.dec.Decode(&resp); err == io.EOF {
			break
		} else if err != nil {
			return err
		} else if err := protocol.CheckID(resp.ID); err != nil {
			return err
		}

		if resp.Error == nil && !resp.Success {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("error unmarshalling result: %v", err)
			}
		}

		var respErr error
		if resp.Error != nil {
			// this is due to Go's handling of nil values with interface types
			respErr = resp.Error
		}

		if !f(resp.ID, respErr, resp.Success) {
			break
		}
	}

	return nil
}

// PipeWriter is a handle to a piped K-RPC server.
type PipeWriter struct {
	w io.Writer

	// atomically incremented id per request sent
	id uint64
}

// NewPipeWriter creates a PipeWriter connected to the given io.Writer.
func NewPipeWriter(w io.Writer) *PipeWriter {
	return &PipeWriter{w, 0}
}

// Send writes a server request for the given serviceMethod and params value.
func (c *PipeWriter) Send(serviceMethod string, params interface{}) error {
	if req, err := encodeRequest(protocol.Version2Streaming, &c.id, serviceMethod, params); err != nil {
		return err
	} else if _, err := c.w.Write(req); err != nil {
		return fmt.Errorf("error writing request: %v", err)
	}
	return nil
}
