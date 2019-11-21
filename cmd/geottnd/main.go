package main

import (
	"context"
	"encoding/hex"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ttnsdk "github.com/TheThingsNetwork/go-app-sdk"
	log "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const appName = "geottn"

var (
	version = "no version from LDFLAGS"

	appID           = flag.String("appID", "akhtestapp", "The things network application ID")
	appAccessKey    = flag.String("appAccessKey", "", "The things network access key")
	httpMetricsPort = flag.Int("httpMetricsPort", 8888, "http port")
	httpAPIPort     = flag.Int("httpAPIPort", 9201, "http API port")
	healthPort      = flag.Int("healthPort", 6666, "grpc health port")

	httpAPIServer     *http.Server
	grpcHealthServer  *grpc.Server
	httpMetricsServer *http.Server
)

func main() {
	flag.Parse()

	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "caller", log.DefaultCaller, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "app", appName)
	logger = level.NewFilter(logger, level.AllowAll())

	stdlog.SetOutput(log.NewStdlibAdapter(logger))

	level.Info(logger).Log("msg", "Starting app", "version", version)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// catch termination
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	g, ctx := errgroup.WithContext(ctx)

	// gRPC Health Server
	healthServer := health.NewServer()
	g.Go(func() error {
		grpcHealthServer = grpc.NewServer()

		healthpb.RegisterHealthServer(grpcHealthServer, healthServer)

		haddr := fmt.Sprintf(":%d", *healthPort)
		hln, err := net.Listen("tcp", haddr)
		if err != nil {
			level.Error(logger).Log("msg", "gRPC Health server: failed to listen", "error", err)
			os.Exit(2)
		}
		level.Info(logger).Log("msg", fmt.Sprintf("gRPC health server serving at %s", haddr))
		return grpcHealthServer.Serve(hln)
	})

	// web server metrics
	g.Go(func() error {
		httpMetricsServer = &http.Server{
			Addr:         fmt.Sprintf(":%d", *httpMetricsPort),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		level.Info(logger).Log("msg", fmt.Sprintf("HTTP Metrics server serving at :%d", *httpMetricsPort))

		// Register Prometheus metrics handler.
		http.Handle("/metrics", promhttp.Handler())

		if err := httpMetricsServer.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}

		return nil
	})

	// web API server
	g.Go(func() error {
		mux := runtime.NewServeMux()

		httpAPIServer = &http.Server{
			Addr:         fmt.Sprintf(":%d", *httpAPIPort),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			Handler:      mux,
		}
		level.Info(logger).Log("msg", fmt.Sprintf("HTTP API server serving at :%d", *httpAPIPort))

		if err := httpAPIServer.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}

		return nil
	})

	// TTN client subscriptions
	g.Go(func() error {
		logger := log.With(logger, "component", "ttnclient")
		config := ttnsdk.NewCommunityConfig(appName)
		config.ClientVersion = version

		// Create a new SDK client for the application
		client := config.NewClient(*appID, *appAccessKey)

		// Make sure the client is closed before the function returns
		// In your application, you should call this before the application shuts down
		defer client.Close()

		// Start Publish/Subscribe client (MQTT)
		pubsub, err := client.PubSub()
		if err != nil {
			level.Error(logger).Log("msg", "can't get pub/sub", "error", err)
			return err
		}

		// Make sure the pubsub client is closed before the function returns
		// In your application, you should call this before the application shuts down
		defer pubsub.Close()

		// Get a publish/subscribe client for all devices
		allDevicesPubSub := pubsub.AllDevices()

		// Make sure the pubsub client is closed before the function returns
		// In your application, you will probably call this before the application shuts down
		// This also stops existing subscriptions, in case you forgot to unsubscribe
		defer allDevicesPubSub.Close()

		// Subscribe to msgs
		msgs, err := allDevicesPubSub.SubscribeUplink()
		if err != nil {
			level.Error(logger).Log("msg", "can't subscribe to events", "error", err)
			return err
		}
		level.Info(logger).Log("msg", "subscribed to uplink messages")

		for {
			select {
			case <-ctx.Done():
				// Unsubscribe from events
				level.Info(logger).Log("msg", "unsubscribing to uplink messages")

				if err = allDevicesPubSub.UnsubscribeEvents(); err != nil {
					level.Error(logger).Log("msg", "can't unsubscribe from events", "error", err)
					return err
				}
				return nil
			case msg := <-msgs:
				if msg == nil {
					break
				}
				hexPayload := hex.EncodeToString(msg.PayloadRaw)
				level.Info(logger).Log(
					"devID", msg.DevID,
					"msg", "received msg",
					"payload", hexPayload)

			}

		}

		return nil
	})

	select {
	case <-interrupt:
		cancel()
		break
	case <-ctx.Done():
		break
	}

	level.Warn(logger).Log("msg", "received shutdown signal")

	healthServer.SetServingStatus(fmt.Sprintf("grpc.health.v1.%s", appName), healthpb.HealthCheckResponse_NOT_SERVING)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if httpMetricsServer != nil {
		_ = httpMetricsServer.Shutdown(shutdownCtx)
	}

	if httpAPIServer != nil {
		_ = httpAPIServer.Shutdown(shutdownCtx)
	}

	if grpcHealthServer != nil {
		grpcHealthServer.GracefulStop()
	}

	err := g.Wait()
	if err != nil {
		level.Error(logger).Log("msg", "server returning an error", "error", err)
		os.Exit(2)
	}

}
