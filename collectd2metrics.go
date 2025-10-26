package main

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"github.com/XANi/collectd2metrics/config"
	"github.com/XANi/collectd2metrics/datatypes"
	"github.com/XANi/collectd2metrics/promwriter"
	"github.com/XANi/collectd2metrics/web"
	"github.com/XANi/go-yamlcfg"
	"github.com/efigence/go-mon"
	"github.com/urfave/cli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/fs"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strconv"
)

var version string
var log *zap.SugaredLogger
var debug = true

// /* embeds with all files, just dir/ ignores files starting with _ or .
//
//go:embed static templates
var embeddedWebContent embed.FS

func init() {
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	// naive systemd detection. Drop timestamp if running under it
	if os.Getenv("INVOCATION_ID") != "" || os.Getenv("JOURNAL_STREAM") != "" {
		consoleEncoderConfig.TimeKey = ""
	}
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)
	consoleStderr := zapcore.Lock(os.Stderr)
	_ = consoleStderr
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return (lvl < zapcore.ErrorLevel) != (lvl == zapcore.DebugLevel && !debug)
	})
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, os.Stderr, lowPriority),
		zapcore.NewCore(consoleEncoder, os.Stderr, highPriority),
	)
	logger := zap.New(core)
	if debug {
		logger = logger.WithOptions(
			zap.Development(),
			zap.AddCaller(),
			zap.AddStacktrace(highPriority),
		)
	} else {
		logger = logger.WithOptions(
			zap.AddCaller(),
		)
	}
	log = logger.Sugar()

}

func main() {
	defer log.Sync()
	// register internal stats
	mon.RegisterGcStats()
	app := cli.NewApp()
	app.Name = "foobar"
	app.Description = "do foo to bar"
	app.Version = version
	app.HideHelp = true
	log.Infof("Starting %s version: %s", app.Name, version)
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "help, h", Usage: "show help"},
		cli.BoolFlag{Name: "debug, d", Usage: "enable debug logs"},
		cli.StringFlag{
			Name:  "config-file",
			Usage: "configuration file to use. Will be written if nonexistent. Alternatively /etc/collectd2metrics/config.yaml will be checked",
			Value: "./cfg/config.yaml",
		},
		cli.StringFlag{
			Name:   "listen-addr",
			Value:  "127.0.0.1:3001",
			Usage:  "Listen addr",
			EnvVar: "LISTEN_ADDR",
		},
		cli.StringFlag{
			Name:  "pprof-addr",
			Value: "",
			Usage: "address to run pprof on, disabled by default",
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.Bool("help") {
			cli.ShowAppHelp(c)
			os.Exit(1)
		}
		debug = c.Bool("debug")
		log.Debug("debug enabled")

		cfgFiles := []string{
			c.String("config-file"),
			"/etc/collectd2metrics/config.yaml",
		}
		var cfg config.Config
		err := yamlcfg.LoadConfig(cfgFiles, &cfg)
		var webDir fs.FS
		webDir = embeddedWebContent
		if st, err := os.Stat("./static"); err == nil && st.IsDir() {
			if st, err := os.Stat("./templates"); err == nil && st.IsDir() {
				webDir = os.DirFS(".")
				log.Infof(`detected directories "static" and "templates", using local static files instead of ones embedded in binary`)
			}
		}
		writers := []datatypes.CollectdWriter{}
		if len(cfg.PrometheusWriter) > 0 {
			for i, w := range cfg.PrometheusWriter {
				wrUrl, err := url.Parse(w.URL)
				if err != nil {
					log.Fatalf("failed to parse url %s:%s", w.URL, err)
				}
				sum := sha256.Sum256([]byte(wrUrl.String() + strconv.Itoa(i)))
				instance := fmt.Sprintf("%s_%s_%x", wrUrl.Host, wrUrl.Port(), sum[0:4])
				w.InstanceName = instance
				w.Logger = log.Named(fmt.Sprintf("wr_prom[%d]", i))
				log.Infof("starting promwriter instance %s", w.InstanceName)
				promWr, err := promwriter.New(w)
				if err != nil {
					log.Fatalf("error starting Prometheus writer: %s", err)
				}
				writers = append(writers, promWr)
			}
		} else {
			log.Panicf("no instance in config: %+v", cfg)
		}
		w, err := web.New(web.Config{
			Logger:     log,
			ListenAddr: c.String("listen-addr"),
			Writers:    writers,
		}, webDir)
		if err != nil {
			log.Panicf("error starting web listener: %s", err)
		}
		if len(c.String("pprof-addr")) > 0 {
			log.Infof("listening pprof on %s", c.String("pprof-addr"))
			go func() {
				log.Errorf("failed to start debug listener: %s (ignoring)", http.ListenAndServe(c.String("pprof-addr"), nil))
			}()
		}

		return w.Run()
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("err: %s", err)
	}
}
