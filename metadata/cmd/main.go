package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/SantiagoBedoya/movies/gen"
	"github.com/SantiagoBedoya/movies/metadata/internal/controller/metadata"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"

	// httphandler "github.com/SantiagoBedoya/movies/metadata/internal/handler/http"
	grpchandler "github.com/SantiagoBedoya/movies/metadata/internal/handler/grpc"
	"github.com/SantiagoBedoya/movies/metadata/internal/repository/memory"
	"github.com/SantiagoBedoya/movies/pkg/discovery"
	"github.com/SantiagoBedoya/movies/pkg/discovery/consul"
)

const serviceName = "metadata"

func main() {
	f, err := os.Open("base.yaml")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var cfg serviceConfig
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		panic(err)
	}
	log.Printf("Starting the metadata service on port %d", cfg.APIConfig.Port)

	registry, err := consul.NewRegistry("consul:8500")
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	instanceID := discovery.GenerateInstanceID(serviceName)
	if err := registry.Register(ctx, instanceID, serviceName, fmt.Sprintf("localhost:%d", cfg.APIConfig.Port)); err != nil {
		panic(err)
	}

	go func() {
		for {
			if err := registry.ReportHealthyState(instanceID, serviceName); err != nil {
				log.Println("Failed to report healthy state: " + err.Error())
			}
			time.Sleep(1 * time.Second)
		}
	}()
	defer registry.Deregister(ctx, instanceID, serviceName)

	repo := memory.New()
	ctrl := metadata.New(repo)
	h := grpchandler.New(ctrl)
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", cfg.APIConfig.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv := grpc.NewServer()
	gen.RegisterMetadataServiceServer(srv, h)
	srv.Serve(lis)

	// h := httphandler.New(ctrl)
	// http.Handle("/metadata", http.HandlerFunc(h.GetMetadata))
	// if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
	// 	panic(err)
	// }
}
