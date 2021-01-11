package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	_ "net/http/pprof"

	log "github.com/golang/glog"
	"github.com/peekjef72/prometheus_oracle_exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
)

var (
	showVersion   = flag.Bool("version", false, "Print version information.")
	listenAddress = flag.String("web.listen-address", ":9161", "Address to listen on for web interface and telemetry.")
	metricsPath   = flag.String("web.metrics-path", "/metrics", "Path under which to expose metrics.")
	configFile    = flag.String("config.file", "oracle.yml", "Prometheus Oracle Exporter configuration file name.")
)

// Branch is set during build to the git branch.
var Branch = "master"

// BuildDate is set during build to the ISO-8601 date and time.
var BuildUser = "d107684"
var BuildDate = time.Now().Format(time.RFC3339)

// Revision is set during build to the git commit revision.
var Revision = "2"

// Version is set during build to the git describe version
// (semantic version)-(commitish) form.
var Version = "0.0.1"

// VersionShort is set during build to the semantic version.
var VersionShort = "0.0.1"

func init() {
	// Setup build info metric.
	version.Branch = Branch
	version.BuildUser = BuildUser
	version.BuildDate = BuildDate
	version.Revision = Revision
	version.Version = VersionShort

	prometheus.MustRegister(version.NewCollector("prometheus_oracle_exporter"))
}

func main() {
	if os.Getenv("DEBUG") != "" {
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(1)
	}

	// Override --alsologtostderr default value.
	if alsoLogToStderr := flag.Lookup("alsologtostderr"); alsoLogToStderr != nil {
		alsoLogToStderr.DefValue = "true"
		alsoLogToStderr.Value.Set("true")
	}
	// Override the config.file default with the CONFIG environment variable, if set. If the flag is explicitly set, it
	// will end up overriding either.
	if envConfigFile := os.Getenv("CONFIG"); envConfigFile != "" {
		*configFile = envConfigFile
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version.Print("prometheus_oracle_exporter"))
		os.Exit(0)
	}

	log.Infof("Starting ObjectServer Exporter %s %s", version.Info(), version.BuildContext())

	exporter, err := prometheus_oracle_exporter.NewExporter(*configFile)
	if err != nil {
		log.Fatalf("Error creating exporter: %s", err)
	}

	// Setup and start webserver.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "OK", http.StatusOK) })
	http.HandleFunc("/", HomeHandlerFunc(*metricsPath))
	http.HandleFunc("/config", ConfigHandlerFunc(*metricsPath, exporter))
	http.Handle(*metricsPath, ExporterHandlerFor(exporter))
	// Expose exporter metrics separately, for debugging purposes.
	http.Handle("/oracle_exporter_metrics", promhttp.Handler())

	log.Infof("Listening on %s", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}

// LogFunc is an adapter to allow the use of any function as a promhttp.Logger. If f is a function, LogFunc(f) is a
// promhttp.Logger that calls f.
type LogFunc func(args ...interface{})

// Println implements promhttp.Logger.
func (log LogFunc) Println(args ...interface{}) {
	log(args)
}
