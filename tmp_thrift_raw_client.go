//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
)

func main() {
	ctx := context.Background()
	cfg := &thrift.TConfiguration{}
	_ = time.Second

	addr := "127.0.0.1:19091"

	sock := thrift.NewTSocketConf(addr, cfg)
	trans := thrift.NewTFramedTransportConf(sock, cfg)
	protoFactory := thrift.NewTBinaryProtocolFactoryConf(cfg)
	iprot := protoFactory.GetProtocol(trans)
	oprot := protoFactory.GetProtocol(trans)

	if err := trans.Open(); err != nil {
		log.Fatal(err)
	}
	defer trans.Close()

	seqID := int32(1)

	// ---- CALL IssueToken(TokenRequest)
	if err := oprot.WriteMessageBegin(ctx, "IssueToken", thrift.CALL, seqID); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteStructBegin(ctx, "IssueToken_args"); err != nil {
		log.Fatal(err)
	}
	// field 1: request (TokenRequest)
	if err := oprot.WriteFieldBegin(ctx, "request", thrift.STRUCT, 1); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteStructBegin(ctx, "TokenRequest"); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteFieldBegin(ctx, "name", thrift.STRING, 1); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteString(ctx, "alice"); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteFieldEnd(ctx); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteFieldBegin(ctx, "role", thrift.STRING, 2); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteString(ctx, "enduser"); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteFieldEnd(ctx); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteFieldStop(ctx); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteStructEnd(ctx); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteFieldEnd(ctx); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteFieldStop(ctx); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteStructEnd(ctx); err != nil {
		log.Fatal(err)
	}
	if err := oprot.WriteMessageEnd(ctx); err != nil {
		log.Fatal(err)
	}
	if err := oprot.Flush(ctx); err != nil {
		log.Fatal(err)
	}

	// ---- READ REPLY
	// (timeout handled at socket layer; keep demo simple)
	name, mtype, rseq, err := iprot.ReadMessageBegin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("reply:", name, mtype, rseq)
	if mtype == thrift.EXCEPTION {
		x := thrift.NewTApplicationException(thrift.UNKNOWN_APPLICATION_EXCEPTION, "")
		_ = x.Read(ctx, iprot)
		_ = iprot.ReadMessageEnd(ctx)
		log.Fatalf("thrift exception: %v", x)
	}
	if _, err := iprot.ReadStructBegin(ctx); err != nil {
		log.Fatal(err)
	}
	var token string
	var expires int64
	for {
		_, ft, fid, err := iprot.ReadFieldBegin(ctx)
		if err != nil {
			log.Fatal(err)
		}
		if ft == thrift.STOP {
			break
		}
		if fid == 0 && ft == thrift.STRUCT {
			// TokenResponse
			if _, err := iprot.ReadStructBegin(ctx); err != nil {
				log.Fatal(err)
			}
			for {
				_, nft, nfid, err := iprot.ReadFieldBegin(ctx)
				if err != nil {
					log.Fatal(err)
				}
				if nft == thrift.STOP {
					break
				}
				switch nfid {
				case 1:
					token, err = iprot.ReadString(ctx)
				case 2:
					expires, err = iprot.ReadI64(ctx)
				default:
					err = iprot.Skip(ctx, nft)
				}
				if err != nil {
					log.Fatal(err)
				}
				_ = iprot.ReadFieldEnd(ctx)
			}
			_ = iprot.ReadStructEnd(ctx)
		} else {
			_ = iprot.Skip(ctx, ft)
		}
		_ = iprot.ReadFieldEnd(ctx)
	}
	_ = iprot.ReadStructEnd(ctx)
	_ = iprot.ReadMessageEnd(ctx)

	fmt.Println("token len:", len(token), "expires:", expires)
}
