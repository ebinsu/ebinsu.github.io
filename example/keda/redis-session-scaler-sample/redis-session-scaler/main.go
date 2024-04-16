package main

import (
	"log"
	"net"
	"os"

	"github.com/go-redis/redis"
	"google.golang.org/grpc"

	pb "redis-session-scaler/externalscaler"
)

func main() {
	redisAddress := os.Getenv("REDIS_ADDRESS")
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: "",
		DB:       0,
	})
	if err := rdb.Ping().Err(); err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":6000")
	pb.RegisterExternalScalerServer(grpcServer, pb.NewScaler(*rdb))

	log.Println("listenting on :6000")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
