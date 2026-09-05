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

	"github.com/labstack/echo/v4"
	ew "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

const (
	defaultDumpBodyMaxBytes      = 64 << 10
	dumpRequestBodyTruncatedKey  = "_dump_request_body_truncated"
	dumpResponseBodyTruncatedKey = "_dump_response_body_truncated"
)

type DumpHandler func(echo.Context, []byte, []byte)

type HeaderSkipper func(string) bool

type DumpConfig struct {
	Skipper ew.Skipper

	RequestHeader        bool
	RequestHeaderSkipper HeaderSkipper
	RequestBody          bool
	RequestBodySkipper   ew.Skipper

	ResponseHeader        bool
	ResponseHeaderSkipper HeaderSkipper
	ResponseBody          bool
	ResponseBodySkipper   ew.Skipper
	BodyMaxBytes          int64

	Extra func(echo.Context) []byte
}

type dumpBodyCapture struct {
	buffer    bytes.Buffer
	limit     int64
	truncated bool
}

func newDumpBodyCapture(limit int64) *dumpBodyCapture {
	return &dumpBodyCapture{limit: limit}
}

func (capture *dumpBodyCapture) Write(data []byte) (n int, err error) {
	n = len(data)

	remaining := capture.limit - int64(capture.buffer.Len())
	if remaining <= 0 {
		capture.truncated = capture.truncated || len(data) > 0
		return
	}

	writeLength := len(data)
	if int64(writeLength) > remaining {
		writeLength = int(remaining)
		capture.truncated = true
	}

	_, _ = capture.buffer.Write(data[:writeLength])

	return
}

func (capture *dumpBodyCapture) Bytes() []byte {
	if capture == nil {
		return nil
	}

	return capture.buffer.Bytes()
}

type dumpReadCloser struct {
	io.Reader
	io.Closer
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

func (w *DumpResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func dump(cfg *DumpConfig, handler DumpHandler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if cfg.Skipper != nil && cfg.Skipper(c) {
				return next(c)
			}

			var requestBodyCapture *dumpBodyCapture

			captureRequestBody := cfg.RequestBody && (cfg.RequestBodySkipper == nil || !cfg.RequestBodySkipper(c))
			if captureRequestBody && c.Request().Body != nil {
				requestBodyCapture = newDumpBodyCapture(cfg.BodyMaxBytes)
				requestBody := c.Request().Body
				c.Request().Body = &dumpReadCloser{
					Reader: io.TeeReader(requestBody, requestBodyCapture),
					Closer: requestBody,
				}
			}

			var responseBodyCapture *dumpBodyCapture
			originalResponseWriter := c.Response().Writer

			if cfg.ResponseBody {
				responseBodyCapture = newDumpBodyCapture(cfg.BodyMaxBytes)
				writer := &DumpResponseWriter{
					Writer:         io.MultiWriter(originalResponseWriter, responseBodyCapture),
					ResponseWriter: originalResponseWriter,
				}
				c.Response().Writer = writer
			}

			defer func() {
				c.Response().Writer = originalResponseWriter
			}()

			err = next(c)

			if requestBodyCapture != nil && requestBodyCapture.truncated {
				c.Set(dumpRequestBodyTruncatedKey, true)
			}

			if responseBodyCapture != nil && responseBodyCapture.truncated {
				c.Set(dumpResponseBodyTruncatedKey, true)
			}

			handler(c, requestBodyCapture.Bytes(), responseBodyCapture.Bytes())

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
	config := normalizeDumpConfig(cfg)

	return dump(&config, func(c echo.Context, reqBody []byte, resBody []byte) {
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
		if config.RequestHeader {
			fields = append(fields, zap.Strings("request_header", getHeaders(c.Request().Header, config.RequestHeaderSkipper)))
		}

		// log request body
		if len(reqBody) > 0 {
			fields = append(fields, zap.ByteString("request_body", reqBody))
		}

		if truncated, _ := c.Get(dumpRequestBodyTruncatedKey).(bool); truncated {
			fields = append(fields, zap.Bool("request_body_truncated", true))
		}

		// log response header
		if config.ResponseHeader {
			fields = append(fields, zap.Strings("response_header", getHeaders(c.Response().Header(), config.ResponseHeaderSkipper)))
		}

		if len(resBody) > 0 && (config.ResponseBodySkipper == nil || !config.ResponseBodySkipper(c)) {
			fields = append(fields, zap.ByteString("response_body", resBody))
		}

		if truncated, _ := c.Get(dumpResponseBodyTruncatedKey).(bool); truncated {
			fields = append(fields, zap.Bool("response_body_truncated", true))
		}

		if config.Extra != nil {
			extraData := config.Extra(c)
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

func normalizeDumpConfig(cfg *DumpConfig) DumpConfig {
	if cfg == nil {
		return DumpConfig{BodyMaxBytes: defaultDumpBodyMaxBytes}
	}

	config := *cfg
	if config.BodyMaxBytes <= 0 {
		config.BodyMaxBytes = defaultDumpBodyMaxBytes
	}

	return config
}

func (mw *DumpZapLoggerMiddleware) WithDefaultConfig(skipper ew.Skipper) echo.MiddlewareFunc {
	return mw.WithConfig(&DumpConfig{
		Skipper: skipper,
	})
}

func DumpSkip() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(MiddlewareKeyDumpSkip, true)
			return next(c)
		}
	}
}
