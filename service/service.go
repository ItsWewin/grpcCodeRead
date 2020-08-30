package service

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"sync"
)

type ServerStream interface {
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
}

type UnaryServerInfo struct {
	// Server is the service implementation the user provides. This is read-only.
	Server interface{}
	// FullMethod is the full RPC method string, i.e., /package.service/method.
	FullMethod string
}

type UnaryHandler func(ctx context.Context, req interface{}) (interface{}, error)

type UnaryServerInterceptor func(ctx context.Context, req interface{}, info *UnaryServerInfo, handler UnaryHandler) (resp interface{}, err error)

type methodHandler func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor UnaryServerInterceptor) (interface{}, error)

type StreamHandler func(srv interface{}, stream ServerStream) error

type MethodDesc struct {
	MethodName string
	Handler    methodHandler
}

// StreamDesc represents a streaming RPC service's method specification.
type StreamDesc struct {
	StreamName string
	Handler    StreamHandler

	// At least one of these is true.
	ServerStreams bool
	ClientStreams bool
}

type service struct {
	server interface{} // the server for service methods
	md     map[string]*MethodDesc
	sd     map[string]*StreamDesc
	mdata  interface{}
}

type grpcServer struct {
	opts serverOptions

	mu     sync.Mutex // guards following
	lis    map[net.Listener]bool

	serve  bool
	drain  bool
	cv     *sync.Cond          // signaled when connections close for GracefulStop
	m      map[string]*service //
}

type serverOptions struct {
	maxConcurrentStreams  uint32
	maxReceiveMessageSize int
	maxSendMessageSize    int
}

type ProdAreas int32

type ProdRequest struct {
	ProdId   int32     `protobuf:"varint,1,opt,name=prod_id,json=prodId,proto3" json:"prod_id,omitempty"`
	ProdArea ProdAreas `protobuf:"varint,2,opt,name=prod_area,json=prodArea,proto3,enum=services.ProdAreas" json:"prod_area,omitempty"`
}

type ProdResponse struct {
	ProdStock int32 `protobuf:"varint,1,opt,name=prod_stock,json=prodStock,proto3" json:"prod_stock,omitempty"`
}

type ProdServer interface {
	GetProdStock(context.Context, *ProdRequest) (*ProdResponse, error)
}

type ServiceDesc struct {
	ServiceName string
	// The pointer to the service interface. Used to check whether the user
	// provided implementation satisfies the interface requirements.
	HandlerType interface{}
	Methods     []MethodDesc
	Streams     []StreamDesc
	Metadata    interface{}
}

func _ProdService_GetProdStock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error,
	interceptor UnaryServerInterceptor) (interface{}, error) {

	in := new(ProdRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProdServer).GetProdStock(ctx, in)
	}
	info := &UnaryServerInfo{
		Server:     srv,
		FullMethod: "/services.ProdService/GetProdStock",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProdServer).GetProdStock(ctx, req.(*ProdRequest))
	}
	return interceptor(ctx, in, info, handler)
}


var _ProdService_serviceDesc = ServiceDesc{
	ServiceName: "services.ProdService",
	HandlerType: (*ProdServer)(nil),
	Methods: []MethodDesc{
		{
			MethodName: "GetProdStock",
			Handler:    _ProdService_GetProdStock_Handler,
		},
	},
	Streams:  []StreamDesc{},
	Metadata: "Prod.proto",
}

func RegisterService(sd *ServiceDesc, ss interface{}) {
	ht := reflect.TypeOf(sd.HandlerType).Elem()
	st := reflect.TypeOf(ss)
	if !st.Implements(ht) {
		fmt.Errorf("grpc: Server.RegisterService found the handler of type %v that does not satisfy %v", st, ht)
	}
	s.register(sd, ss)
}

type Server struct {
	opts serverOptions

	mu     sync.Mutex // guards following
	lis    map[net.Listener]bool
	serve  bool
	drain  bool
	cv     *sync.Cond          // signaled when connections close for GracefulStop
	m      map[string]*service // service name -> service info

	channelzRemoveOnce sync.Once
	serveWG            sync.WaitGroup // counts active Serve goroutines for GracefulStop

	channelzID int64 // channelz unique identification number
}

func (s *Server) register(sd *ServiceDesc, ss interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.serve {
		fmt.Errorf("grpc: Server.RegisterService after Server.Serve for %q", sd.ServiceName)
	}
	if _, ok := s.m[sd.ServiceName]; ok {
		fmt.Errorf("grpc: Server.RegisterService found duplicate service registration for %q", sd.ServiceName)
	}
	srv := &service{
		server: ss,
		md:     make(map[string]*MethodDesc),
		sd:     make(map[string]*StreamDesc),
		mdata:  sd.Metadata,
	}
	for i := range sd.Methods {
		d := &sd.Methods[i]
		srv.md[d.MethodName] = d
	}
	for i := range sd.Streams {
		d := &sd.Streams[i]
		srv.sd[d.StreamName] = d
	}
	s.m[sd.ServiceName] = srv
}

func RegisterProdServiceServer(s *grpcServer, srv ProdServer) {
	RegisterService(&_ProdService_serviceDesc, srv)
}