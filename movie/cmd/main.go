package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/SantiagoBedoya/movies/gen"
	"github.com/SantiagoBedoya/movies/movie/internal/controller/movie"
	metadatagateway "github.com/SantiagoBedoya/movies/movie/internal/gateway/metadata/http"
	ratinggateway "github.com/SantiagoBedoya/movies/movie/internal/gateway/rating/http"
	grpchandler "github.com/SantiagoBedoya/movies/movie/internal/handler/grpc"
	"github.com/SantiagoBedoya/movies/pkg/discovery"
	"github.com/SantiagoBedoya/movies/pkg/discovery/consul"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

const serviceName = "movie"

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
	log.Printf("Starting the movie service on port %d", cfg.APIConfig.Port)

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
				time.Sleep(1 * time.Second)
			}
		}
	}()
	defer registry.Deregister(ctx, instanceID, serviceName)

	metadataGateway := metadatagateway.New(registry)
	ratingGateway := ratinggateway.New(registry)
	ctrl := movie.New(ratingGateway, metadataGateway)
	g := grpchandler.New(ctrl)
	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", cfg.APIConfig.Port))
	if err != nil {
		log.Fatalf("failedn to listen: %v", err)
	}
	srv := grpc.NewServer()
	gen.RegisterMovieServiceServer(srv, g)
	srv.Serve(list)
	// h := httphandler.New(ctrl)
	// http.Handle("/movie", http.HandlerFunc(h.GetMovieDetails))
	// if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
	// 	panic(err)
	// }
}
