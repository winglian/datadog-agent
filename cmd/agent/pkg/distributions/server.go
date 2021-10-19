package distributions

import (
	"context"
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/DataDog/datadog-agent/pkg/trace/api/apiutil"
	"github.com/DataDog/datadog-agent/pkg/trace/logutil"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics/timing"
	"github.com/DataDog/datadog-agent/pkg/trace/osutil"
	"github.com/DataDog/sketches-go/ddsketch"
	"github.com/DataDog/sketches-go/ddsketch/mapping"
	"github.com/DataDog/sketches-go/ddsketch/store"
	"github.com/tinylib/msgp/msgp"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

var sketchMapping, _ = mapping.NewLogarithmicMapping(0.01)

type Server struct {
	distributions chan *metrics.Distribution
	server           *http.Server
}

func NewServer(aggr *aggregator.BufferedAggregator) *Server {
	s := &Server{
		distributions: aggr.GetDistributionsChannel(),
	}
	return s
}

func (s *Server) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/v0.1/distributions", s.handleDistributions)

	httpLogger := logutil.NewThrottled(5, 10*time.Second) // limit to 5 messages every 10 seconds
	timeout := time.Second*5
	s.server = &http.Server{
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		ErrorLog:     log.New(httpLogger, "http.Server: ", 0),
		Handler:      mux,
	}
	addr := fmt.Sprintf("%s:%d", "0.0.0.0", 8888)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		killProcess("Error creating tcp listener: %v", err)
	}
	go s.server.Serve(ln)
	log.Println("Server listening for distributions.")
}

func (s *Server) Stop() error {
	expiry := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), expiry)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Server) handleDistributions(w http.ResponseWriter, req *http.Request) {
	defer timing.Since("datadog.agent.receiver.distribution_process_ms", time.Now())
	log.Printf("handling distribution payload\n")

	var in distributionPayload
	if err := msgp.Decode(req.Body, &in); err != nil {
		log.Printf("failed decoding")
		httpDecodingError(err, []string{"handler:distributions", "codec:msgpack", "v:v0.1"}, w)
		return
	}
	sketch, err := ddsketch.DecodeDDSketch(in.Sketch, store.BufferedPaginatedStoreConstructor, sketchMapping)
	if err != nil {
		log.Printf("failed decoding sketch %v", err)
		httpDecodingError(err, []string{"handler:distributions", "codec:msgpack", "v:v0.1"}, w)
		return
	}
	log.Printf("all good")
	s.distributions <- &metrics.Distribution{Name: in.Name, Value: sketch, Tags: in.Tags, Timestamp: in.Timestamp}
}

// httpDecodingError is used for errors happening in decoding
func httpDecodingError(err error, tags []string, w http.ResponseWriter) {
	status := http.StatusBadRequest
	errtag := "decoding-error"
	msg := err.Error()

	switch err {
	case apiutil.ErrLimitedReaderLimitReached:
		status = http.StatusRequestEntityTooLarge
		errtag = "payload-too-large"
		msg = errtag
	case io.EOF, io.ErrUnexpectedEOF:
		errtag = "unexpected-eof"
		msg = errtag
	}
	if err, ok := err.(net.Error); ok && err.Timeout() {
		status = http.StatusRequestTimeout
		errtag = "timeout"
		msg = errtag
	}

	tags = append(tags, fmt.Sprintf("error:%s", errtag))
	http.Error(w, msg, status)
}

// httpOK is a dumb response for when things are a OK. It returns the number
// of bytes written along with a boolean specifying if the response was successful.
func httpOK(w http.ResponseWriter) (n uint64, ok bool) {
	nn, err := io.WriteString(w, "OK\n")
	return uint64(nn), err == nil
}

// killProcess exits the process with the given msg; replaced in tests.
var killProcess = func(format string, a ...interface{}) { osutil.Exitf(format, a...) }
