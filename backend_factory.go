package krakend

import (
	"context"

	amqp "github.com/devopsfaith/krakend-amqp"
	cel "github.com/devopsfaith/krakend-cel"
	"github.com/devopsfaith/krakend/encoding"
	cb "github.com/devopsfaith/krakend-circuitbreaker/gobreaker/proxy"
	httpcache "github.com/devopsfaith/krakend-httpcache"
	"github.com/devopsfaith/krakend-martian"
	metrics "github.com/devopsfaith/krakend-metrics/gin"
	"github.com/devopsfaith/krakend-oauth2-clientcredentials"
	opencensus "github.com/devopsfaith/krakend-opencensus"
	juju "github.com/devopsfaith/krakend-ratelimit/juju/proxy"
	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	"github.com/devopsfaith/krakend/transport/http/client"
)

// NewBackendFactory creates a BackendFactory by stacking all the available middlewares:
// - oauth2 client credentials
// - http cache
// - martian
// - amqp
// - cel
// - rate-limit
// - circuit breaker
// - metrics collector
// - opencensus collector
func NewBackendFactory(logger logging.Logger, metricCollector *metrics.Metrics) proxy.BackendFactory {
	return NewBackendFactoryWithContext(context.Background(), logger, metricCollector)
}

// NewBackendFactory creates a BackendFactory by stacking all the available middlewares and injecting the received context
func NewBackendFactoryWithContext(ctx context.Context, logger logging.Logger, metricCollector *metrics.Metrics) proxy.BackendFactory {
	requestExecutorFactory := func(cfg *config.Backend) client.HTTPRequestExecutor {
		var clientFactory client.HTTPClientFactory
		if _, ok := cfg.ExtraConfig[oauth2client.Namespace]; ok {
			clientFactory = oauth2client.NewHTTPClient(cfg)
		} else {
			clientFactory = httpcache.NewHTTPClient(cfg)
		}
		return opencensus.HTTPRequestExecutor(clientFactory)
	}
	backendFactory := NewCustomMartianBackendFactory(logger, requestExecutorFactory)
	backendFactory = amqp.NewBackendFactory(ctx, logger, backendFactory)
	backendFactory = cel.BackendFactory(logger, backendFactory)
	backendFactory = juju.BackendFactory(backendFactory)
	backendFactory = cb.BackendFactory(backendFactory, logger)
	backendFactory = metricCollector.BackendFactory("backend", backendFactory)
	backendFactory = opencensus.BackendFactory(backendFactory)
	return backendFactory
}

func NewCustomMartianBackendFactory(logger logging.Logger, ref func(*config.Backend) client.HTTPRequestExecutor) proxy.BackendFactory {
	return func(remote *config.Backend) proxy.Proxy {
		re := ref(remote)
		result, ok := martian.ConfigGetter(remote.ExtraConfig).(martian.Result)
		if !ok {
			return NewCustomHTTPProxyWithHTTPExecutor(remote, re, remote.Decoder)
		}
		switch result.Err {
		case nil:
			return NewCustomHTTPProxyWithHTTPExecutor(remote, martian.HTTPRequestExecutor(result.Result, re), remote.Decoder)
		case martian.ErrEmptyValue:
			return NewCustomHTTPProxyWithHTTPExecutor(remote, re, remote.Decoder)
		default:
			logger.Error(result, remote.ExtraConfig)
			return NewCustomHTTPProxyWithHTTPExecutor(remote, re, remote.Decoder)
		}
	}
}

func NewCustomHTTPProxyWithHTTPExecutor(remote *config.Backend, re client.HTTPRequestExecutor, dec encoding.Decoder) proxy.Proxy {
	if remote.Encoding == encoding.NOOP {
		return proxy.NewHTTPProxyDetailed(remote, re, client.NoOpHTTPStatusHandler, proxy.NoOpHTTPResponseParser)
	}

	ef := proxy.NewEntityFormatter(remote)
	rp := proxy.DefaultHTTPResponseParserFactory(proxy.HTTPResponseParserConfig{dec, ef})
	return proxy.NewHTTPProxyDetailed(remote, re, RestlessHTTPStatusHandler, rp)
}