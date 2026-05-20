//go:build js

package esplora

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"syscall/js"
)

func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: jsFetchRoundTripper{},
	}
}

type jsFetchRoundTripper struct{}

func (j jsFetchRoundTripper) RoundTrip(req *http.Request) (*http.Response,
	error) {

	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
	}

	init := js.Global().Get("Object").New()
	init.Set("method", req.Method)
	init.Set("mode", "cors")
	init.Set("credentials", "omit")

	headers := js.Global().Get("Headers").New()
	for name, values := range req.Header {
		for _, value := range values {
			headers.Call("append", name, value)
		}
	}
	init.Set("headers", headers)

	if len(bodyBytes) > 0 {
		body := js.Global().Get("Uint8Array").New(len(bodyBytes))
		js.CopyBytesToJS(body, bodyBytes)
		init.Set("body", body)
	}

	result := make(chan fetchResult, 2)
	var thenResp, thenBody, catch js.Func
	catch = js.FuncOf(func(_ js.Value, args []js.Value) any {
		result <- fetchResult{err: jsError(args)}
		return nil
	})
	thenBody = js.FuncOf(func(_ js.Value, args []js.Value) any {
		buffer := args[0]
		array := js.Global().Get("Uint8Array").New(buffer)
		body := make([]byte, array.Get("byteLength").Int())
		js.CopyBytesToGo(body, array)
		result <- fetchResult{body: body}
		return nil
	})
	thenResp = js.FuncOf(func(_ js.Value, args []js.Value) any {
		resp := args[0]
		result <- fetchResult{resp: resp}
		resp.Call("arrayBuffer").Call("then", thenBody).Call("catch", catch)
		return nil
	})
	defer func() {
		thenResp.Release()
		thenBody.Release()
		catch.Release()
	}()

	js.Global().Call("fetch", req.URL.String(), init).
		Call("then", thenResp).
		Call("catch", catch)

	var resp js.Value
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()

	case first := <-result:
		if first.err != nil {
			return nil, first.err
		}
		resp = first.resp
	}

	var body []byte
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()

	case second := <-result:
		if second.err != nil {
			return nil, second.err
		}
		body = second.body
	}

	return &http.Response{
		Status:        fmt.Sprintf("%d %s", resp.Get("status").Int(), resp.Get("statusText").String()),
		StatusCode:    resp.Get("status").Int(),
		Header:        responseHeaders(resp.Get("headers")),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}, nil
}

type fetchResult struct {
	resp js.Value
	body []byte
	err  error
}

func responseHeaders(headers js.Value) http.Header {
	result := make(http.Header)
	forEach := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) >= 2 {
			result.Add(args[1].String(), args[0].String())
		}
		return nil
	})
	defer forEach.Release()
	headers.Call("forEach", forEach)
	return result
}

func jsError(args []js.Value) error {
	if len(args) == 0 {
		return context.Canceled
	}
	value := args[0]
	if value.Type() == js.TypeObject {
		message := value.Get("message")
		if message.Type() == js.TypeString {
			return fmt.Errorf("fetch failed: %s", message.String())
		}
	}
	text := strings.TrimSpace(value.String())
	if text == "" || text == "<undefined>" || text == "<null>" {
		text = "unknown fetch error"
	}
	return fmt.Errorf("fetch failed: %s", text)
}
