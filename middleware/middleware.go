package middleware

import (
	"context"
	"github.com/kubemq-io/kubemq-bridges/pkg/retry"
	"reflect"
)

type Middleware interface {
	Do(ctx context.Context, request interface{}) (interface{}, error)
}

type DoFunc func(ctx context.Context, request interface{}) (interface{}, error)

func (df DoFunc) Do(ctx context.Context, request interface{}) (interface{}, error) {
	return df(ctx, request)
}

type MiddlewareFunc func(Middleware) Middleware

func Log(log *LogMiddleware) MiddlewareFunc {
	return func(df Middleware) Middleware {
		return DoFunc(func(ctx context.Context, request interface{}) (interface{}, error) {
			result, err := df.Do(ctx, request)
			switch log.minLevel {
			case "debug":

				log.Infof("request: %v, response: %v, error:%+v", request, result, err)
			case "info":
				if err != nil {
					log.Errorf("error processing request: %s", err.Error())
				} else {
					log.Infof("request processed with successful response")
				}
			case "error":
				if err != nil {
					log.Errorf("error processing request: %s", err.Error())
				}
			}
			return result, err
		})
	}
}
func RateLimiter(rl *RateLimitMiddleware) MiddlewareFunc {
	return func(df Middleware) Middleware {
		return DoFunc(func(ctx context.Context, request interface{}) (interface{}, error) {
			rl.Take()
			return df.Do(ctx, request)
		})
	}
}

func Retry(r *RetryMiddleware) MiddlewareFunc {
	return func(df Middleware) Middleware {
		return DoFunc(func(ctx context.Context, request interface{}) (interface{}, error) {
			var resp interface{}
			err := retry.Do(func() error {
				var doErr error
				resp, doErr = df.Do(ctx, request)
				if doErr != nil {
					return doErr
				}
				return nil
			}, r.opts...)
			return resp, err
		})
	}
}
func Metric(m *MetricsMiddleware) MiddlewareFunc {
	return func(df Middleware) Middleware {
		return DoFunc(func(ctx context.Context, request interface{}) (interface{}, error) {
			resp, err := df.Do(ctx, request)
			m.clearReport()
			if request != nil {
				m.metricReport.RequestVolume = float64(reflect.TypeOf(request).Size())
				m.metricReport.RequestCount = 1
			}
			if resp != nil {
				m.metricReport.ResponseVolume = float64(reflect.TypeOf(resp).Size())
				m.metricReport.ResponseCount = 1
			}
			if err != nil {
				m.metricReport.ErrorsCount = 1
			}
			m.exporter.Report(m.metricReport)
			return resp, err
		})
	}
}
func Chain(md Middleware, list ...MiddlewareFunc) Middleware {
	chain := md
	for _, middleware := range list {
		chain = middleware(chain)
	}
	return chain
}
