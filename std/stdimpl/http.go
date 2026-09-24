package stdimpl

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/textproto"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/soumt-r/hana/errs"
)

// [HTTP] speaks HTTP/1.1 itself over a connection (a plain socket, or one wrapped
// in TLS for https) instead of using net/http: that package brings HTTP/2, proxies,
// cookies and more, and would add about 3.5MB to every packed program. This is the
// small part scripts need — one request, one answer, redirects followed.

const (
	maxHTTPBody   = 32 << 20
	maxRedirects  = 10
	maxHeaderSize = 1 << 20
)

var httpMethods = map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true, "HEAD": true}

// headersArg reads an optional dictionary of request headers (strings to strings).
func headersArg(args []interface{}, i int) (map[string]string, error) {
	headers := map[string]string{}
	if i >= len(args) {
		return headers, nil
	}
	dict, ok := args[i].(map[interface{}]interface{})
	if !ok {
		return nil, errs.New(errs.NativeArgDict, i+1)
	}
	for k, v := range dict {
		name, ok1 := k.(string)
		value, ok2 := v.(string)
		if !ok1 || !ok2 {
			return nil, errs.New(errs.NativeDictStrings, i+1)
		}
		headers[name] = value
	}
	return headers, nil
}

// httpResponse is what one round trip returned.
type httpResponse struct {
	status int
	header textproto.MIMEHeader
	body   []byte
}

// doRequest sends one request, follows redirects, and returns the answer as a
// dictionary: "status" (number), "body" (text) and "headers" (lower-case names,
// several values of one header joined with ", ").
func doRequest(method, rawURL, body string, headers map[string]string) (interface{}, error) {
	if !httpMethods[method] {
		return nil, errs.New(errs.HTTPBadMethod, method)
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return nil, errs.New(errs.HTTPBadURL, rawURL)
	}
	if err := NetAccess("http", rawURL); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(defaultNetTimeout)
	var resp *httpResponse
	for redirects := 0; ; redirects++ {
		if resp, err = roundTrip(method, u, body, headers, deadline); err != nil {
			return nil, httpError(err, rawURL)
		}
		location := resp.header.Get("Location")
		if location == "" || redirects >= maxRedirects || !isRedirect(resp.status) {
			break
		}
		next, err := u.Parse(location)
		if err != nil || (next.Scheme != "http" && next.Scheme != "https") {
			break
		}
		u = next
		if resp.status == 303 || ((resp.status == 301 || resp.status == 302) && method == "POST") {
			method, body = "GET", ""
		}
	}
	return answer(resp), nil
}

func isRedirect(status int) bool {
	return status == 301 || status == 302 || status == 303 || status == 307 || status == 308
}

var errBodyTooLarge = errs.New(errs.HTTPTooLarge, "")

// httpError words a failed request: too large is told apart, the rest is a
// request that could not be made.
func httpError(err error, rawURL string) error {
	if err == errBodyTooLarge {
		return errs.New(errs.HTTPTooLarge, rawURL)
	}
	return errs.New(errs.HTTPFailed, rawURL)
}

func answer(resp *httpResponse) map[interface{}]interface{} {
	text := string(resp.body)
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "�")
	}
	names := make([]string, 0, len(resp.header))
	for name := range resp.header {
		names = append(names, name)
	}
	sort.Strings(names)
	headers := map[interface{}]interface{}{}
	for _, name := range names {
		headers[strings.ToLower(name)] = strings.Join(resp.header[name], ", ")
	}
	return map[interface{}]interface{}{"status": float64(resp.status), "body": text, "headers": headers}
}

// roundTrip makes one request on a fresh connection and reads the whole answer.
func roundTrip(method string, u *url.URL, body string, headers map[string]string, deadline time.Time) (*httpResponse, error) {
	conn, err := dial(u, deadline)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(deadline)

	if _, err := io.WriteString(conn, buildRequest(method, u, body, headers)); err != nil {
		return nil, err
	}
	return readResponse(bufio.NewReader(conn), method)
}

func dial(u *url.URL, deadline time.Time) (net.Conn, error) {
	port := u.Port()
	if port == "" {
		port = map[string]string{"http": "80", "https": "443"}[u.Scheme]
	}
	dialer := &net.Dialer{Deadline: deadline}
	addr := net.JoinHostPort(u.Hostname(), port)
	if u.Scheme == "https" {
		return tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: u.Hostname()})
	}
	return dialer.Dial("tcp", addr)
}

