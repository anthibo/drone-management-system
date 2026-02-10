//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/apache/thrift/lib/go/thrift"

	"penny-assesment/tmp_thrift_gen/thriftapi"
)

func main() {
	addr := "127.0.0.1:19091"

	cfg := &thrift.TConfiguration{}
	// Try non-strict binary read/write (some clients/servers differ here)
	cfg.TBinaryStrictRead = thrift.BoolPtr(false)
	cfg.TBinaryStrictWrite = thrift.BoolPtr(false)

	sock := thrift.NewTSocketConf(addr, cfg)
	trans := thrift.NewTFramedTransportConf(sock, cfg)
	protoFactory := thrift.NewTBinaryProtocolFactoryConf(cfg)

	if err := trans.Open(); err != nil {
		log.Fatal(err)
	}
	defer trans.Close()

	client := thrift.NewTStandardClient(protoFactory.GetProtocol(trans), protoFactory.GetProtocol(trans))
	authClient := thriftapi.NewAuthServiceClient(client)
	orderClient := thriftapi.NewOrderServiceClient(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tr, err := authClient.IssueToken(ctx, &thriftapi.TokenRequest{Name: "alice", Role: "enduser"})
	if err != nil {
		log.Fatalf("issue token: %v", err)
	}
	fmt.Println("token issued")

	resp, err := orderClient.SubmitOrder(ctx, &thriftapi.SubmitOrderRequest{
		AuthToken: tr.Token,
		Origin:    &thriftapi.Location{Lat: 24.7136, Lng: 46.6753},
		Destination: &thriftapi.Location{Lat: 24.7743, Lng: 46.7386},
	})
	if err != nil {
		log.Fatalf("submit: %v", err)
	}
	fmt.Println("order id:", resp.ID, "status:", resp.Status)

	fmt.Println("Thrift client OK")
}
