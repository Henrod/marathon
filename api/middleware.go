package api

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/kataras/iris"
	"github.com/uber-go/zap"
)

//VersionMiddleware automatically adds a version header to response
type VersionMiddleware struct {
	Application *Application
}

// Serve automatically adds a version header to response
func (m *VersionMiddleware) Serve(c *iris.Context) {
	c.SetHeader("MARATHON-VERSION", VERSION)
	c.Next()
}

//RecoveryMiddleware recovers from errors in Iris
type RecoveryMiddleware struct {
	OnError func(error, []byte)
}

//Serve executes on error handler when errors happen
func (r RecoveryMiddleware) Serve(ctx *iris.Context) {
	defer func() {
		if err := recover(); err != nil {
			if r.OnError != nil {
				switch err.(type) {
				case error:
					r.OnError(err.(error), debug.Stack())
				default:
					r.OnError(fmt.Errorf("%v", err), debug.Stack())
				}
			}
			ctx.Panic()
		}
	}()
	ctx.Next()
}

//LoggerMiddleware is responsible for logging to Zap all requests
type LoggerMiddleware struct {
	Logger zap.Logger
}

// Serve serves the middleware
func (l *LoggerMiddleware) Serve(ctx *iris.Context) {
	log := l.Logger.With(
		zap.String("source", "request"),
	)

	//all except latency to string
	var ip, method, path string
	var status int
	var latency time.Duration
	var startTime, endTime time.Time

	path = ctx.PathString()
	method = ctx.MethodString()

	startTime = time.Now()

	ctx.Next()

	//no time.Since in order to format it well after
	endTime = time.Now()
	latency = endTime.Sub(startTime)

	status = ctx.Response.StatusCode()
	ip = ctx.RemoteAddr()

	qs, headers, cookies := getHTTPParams(ctx)

	reqLog := log.With(
		zap.Time("endTime", endTime),
		zap.Int("statusCode", status),
		zap.Duration("latency", latency),
		zap.String("ip", ip),
		zap.String("method", method),
		zap.String("path", path),
		zap.String("querystring", qs),
		zap.Object("headers", headers),
		zap.String("cookies", cookies),
		zap.String("body", string(ctx.Response.Body())),
	)

	//request failed
	if status > 399 && status < 500 {
		reqLog.Warn("Request failed.")
		return
	}

	//request is ok, but server failed
	if status > 499 {
		reqLog.Error("Response failed.")
		return
	}

	//Everything went ok
	reqLog.Info("Request successful.")
}

// NewLoggerMiddleware returns the logger middleware
func NewLoggerMiddleware(theLogger zap.Logger) iris.HandlerFunc {
	l := &LoggerMiddleware{Logger: theLogger}
	return l.Serve
}

//SentryMiddleware is responsible for sending all exceptions to sentry
type SentryMiddleware struct {
	Application *Application
}

func getHTTPParams(ctx *iris.Context) (string, map[string]string, string) {
	qs := ""
	if len(ctx.URLParams()) > 0 {
		qsBytes, _ := json.Marshal(ctx.URLParams())
		qs = string(qsBytes)
	}

	headers := map[string]string{}
	ctx.RequestCtx.Response.Header.VisitAll(func(key []byte, value []byte) {
		headers[string(key)] = string(value)
	})

	cookies := string(ctx.RequestCtx.Response.Header.Peek("Cookie"))
	return qs, headers, cookies
}

//NewHTTPFromCtx returns a new context for Raven
func NewHTTPFromCtx(ctx *iris.Context) *raven.Http {
	qs, headers, cookies := getHTTPParams(ctx)

	h := &raven.Http{
		Method:  string(ctx.Method()),
		Cookies: cookies,
		Query:   qs,
		URL:     ctx.URI().String(),
		Headers: headers,
	}
	return h
}

// Serve serves the middleware
func (l *SentryMiddleware) Serve(ctx *iris.Context) {
	ctx.Next()

	if ctx.Response.StatusCode() > 499 {
		tags := map[string]string{
			"source": "app",
			"type":   "Internal server error",
			"url":    ctx.Request.URI().String(),
		}
		raven.SetHttpContext(NewHTTPFromCtx(ctx))
		raven.CaptureError(fmt.Errorf("%s", string(ctx.Response.Body())), tags)
	}
}