// buildRequest is the request text. The connection is used once ("close"), the
// body is never compressed, and a header the script gives replaces the default of
// the same name.
func buildRequest(method string, u *url.URL, body string, headers map[string]string) string {
	fields := map[string][2]string{ // lower-case name -> name, value
		"host":            {"Host", u.Host},
		"user-agent":      {"User-Agent", "hari"},
		"accept-encoding": {"Accept-Encoding", "identity"},
		"connection":      {"Connection", "close"},
	}
	if body != "" || method == "POST" || method == "PUT" || method == "PATCH" {
		fields["content-length"] = [2]string{"Content-Length", strconv.Itoa(len(body))}
	}
	for name, value := range headers {
		fields[strings.ToLower(name)] = [2]string{name, value}
	}
	var b strings.Builder
	b.WriteString(method + " " + u.RequestURI() + " HTTP/1.1\r\n")
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		b.WriteString(fields[key][0] + ": " + fields[key][1] + "\r\n")
	}
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}

// readResponse reads the status line, the headers and the body (by length, by
// chunks, or up to the end of the connection).
func readResponse(r *bufio.Reader, method string) (*httpResponse, error) {
	tp := textproto.NewReader(r)
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return nil, err
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 2 || !strings.HasPrefix(parts[0], "HTTP/") {
			return nil, io.ErrUnexpectedEOF
		}
		status, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		header, err := tp.ReadMIMEHeader()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if status >= 100 && status < 200 {
			continue // 100 Continue and the like: the real answer follows
		}
		resp := &httpResponse{status: status, header: header}
		if method == "HEAD" || status == 204 || status == 304 {
			return resp, nil
		}
		resp.body, err = readBody(r, header)
		return resp, err
	}
}

func readBody(r *bufio.Reader, header textproto.MIMEHeader) ([]byte, error) {
	if strings.Contains(strings.ToLower(header.Get("Transfer-Encoding")), "chunked") {
		return readChunked(r)
	}
	if length := header.Get("Content-Length"); length != "" {
		n, err := strconv.ParseInt(strings.TrimSpace(length), 10, 64)
		if err != nil || n < 0 {
			return nil, io.ErrUnexpectedEOF
		}
		if n > maxHTTPBody {
			return nil, errBodyTooLarge
		}
		body := make([]byte, n)
		_, err = io.ReadFull(r, body)
		return body, err
	}
	body, err := io.ReadAll(io.LimitReader(r, maxHTTPBody+1))
	if len(body) > maxHTTPBody {
		return nil, errBodyTooLarge
	}
	return body, err
}

func readChunked(r *bufio.Reader) ([]byte, error) {
	tp := textproto.NewReader(r)
	var body []byte
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return nil, err
		}
		if i := strings.IndexByte(line, ';'); i >= 0 {
			line = line[:i] // a chunk extension
		}
		size, err := strconv.ParseInt(strings.TrimSpace(line), 16, 64)
		if err != nil || size < 0 {
			return nil, io.ErrUnexpectedEOF
		}
		if size == 0 {
			_, err = tp.ReadMIMEHeader() // trailers, up to the blank line
			if err == io.EOF {
				err = nil
			}
			return body, err
		}
		if int64(len(body))+size > maxHTTPBody {
			return nil, errBodyTooLarge
		}
		chunk := make([]byte, size)
		if _, err := io.ReadFull(r, chunk); err != nil {
			return nil, err
		}
		body = append(body, chunk...)
		if _, err := tp.ReadLine(); err != nil { // the CRLF after the chunk
			return nil, err
		}
	}
}

// httpGet(주소[, 헤더]).
func httpGet(args []interface{}) (interface{}, error) {
	if err := between(args, 1, 2); err != nil {
		return nil, err
	}
	target, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	headers, err := headersArg(args, 1)
	if err != nil {
		return nil, err
	}
	return doRequest("GET", target, "", headers)
}

// httpPost(주소, 본문[, 헤더]).
func httpPost(args []interface{}) (interface{}, error) {
	if err := between(args, 2, 3); err != nil {
		return nil, err
	}
	target, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	body, err := stringArg(args, 1)
	if err != nil {
		return nil, err
	}
	headers, err := headersArg(args, 2)
	if err != nil {
		return nil, err
	}
	return doRequest("POST", target, body, headers)
}

// httpRequest(방식, 주소[, 본문[, 헤더]]) for the methods the other two do not cover.
func httpRequest(args []interface{}) (interface{}, error) {
	if err := between(args, 2, 4); err != nil {
		return nil, err
	}
	method, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	target, err := stringArg(args, 1)
	if err != nil {
		return nil, err
	}
	body := ""
	if len(args) >= 3 {
		if body, err = stringArg(args, 2); err != nil {
			return nil, err
		}
	}
	headers, err := headersArg(args, 3)
	if err != nil {
		return nil, err
	}
	return doRequest(strings.ToUpper(method), target, body, headers)
}
