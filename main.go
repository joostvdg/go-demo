package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	zipkin "github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/middleware/http"
	logreporter "github.com/openzipkin/zipkin-go/reporter/log"
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

type Person struct {
	Name string
}

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

func RunServer() {
	logrus.Info("Running the server")

	// set up a span reporter
	reporter := logreporter.NewReporter(log.New(os.Stderr, "", log.LstdFlags))
	defer reporter.Close()

	if len(os.Getenv("ZIPKIN_HOST")) > 0 {
		zipkinHost = os.Getenv("ZIPKIN_HOST")
	}
	if len(os.Getenv("ZIPKIN_PORT")) > 0 {
		zipkinPort = os.Getenv("ZIPKIN_PORT")
	}

	zipkinHostAndPort := fmt.Sprintf("%s:%s", zipkinHost, zipkinPort)
	endpoint, err := zipkin.NewEndpoint("myService", zipkinHostAndPort)
	if err != nil {
		log.Fatalf("unable to create local endpoint: %+v\n", err)
	}

	// initialize our tracer
	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint))
	if err != nil {
		log.Fatalf("unable to create tracer: %+v\n", err)
	}

	// create global zipkin traced http client
	zipkinClient, err = zipkinhttp.NewClient(tracer, zipkinhttp.ClientTrace(true))
	if err != nil {
		log.Fatalf("unable to create client: %+v\n", err)
	}

	// create global zipkin http server middleware
	serverMiddleware := zipkinhttp.NewServerMiddleware(
		tracer, zipkinhttp.TagResponseSize(true),
	)
	mux := mux.NewRouter()
	mux.Use(serverMiddleware)
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
	logrus.Infof("%s request to %s", req.Method, req.RequestURI)

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
	zipkin.SpanFromContext(req.Context())

	logrus.Infof("%s request to %s", req.Method, req.RequestURI)
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
	zipkin.SpanFromContext(req.Context())

	logrus.Infof("%s request to %s", req.Method, req.RequestURI)
	delay := rand.Intn(2000)
	sleep(time.Duration(delay) * time.Millisecond)
	io.WriteString(w, "hello, world!\n")
}

func VersionServer(w http.ResponseWriter, req *http.Request) {
	logrus.Infof("%s request to %s", req.Method, req.RequestURI)
	release := req.Header.Get("release")
	if release == "" {
		release = "unknown"
	}
	msg := fmt.Sprintf("Version: %s; Release: %s\n", os.Getenv("VERSION"), release)
	io.WriteString(w, msg)
}

func LimiterServer(w http.ResponseWriter, req *http.Request) {
	logrus.Infof("%s request to %s", req.Method, req.RequestURI)
	zipkin.SpanFromContext(req.Context())
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
