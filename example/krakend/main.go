package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	grpc "github.com/devopsfaith/krakend-grpc"
	"github.com/devopsfaith/krakend-grpc/plugin"
	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	"github.com/devopsfaith/krakend/router/gin"
	"github.com/devopsfaith/krakend/transport/http/client"
)

func main() {
	port := flag.Int("p", 0, "Port of the service")
	logLevel := flag.String("l", "ERROR", "Logging level")
	debug := flag.Bool("d", false, "Enable the debug")
	configFile := flag.String("c", "/etc/krakend/configuration.json", "Path to the configuration filename")
	pluginsEnabled := flag.Bool("plugin", false, "enable loading the gRPC client implementations as plugins")
	flag.Parse()

	parser := config.NewParser()
	serviceConfig, err := parser.Parse(*configFile)
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}
	serviceConfig.Debug = serviceConfig.Debug || *debug
	if *port != 0 {
		serviceConfig.Port = *port
	}

	logger, err := logging.NewLogger(*logLevel, os.Stdout, "[KRAKEND]")
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}

	if !*pluginsEnabled {
		registerer := Registerer(0)
		registerer.RegisterClients(grpc.RegisterClient)
	} else {
		tot, err := plugin.Load(serviceConfig.Plugin.Folder, serviceConfig.Plugin.Pattern, grpc.RegisterClient)
		if err != nil {
			logger.Fatal("ERROR:", err.Error())
		}
		logger.Debug("gRPC: total loaded plugins =", tot)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// backend proxy wrapper
	bf := grpc.NewGRPCProxy(logger, proxy.HTTPProxyFactory(client.NewHTTPClient(ctx)))
	routerFactory := gin.DefaultFactory(proxy.NewDefaultFactory(bf, logger), logger)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		select {
		case sig := <-sigs:
			logger.Info("Signal intercepted:", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	routerFactory.NewWithContext(ctx).Run(serviceConfig)
}
