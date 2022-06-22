package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	"github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/middleware/http"
	"github.com/openzipkin/zipkin-go/model"
	reporterhttp "github.com/openzipkin/zipkin-go/reporter/http"
)

var sleep = time.Sleep
var httpListenAndServe = http.ListenAndServe
var serviceName = "go-demo"
var zipkinHost = "localhost"
var zipkinPort = "9411"
var limiter = rate.NewLimiter(5, 10)
var limitReachedTime = time.Now().Add(time.Second * (-60))
var limitReached = false
var zipkinClient *zipkinhttp.Client
var tracer *zipkin.Tracer

type Person struct {
	Name string
}

var GitCommit string
var SemVer string

var (
	histogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: "http_server",
		Name:      "resp_time",
		Help:      "Request response time",
	}, []string{
		"service",
		"code",
		"method",
		"path",
	})
)

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.Info("Starting the application")
	if len(os.Getenv("SERVICE_NAME")) > 0 {
		serviceName = os.Getenv("SERVICE_NAME")
	}

	RunServer()
}

func init() {
	prometheus.MustRegister(histogram)
}

func newTracer() (*zipkin.Tracer, error) {
	// inspired by: https://medium.com/devthoughts/instrumenting-a-go-application-with-zipkin-b79cc858ac3e

	// The reporter sends traces to zipkin server
	if len(os.Getenv("ZIPKIN_HOST")) > 0 {
		zipkinHost = os.Getenv("ZIPKIN_HOST")
	}
	if len(os.Getenv("ZIPKIN_PORT")) > 0 {
		zipkinPort = os.Getenv("ZIPKIN_PORT")
	}

	endpointURL := fmt.Sprintf("http://%s:%s/api/v2/spans", zipkinHost, zipkinPort)
	reporter := reporterhttp.NewReporter(endpointURL)

	// Local endpoint represent the local service information
	localEndpoint := &model.Endpoint{ServiceName: serviceName, Port: 8080}

	// Sampler tells you which traces are going to be sampled or not. In this case we will record 100% (1.00) of traces.
	sampler, err := zipkin.NewCountingSampler(1)
	if err != nil {
		return nil, err
	}

	t, err := zipkin.NewTracer(
		reporter,
		zipkin.WithSampler(sampler),
		zipkin.WithLocalEndpoint(localEndpoint),
	)
	if err != nil {
		return nil, err
	}

	return t, err
}

func RunServer() {
	logrus.Info("Running the server")
	var err error = nil
	tracer, err = newTracer()
	if err != nil {
		log.Fatal(err)
	}

	mux := mux.NewRouter()
	mux.Use(zipkinhttp.NewServerMiddleware(
		tracer,
		zipkinhttp.SpanName("request")), // name for request span
	)
	mux.HandleFunc("/", VersionServer)
	mux.HandleFunc("/hello", HelloServer)
	mux.HandleFunc("/random-error", RandomErrorServer)
	mux.HandleFunc("/random-delay", RandomDelayServer)
	mux.HandleFunc("/version", VersionServer)
	mux.HandleFunc("/limiter", LimiterServer)
	mux.Handle("/metrics", promhttp.Handler())

	log.Fatal("ListenAndServe: ", httpListenAndServe(":8080", mux))
}

func HelloServer(w http.ResponseWriter, req *http.Request) {
	start := time.Now()
	defer func() { recordMetrics(start, req, http.StatusOK) }()
	span := zipkin.SpanFromContext(req.Context())
	logrus.WithFields(logrus.Fields{
		"method":  req.Method,
		"path":    req.RequestURI,
		"traceID": span.Context().TraceID,
	}).Info("Request received")

	delay := req.URL.Query().Get("delay")
	if len(delay) > 0 {
		delayNum, _ := strconv.Atoi(delay)
		sleep(time.Duration(delayNum) * time.Millisecond)
		span.Annotate(time.Now(), "delay: "+delay)
	}

	io.WriteString(w, "hello, world!\n")
}

