package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/f-taxes/german_tax_report/conf"
	"github.com/f-taxes/german_tax_report/global"
	g "github.com/f-taxes/german_tax_report/grpc_client"
	"github.com/f-taxes/german_tax_report/web"
	"github.com/kataras/golog"
)

//go:embed frontend-dist/*
var WebAssets embed.FS

func init() {
	manifestContent, err := os.ReadFile("./manifest.json")

	if err != nil {
		golog.Fatalf("Failed to read manifest file: %v", err)
		os.Exit(1)
	}

	err = json.Unmarshal(manifestContent, &global.Plugin)

	if err != nil {
		golog.Fatalf("Failed to parse manifest: %v", err)
		os.Exit(1)
	}
}

func main() {
	grpcAddress := flag.String("grpc-addr", "127.0.0.1:4222", "GRPC address of the f-taxes server that the plugin will attempt to connect to.")
	flag.Parse()

	ctx := context.Background()
	g.GrpcClient = g.NewFTaxesClient(*grpcAddress)

	err := g.GrpcClient.Connect(ctx)
	if err != nil {
		golog.Fatal(err)
	}

	go func() {
		for {
			g.GrpcClient.PluginHeartbeat(context.Background())
			time.Sleep(time.Second * 5)
		}
	}()

	conf.LoadAppConfig("config.yaml")

	web.Start(global.Plugin.Web.Address, WebAssets)

	// ctl.Start(global.Plugin.Ctl.Address)
}
