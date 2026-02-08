package thriftapi

import (
	"github.com/apache/thrift/lib/go/thrift"

	"penny-assesment/internal/auth"
	"penny-assesment/internal/service"
)

type Server struct {
	server *thrift.TSimpleServer
}

func NewServer(addr string, svc *service.Service, authenticator *auth.Authenticator) (*Server, error) {
	transport, err := thrift.NewTServerSocket(addr)
	if err != nil {
		return nil, err
	}
	processor := NewProcessor(svc, authenticator)
	transportFactory := thrift.NewTFramedTransportFactory(thrift.NewTTransportFactory())
	protocolFactory := thrift.NewTBinaryProtocolFactoryConf(&thrift.TConfiguration{})
	server := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)
	return &Server{server: server}, nil
}

func (s *Server) Serve() error {
	return s.server.Serve()
}

func (s *Server) Stop() {
	s.server.Stop()
}
