// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	ew "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

var (
	Newline            = []byte{10} // \n
	Space              = []byte{32} //
	Hyphen             = []byte{45} // -
	Equal              = []byte{61} // =
	LeftSquareBracket  = []byte{91} // [
	RightSquareBracket = []byte{93} // ]
)

var (
	dumpReqHeader  = []byte("Request Header")
	dumpReqBody    = []byte("Request Body")
	dumpResHeader  = []byte("Response Header")
	dumpResBody    = []byte("Response Body")
	dumpEqual      = append(Space, append(Equal, Space...)...)
	dumpLeftSplit  = append(bytes.Repeat(Hyphen, 4), LeftSquareBracket...)
	dumpRightSplit = append(RightSquareBracket, append(bytes.Repeat(Hyphen, 4), Newline...)...)
)

type DumpHandler func(echo.Context, []byte, []byte)

type HeaderSkipper func(string) bool

type DumpConfig struct {
	Skipper ew.Skipper

	RequestHeader        bool
	RequestHeaderSkipper HeaderSkipper

	ResponseHeader        bool
	ResponseHeaderSkipper HeaderSkipper

	ResponseBodySkipper ew.Skipper

	Extra func(echo.Context) []byte
}

type DumpResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *DumpResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *DumpResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *DumpResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *DumpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}

	return nil, nil, http.ErrNotSupported
}

func dumpBuffer(cfg *DumpConfig, c echo.Context, reqBody, resBody []byte) []byte {
	// if skip dump
	if cfg.Skipper != nil && cfg.Skipper(c) {
		return nil
	}

	var buffer bytes.Buffer

	// log time
	buffer.WriteString(time.Now().Format("2006-01-02 15:04:05.00000"))

	// log [METHOD]
	buffer.Write(Space)
	buffer.Write(LeftSquareBracket)
	buffer.WriteString(c.Request().Method)
	buffer.Write(RightSquareBracket)
	buffer.Write(Space)

	// log uri \n
	buffer.WriteString(c.Request().RequestURI)
	buffer.Write(Newline)

	// log request header
	if cfg.RequestHeader {
		// ----[Request Header]----
		buffer.Write(dumpLeftSplit)
		buffer.Write(dumpReqHeader)
		buffer.Write(dumpRightSplit)

		// TODO c.Request().Header.Write
		// k = v
		for _, s := range getHeaders(c.Request().Header, cfg.RequestHeaderSkipper) {
			buffer.WriteString(s)
			buffer.Write(Newline)
		}
	}

	// log request body
	if len(reqBody) > 0 {
		// ----[Request Body]----
		buffer.Write(dumpLeftSplit)
		buffer.Write(dumpReqBody)
		buffer.Write(dumpRightSplit)
		buffer.Write(reqBody)
		buffer.Write(Newline)
	}

	// log response header
	if cfg.ResponseHeader {
		// ----[Response Header]----
		buffer.Write(dumpLeftSplit)
		buffer.Write(dumpResHeader)
		buffer.Write(dumpRightSplit)

		// k = v

		for _, s := range getHeaders(c.Response().Header(), cfg.ResponseHeaderSkipper) {
			buffer.WriteString(s)
			buffer.Write(Newline)
		}
	}

	// log response body
	if len(resBody) > 0 {
		// ----[Response Body]----
		buffer.Write(dumpLeftSplit)
		buffer.Write(dumpResBody)
		buffer.Write(dumpRightSplit)
		buffer.Write(resBody)
		buffer.Write(Newline)
	}

	buffer.Write(Newline)

	return buffer.Bytes()
}

func dump(handler DumpHandler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			// Request
			var reqBody []byte

			if c.Request().Body != nil { // Read
				reqBody, _ = io.ReadAll(c.Request().Body)
			}

			c.Request().Body = io.NopCloser(bytes.NewBuffer(reqBody)) // Reset

			// Response
			resBody := new(bytes.Buffer)
			mw := io.MultiWriter(c.Response().Writer, resBody)
			writer := &DumpResponseWriter{Writer: mw, ResponseWriter: c.Response().Writer}

			c.Response().Writer = writer

			err = next(c)

			// if err != nil {
			//     c.Error(err)
			// }

			// Callback
			handler(c, reqBody, resBody.Bytes())

			return
		}
	}
}

type DumpZapLoggerMiddleware struct {
}

func NewDumpLoggerMiddleware() *DumpZapLoggerMiddleware {
	return &DumpZapLoggerMiddleware{}
}

func DumpMiddleware(skipper ew.Skipper) echo.MiddlewareFunc {
	return NewDumpLoggerMiddleware().WithDefaultConfig(skipper)
}

func getHeaders(headers http.Header, skipper HeaderSkipper) (strs []string) {
	for k := range headers {
		if skipper != nil && skipper(k) {
			continue
		}

		strs = append(strs, k+" = "+headers.Get(k))
	}

	return
}

type DumpReceived = int8

const (
	DumpReceivedRestServer DumpReceived = 1 // 1: reset server 收到请求
)

func (mw *DumpZapLoggerMiddleware) WithConfig(cfg *DumpConfig) echo.MiddlewareFunc {
	return dump(func(c echo.Context, reqBody []byte, resBody []byte) {
		if cfg.Skipper != nil && cfg.Skipper(c) {
			return
		}

		if c.Get(MiddlewareKeyDumpSkip) != nil {
			if skip, ok := c.Get(MiddlewareKeyDumpSkip).(bool); ok && skip {
				return
			}
		}

		fields := []zap.Field{
			zap.Int("dump", 1),
			zap.String("method", c.Request().Method),
			zap.String("url", c.Request().RequestURI),
			zap.Int8("received", DumpReceivedRestServer),
			zap.String("remote_addr", c.RealIP()),
		}

		// log request header
		if cfg.RequestHeader {
			fields = append(fields, zap.Strings("request_header", getHeaders(c.Request().Header, cfg.RequestHeaderSkipper)))
		}

		// log request body
		if len(reqBody) > 0 {
			fields = append(fields, zap.ByteString("request_body", reqBody))
		}

		// log response header
		if cfg.ResponseHeader {
			fields = append(fields, zap.Strings("response_header", getHeaders(c.Response().Header(), cfg.ResponseHeaderSkipper)))
		}

		if cfg.ResponseBodySkipper == nil {
			cfg.ResponseBodySkipper = func(_ echo.Context) bool {
				return false
			}
		}

		// log response body
		if len(resBody) > 0 && !cfg.ResponseBodySkipper(c) {
			fields = append(fields, zap.ByteString("response_body", resBody))
		}

		if cfg.Extra != nil {
			extraData := cfg.Extra(c)
			if extraData != nil {
				fields = append(fields, zap.ByteString("extra", extraData))
			}
		}

		zap.L().Info(
			"DUMP",
			fields...,
		)
	})
}

func (mw *DumpZapLoggerMiddleware) WithDefaultConfig(skipper ew.Skipper) echo.MiddlewareFunc {
	return mw.WithConfig(&DumpConfig{
		Skipper: skipper,
	})
}

var _ = DumpSkip

func DumpSkip() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(MiddlewareKeyDumpSkip, true)
			return next(c)
		}
	}
}
