package api

import (
	"errors"
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/trace/config"
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/logutil"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics"
	"github.com/DataDog/datadog-agent/pkg/util/fargate"
)

const (
	pipelinesURLSuffix = "/api/v0.1/pipeline_stats"
)

// profilingEndpoints returns the profiling intake urls and their corresponding
// api keys based on agent configuration. The main endpoint is always returned as
// the first element in the slice.
func pipelineEndpoint(cfg *config.AgentConfig) (url *url.URL, apiKey string, err error) {
	if e := cfg.Endpoints; len(e) == 0 || e[0].Host == "" || e[0].APIKey == "" {
		return nil, "", errors.New("config was not properly validated")
	}
	urlStr := cfg.Endpoints[0].Host + pipelinesURLSuffix
	url, err = url.Parse(urlStr)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing main pipelines intake URL %s: %v", urlStr, err)
	}
	return url, apiKey, nil
}

// pipelineProxyHandler returns a new HTTP handler which will proxy requests to the profiling intakes.
// If the main intake URL can not be computed because of config, the returned handler will always
// return http.StatusInternalServerError along with a clarification.
func (r *HTTPReceiver) pipelineProxyHandler() http.Handler {
	target, key, err := pipelineEndpoint(r.conf)
	if err != nil {
		return pipelineErrorHandler(err)
	}
	tags := fmt.Sprintf("host:%s,default_env:%s,agent_version:%s", r.conf.Hostname, r.conf.DefaultEnv, info.Version)
	if orch := r.conf.FargateOrchestrator; orch != fargate.Unknown {
		tag := fmt.Sprintf("orchestrator:fargate_%s", strings.ToLower(string(orch)))
		tags = tags + "," + tag
	}
	return newPipelineProxy(r.conf.NewHTTPTransport(), target, key, tags)
}

func pipelineErrorHandler(err error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		msg := fmt.Sprintf("Pipeline forwarder is OFF: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
	})
}

// newProfileProxy creates an http.ReverseProxy which can forward requests to
// one or more endpoints.
//
// The endpoint URLs are passed in through the targets slice. Each endpoint
// must have a corresponding API key in the same position in the keys slice.
//
// The tags will be added as a header to all proxied requests.
// For more details please see multiTransport.
func newPipelineProxy(transport http.RoundTripper, target *url.URL, key string, tags string) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.Header.Set("Via", fmt.Sprintf("trace-agent %s", info.Version))
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to the default value
			// that net/http gives it: Go-http-client/1.1
			// See https://codereview.appspot.com/7532043
			req.Header.Set("User-Agent", "")
		}
		containerID := req.Header.Get(headerContainerID)
		if ctags := getContainerTags(containerID); ctags != "" {
			req.Header.Set("X-Datadog-Container-Tags", ctags)
		}
		req.Header.Set("X-Datadog-Additional-Tags", tags)
		metrics.Count("datadog.trace_agent.pipelines_stats", 1, nil, 1)
		// URL, Host and key are set in the transport for each outbound request
	}
	logger := logutil.NewThrottled(5, 10*time.Second) // limit to 5 messages every 10 seconds
	return &httputil.ReverseProxy{
		Director:  director,
		ErrorLog:  stdlog.New(logger, "pipelines.Proxy: ", 0),
		Transport: &pipelineTransport{transport, target, key},
	}
}

// pipelineTransport sends HTTP requests to multiple targets using an
// underlying http.RoundTripper. API keys are set separately for each target.
// When multiple endpoints are in use the response from the main endpoint
// is proxied back to the client, while for all aditional endpoints the
// response is discarded. There is no de-duplication done between endpoint
// hosts or api keys.
type pipelineTransport struct {
	http.RoundTripper
	target *url.URL
	key    string
}

func (t *pipelineTransport) RoundTrip(req *http.Request) (rresp *http.Response, rerr error) {
	req.Host = t.target.Host
	req.URL= t.target
	req.Header.Set("DD-API-KEY", t.key)
	return t.RoundTrip(req)
}