func RandomErrorServer(w http.ResponseWriter, req *http.Request) {
	code := http.StatusOK
	start := time.Now()
	defer func() { recordMetrics(start, req, code) }()
	span := zipkin.SpanFromContext(req.Context())
	logrus.WithFields(logrus.Fields{
		"method":  req.Method,
		"path":    req.RequestURI,
		"traceID": span.Context().TraceID,
	}).Info("Request received")

	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(10)
	msg := "Everything is still OK"
	if n == 0 {
		code = http.StatusInternalServerError
		msg = "ERROR: Something, somewhere, went wrong!"
		logrus.Info(msg)
	}
	w.WriteHeader(code)
	io.WriteString(w, msg)
}

func RandomDelayServer(w http.ResponseWriter, req *http.Request) {
	start := time.Now()
	defer func() { recordMetrics(start, req, http.StatusOK) }()
	span := zipkin.SpanFromContext(req.Context())
	logrus.WithFields(logrus.Fields{
		"method":  req.Method,
		"path":    req.RequestURI,
		"traceID": span.Context().TraceID,
	}).Info("Request received")

	delay := rand.Intn(250)
	sleep(time.Duration(delay) * time.Millisecond)
	calculateDelay(req)
	delay = rand.Intn(250)
	sleep(time.Duration(delay) * time.Millisecond)

	io.WriteString(w, "hello, world!\n")
}

func calculateDelay(req *http.Request) {
	parentSpan := zipkin.SpanFromContext(req.Context())
	spanOptions := zipkin.Parent(parentSpan.Context())
	span := tracer.StartSpan("delay", spanOptions)
	defer span.Finish()
	span.Annotate(time.Now(), "delay start")
	delay := rand.Intn(1500)
	sleep(time.Duration(delay) * time.Millisecond)
	span.Tag("delay", string(delay))
	span.Annotate(time.Now(), "delay finished")
}

func VersionServer(w http.ResponseWriter, req *http.Request) {
	logrus.Infof("%s request to %s", req.Method, req.RequestURI)
	release := req.Header.Get("release")
	if release == "" {
		release = "unknown"
	}

	// HOSTNAME, K_REVISION, K_SERVICE

	msg := fmt.Sprintf("Chart Version: %s; Image Version: %s; Release: %s, "+
		"SemVer: %s, GitCommit: %s,"+
		"Host: %s, Revision: %s, Service: %s\n",
		os.Getenv("CHART_VERSION"), os.Getenv("IMAGE_VERSION"), release,
		SemVer, GitCommit,
		os.Getenv("HOSTNAME"), os.Getenv("K_REVISION"), os.Getenv("K_SERVICE"))
	io.WriteString(w, msg)
}

func LimiterServer(w http.ResponseWriter, req *http.Request) {
	span := zipkin.SpanFromContext(req.Context())
	logrus.WithFields(logrus.Fields{
		"method":  req.Method,
		"path":    req.RequestURI,
		"traceID": span.Context().TraceID,
	}).Info("Request received")
	if limiter.Allow() == false {
		logrus.Info("Limiter in action")
		http.Error(w, http.StatusText(500), http.StatusTooManyRequests)
		limitReached = true
		limitReachedTime = time.Now()
		return
	} else if time.Since(limitReachedTime).Seconds() < 15 {
		logrus.Info("Cooling down after the limiter")
		http.Error(w, http.StatusText(500), http.StatusTooManyRequests)
		return
	}
	msg := fmt.Sprintf("Everything is OK\n")
	io.WriteString(w, msg)
}

func recordMetrics(start time.Time, req *http.Request, code int) {
	duration := time.Since(start)
	histogram.With(
		prometheus.Labels{
			"service": serviceName,
			"code":    fmt.Sprintf("%d", code),
			"method":  req.Method,
			"path":    req.URL.Path,
		},
	).Observe(duration.Seconds())
}
