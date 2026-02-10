//go:build ignore
// +build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"

	grpcapi "penny-assesment/internal/transport/grpcapi"
	"penny-assesment/internal/transport"
)

type jsonCodec struct{}

func (jsonCodec) Name() string { return "json" }
func (jsonCodec) Marshal(v any) ([]byte, error) { return json.Marshal(v) }
func (jsonCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

func main() {
	addr := "127.0.0.1:19090"
	encoding.RegisterCodec(jsonCodec{})

	conn, err := grpc.Dial(
		addr,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(jsonCodec{})),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	endTok := issueToken(ctx, conn, "alice", "enduser")
	droneTok := issueToken(ctx, conn, "dr-1", "drone")

	ctxEnd := withBearer(ctx, endTok)
	var order transport.OrderResponse
	err = conn.Invoke(ctxEnd, "/drone.OrderService/SubmitOrder", &grpcapi.SubmitOrderRequest{
		Origin:      transport.Location{Lat: 24.7136, Lng: 46.6753},
		Destination: transport.Location{Lat: 24.7743, Lng: 46.7386},
	}, &order)
	if err != nil {
		log.Fatalf("submit: %v", err)
	}
	fmt.Println("order id:", order.ID)

	ctxDrone := withBearer(ctx, droneTok)
	var reserved transport.OrderResponse
	err = conn.Invoke(ctxDrone, "/drone.DroneService/ReserveJob", &grpcapi.Empty{}, &reserved)
	if err != nil {
		log.Fatalf("reserve: %v", err)
	}
	fmt.Println("reserved status:", reserved.Status)

	var st transport.DroneStatusResponse
	err = conn.Invoke(ctxDrone, "/drone.DroneService/Heartbeat", &grpcapi.HeartbeatRequest{Lat: 24.72, Lng: 46.68}, &st)
	if err != nil {
		log.Fatalf("heartbeat: %v", err)
	}
	fmt.Println("heartbeat ok: drone=", st.Drone.ID)

	fmt.Println("gRPC JSON client OK")
}

func issueToken(ctx context.Context, conn *grpc.ClientConn, name, role string) string {
	var resp grpcapi.TokenResponse
	err := conn.Invoke(ctx, "/drone.AuthService/IssueToken", &grpcapi.TokenRequest{Name: name, Role: role}, &resp)
	if err != nil {
		log.Fatalf("issue token: %v", err)
	}
	return resp.Token
}

func withBearer(ctx context.Context, token string) context.Context {
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(ctx, md)
}
