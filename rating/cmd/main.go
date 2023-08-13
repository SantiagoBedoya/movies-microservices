package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/SantiagoBedoya/movies/gen"
	"github.com/SantiagoBedoya/movies/pkg/discovery"
	"github.com/SantiagoBedoya/movies/pkg/discovery/consul"
	"github.com/SantiagoBedoya/movies/rating/internal/controller/rating"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"

	// httphandler "github.com/SantiagoBedoya/movies/rating/internal/handler/http"
	grpchandler "github.com/SantiagoBedoya/movies/rating/internal/handler/grpc"

	"github.com/SantiagoBedoya/movies/rating/internal/repository/mysql"
)

const serviceName = "rating"

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
	log.Printf("Starting the rating service on port %d", cfg.APIConfig.Port)

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

	repo, err := mysql.New()
	if err != nil {
		panic(err)
	}
	ctrl := rating.New(repo, nil)
	h := grpchandler.New(ctrl)
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", cfg.APIConfig.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv := grpc.NewServer()
	reflection.Register(srv)
	gen.RegisterRatingServiceServer(srv, h)
	srv.Serve(lis)
}

// 	h := httphandler.New(ctrl)
// 	http.Handle("/rating", http.HandlerFunc(h.Handle))
// 	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
// 		panic(err)
// 	}
// }
