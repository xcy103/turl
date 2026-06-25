package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/beihai0xff/turl/app/turl"
	"github.com/beihai0xff/turl/configs"
	"github.com/beihai0xff/turl/pkg/log"
	"github.com/beihai0xff/turl/pkg/metrics"
	"github.com/beihai0xff/turl/pkg/observability"
	"github.com/beihai0xff/turl/pkg/otel"
	"github.com/beihai0xff/turl/pkg/shutdown"
)

var (
	configPathFlag = &cli.StringFlag{
		Name:    "file",
		Aliases: []string{"f"},
		Usage:   "TURL Server Config File Path",
		Value:   "./config.yaml",
		EnvVars: []string{"TURL_CONFIG_FILE", "TURL_FILE"},
	}
	readonlyFlag = &cli.BoolFlag{
		Name:    "readonly",
		Aliases: []string{"ro"},
		Usage:   "Start Server IN Read-Only Mode",
		Value:   false,
		EnvVars: []string{"TURL_READONLY", "TURL_RO"},
	}
	debugFlag = &cli.BoolFlag{
		Name:    "debug",
		Aliases: []string{"d"},
		Usage:   "Enable Debug Mode",
		Value:   false,
		EnvVars: []string{"TURL_DEBUG"},
	}
)

type serverCLI struct{}

func (c *serverCLI) getServerStartFlags() []cli.Flag {
	return []cli.Flag{configPathFlag, readonlyFlag, debugFlag}
}

func (c *serverCLI) serverHealth(ctx *cli.Context) error {
	rsp, err := http.Get(fmt.Sprintf("http://%s%s", ctx.String("addr"), turl.HealthCheckPath))
	if err != nil {
		slog.Error("health check failed", slog.Any("error", err))
		return err
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		slog.Error("health check failed", slog.Any("status", rsp.Status))
		return fmt.Errorf("health check failed, status: %s", rsp.Status)
	}

	slog.Info("health check success")

	return nil
}

func (c *serverCLI) serverStart(ctx *cli.Context) error {
	conf, err := c.parseServerStartConfig(ctx)
	if err != nil {
		return err
	}

	if err = log.SetDefaultLogger(conf.Log); err != nil {
		return err
	}

	shutdownTracing, err := initTracing(conf)
	if err != nil {
		return err
	}

	handler, err := turl.NewHandler(conf)
	if err != nil {
		return err
	}

	srv, err := turl.NewServer(handler, conf)
	if err != nil {
		return err
	}

	go func() {
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen and serve failed", slog.Any("error", err))
		}
	}()

	obsSrv := startObservabilityServer(conf.Observability, handler)

	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal

	quitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint:mnd
	defer cancel()

	shutdown.GracefulShutdown(quitCtx, shutdownOptions(srv, obsSrv, handler, shutdownTracing)...)

	slog.Info("HTTP Server exited")

	return nil
}

// initTracing installs the global tracer provider from the (optional)
// observability config and returns a span-flushing shutdown function.
func initTracing(conf *configs.ServerConfig) (otel.ShutdownFunc, error) {
	var tracingCfg *configs.TracingConfig
	if conf.Observability != nil {
		tracingCfg = conf.Observability.Tracing
	}

	return otel.Init(context.Background(), tracingCfg)
}

// shutdownOptions assembles the graceful-shutdown steps in order: drain the
// public and admin HTTP servers, close the handler, then flush buffered spans.
func shutdownOptions(srv *http.Server, obsSrv *observability.Server,
	handler *turl.Handler, shutdownTracing otel.ShutdownFunc) []shutdown.OptionFunc {
	opts := []shutdown.OptionFunc{shutdown.HTTPServerShutdown(srv)}
	if obsSrv != nil {
		opts = append(opts, shutdown.HTTPServerShutdown(obsSrv.HTTPServer()))
	}

	return append(opts,
		shutdown.HandlerShutdown(handler),
		shutdown.OptionFunc(shutdownTracing),
	)
}

// startObservabilityServer starts the admin server when enabled, wiring the
// handler's dependency check into the readiness probe. It returns nil when
// observability is disabled or unconfigured.
func startObservabilityServer(c *configs.ObservabilityConfig, handler *turl.Handler) *observability.Server {
	if c == nil || !c.Enable {
		return nil
	}

	obsSrv := observability.New(c)
	obsSrv.RegisterReadinessCheck("storage", handler.CheckReadiness)
	obsSrv.SetMetricsHandler(metrics.Handler())

	go func() {
		if err := obsSrv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("admin server failed", slog.Any("error", err))
		}
	}()

	slog.Info("admin server started", slog.String("addr", obsSrv.Addr()))

	return obsSrv
}

func (c *serverCLI) parseServerStartConfig(ctx *cli.Context) (*configs.ServerConfig, error) {
	filePath := ctx.String(configPathFlag.Name)

	var mp = map[string]interface{}{}
	if ctx.Bool(readonlyFlag.Name) {
		mp[readonlyFlag.Name] = true
	}

	if ctx.Bool(debugFlag.Name) {
		mp[debugFlag.Name] = true
	}

	conf, err := configs.ReadFile(filePath, mp)
	if err != nil {
		slog.Error("read server config file failed", slog.Any("error", err), slog.Any("file", filePath))
		return nil, err
	}

	if err = conf.Validate(); err != nil {
		slog.Error("invalid server config", slog.Any("error", err), slog.Any("config", conf))
		return nil, err
	}

	return conf, nil
}
