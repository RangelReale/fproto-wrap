package fproto_gowrap

import (
	"errors"
	"strings"

	"github.com/RangelReale/fproto"
	"github.com/RangelReale/fproto-wrap"
)

// Generates service specifications for gRPC
type ServiceGen_gRPC struct {
	WrapErrors bool
}

func NewServiceGen_gRPC() *ServiceGen_gRPC {
	return &ServiceGen_gRPC{
		WrapErrors: true,
	}
}

func (s *ServiceGen_gRPC) ServiceType() string {
	return "grpc"
}

func (s *ServiceGen_gRPC) GenerateService(g *Generator, svc *fproto.ServiceElement) error {
	// import all required dependencies
	ctx_alias := g.FService().DeclDep("golang.org/x/net/context", "context")
	grpc_alias := g.FService().DeclDep("google.golang.org/grpc", "grpc")
	var util_alias string
	util_alias = g.FService().DeclDep("github.com/RangelReale/fproto-wrap/gowrap/util", "fproto_gowrap_util")
	func_alias := g.FService().DeclFileDep(nil, "", false)

	tp_svc := g.dep.DepTypeFromElement(svc)
	if tp_svc == nil {
		return errors.New("service type not found")
	}

	svcName := fproto_wrap.CamelCase(svc.Name)

	//
	// CLIENT
	//

	//
	// type MyServiceClient interface
	//
	if !g.FService().GenerateComment(svc.Comment) {
		g.FService().P()
		g.FService().P("// Client API for ", svcName, " service")
		g.FService().P()
	}

	g.FService().P("type ", svcName, "Client interface {")
	g.FService().In()

	for _, rpc := range svc.RPCs {
		tinfo_req, err := g.GetTypeInfoFromParent(tp_svc, rpc.RequestType)
		if err != nil {
			return err
		}
		tinfo_resp, err := g.GetTypeInfoFromParent(tp_svc, rpc.ResponseType)
		if err != nil {
			return err
		}

		g.FService().GenerateComment(rpc.Comment)

		if rpc.StreamsResponse && !rpc.StreamsRequest {
			//
			// MyRPC(ctx context.Context, in *MyReq, opts ...grpc.CallOption) (MyService_MyRPCClient, error)
			//
			g.FService().P(rpc.Name, "(ctx ", ctx_alias, ".Context, in ", tinfo_req.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", opts ...", grpc_alias, ".CallOption) (", svcName, "_", rpc.Name, "Client, error)")

		} else if rpc.StreamsResponse || rpc.StreamsRequest {
			//
			// MyRPC(ctx context.Context, opts ...grpc.CallOption) (MyService_MyRPCClient, error)
			//
			g.FService().P(rpc.Name, "(ctx ", ctx_alias, ".Context, opts ...", grpc_alias, ".CallOption) (", svcName, "_", rpc.Name, "Client, error)")

		} else {
			//
			// MyRPC(ctx context.Context, in *MyReq, opts ...grpc.CallOption) (*MyResp, error)
			//
			g.FService().P(rpc.Name, "(ctx ", ctx_alias, ".Context, in ", tinfo_req.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", opts ...", grpc_alias, ".CallOption) (", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", error)")
		}
	}

	g.FService().Out()
	g.FService().P("}")
	g.FService().P()

	//
	// type wrapMyServiceClient struct
	//

	wrapClientName := "wrap" + svcName + "Client"

	g.FService().P("type ", wrapClientName, " struct {")
	g.FService().In()

	// the default Golang protobuf client
	g.FService().P("cli ", func_alias, ".", svcName, "Client")

	g.FService().Out()
	g.FService().P("}")
	g.FService().P()

	//
	// func NewMyServiceClient(cc *grpc.ClientConn, errorHandler ...wraputil.ServiceErrorHandler) MyServiceClient
	//

	g.FService().P("func New", svcName, "Client(cc *", grpc_alias, ".ClientConn) ", svcName, "Client {")
	g.FService().In()

	g.FService().P("return NewWrap", svcName, "Client(", func_alias, ".New", svcName, "Client(cc))")

	g.FService().Out()
	g.FService().P("}")
	g.FService().P()

	//
	// func NewWrapMyServiceClient(cli source.MyServiceClient) MyServiceClient
	//
	g.FService().P("func NewWrap", svcName, "Client(cli ", func_alias, ".", svcName, "Client) ", svcName, "Client {")
	g.FService().In()

	g.FService().P("return &", wrapClientName, "{cli: cli}")

	g.FService().Out()
	g.FService().P("}")
	g.FService().P()

	// Implement each RPC wrapper

	for _, rpc := range svc.RPCs {
		tinfo_req, err := g.GetTypeInfoFromParent(tp_svc, rpc.RequestType)
		if err != nil {
			return err
		}
		tinfo_resp, err := g.GetTypeInfoFromParent(tp_svc, rpc.ResponseType)
		if err != nil {
			return err
		}

		if rpc.StreamsResponse && !rpc.StreamsRequest {
			//
			// func (w *wrapMyServiceClient) MyRPC(ctx context.Context, in *MyReq, opts ...grpc.CallOption) (MyService_MyRPCClient, error)
			//
			g.FService().P("func (w *", wrapClientName, ") ", rpc.Name, "(ctx ", ctx_alias, ".Context, in ", tinfo_req.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", opts ...", grpc_alias, ".CallOption) (", svcName, "_", rpc.Name, "Client, error) {")

		} else if rpc.StreamsResponse || rpc.StreamsRequest {
			//
			// func (w *wrapMyServiceClient) MyRPC(ctx context.Context, opts ...grpc.CallOption) (MyService_MyRPCClient, error)
			//
			g.FService().P("func (w *", wrapClientName, ") ", rpc.Name, "(ctx ", ctx_alias, ".Context, opts ...", grpc_alias, ".CallOption) (", svcName, "_", rpc.Name, "Client, error) {")

		} else {
			//
			// func (w *wrapMyServiceClient) MyRPC(ctx context.Context, in *MyReq, opts ...grpc.CallOption) (*MyResp, error)
			//
			g.FService().P("func (w *", wrapClientName, ") ", rpc.Name, "(ctx ", ctx_alias, ".Context, in ", tinfo_req.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", opts ...", grpc_alias, ".CallOption) (", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", error) {")
		}

		g.FService().In()
		g.FService().P("var err error")

		g.FService().P()

		// default return value
		defretvalue := tinfo_resp.Converter().TypeName(g.FService(), TNT_EMPTYVALUE, 0)
		// default return value
		defretnilvalue := tinfo_resp.Converter().TypeName(g.FService(), TNT_EMPTYORNILVALUE, 0)

		// if stream request or response, return is always an interface
		if rpc.StreamsResponse || rpc.StreamsRequest {
			defretvalue = "nil"
		}

		var check_error bool

		// convert request
		if !rpc.StreamsRequest {
			g.FService().P("var wreq ", tinfo_req.Source().TypeName(g.FService(), TNT_TYPENAME, 0))

			check_error, err = tinfo_req.Converter().GenerateExport(g.FService(), "in", "wreq", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheck(g, defretvalue)
			}

			g.FService().P("if wreq == nil {")
			g.FService().In()
			g.FService().P("wreq = ", tinfo_req.Source().TypeName(g.FService(), TNT_EMPTYVALUE, 0))
			g.FService().Out()
			g.FService().P("}")

			g.FService().P()
		}

		// call
		if !rpc.StreamsRequest {
			g.FService().P("resp, err := w.cli.", rpc.Name, "(ctx, wreq, opts...)")
		} else {
			g.FService().P("resp, err := w.cli.", rpc.Name, "(ctx, opts...)")
		}

		s.generateErrorCheck(g, defretvalue)
		g.FService().P()

		// convert response
		if !rpc.StreamsResponse && !rpc.StreamsRequest {
			g.FService().P("var wresp ", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0))

			check_error, err = tinfo_resp.Converter().GenerateImport(g.FService(), "resp", "wresp", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheck(g, defretvalue)
			}
			g.FService().P()

			// Return response
			g.FService().P("return wresp, nil")
		} else {
			// return stream wrapper
			g.FService().P("return &wrap", svcName, "_", rpc.Name, "Client{cli: resp}, nil")
		}

		g.FService().Out()
		g.FService().P("}")
		g.FService().P()

		// Generate stream request / response
		if rpc.StreamsRequest || rpc.StreamsResponse {
			rpcClientStruct := svcName + "_" + rpc.Name + "Client"

			//
			// type MyService_MyRPCClient interface
			//
			g.FService().P("type ", rpcClientStruct, " interface {")
			g.FService().In()

			if rpc.StreamsRequest {
				//
				// Send(*MyReq) error
				//
				g.FService().P("Send(", tinfo_req.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ") error")
			}
			if rpc.StreamsResponse {
				//
				// Recv() (*MyResp, error)
				//
				g.FService().P("Recv() (", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", error)")
			}
			if rpc.StreamsRequest && !rpc.StreamsResponse {
				//
				// CloseAndRecv() (*MyResp, error)
				//
				g.FService().P("CloseAndRecv() (", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", error)")
			}

			g.FService().P("grpc.ClientStream")
			g.FService().Out()
			g.FService().P("}")

			g.FService().P()

			wrapRPCClientName := "wrap" + svcName + "_" + rpc.Name + "Client"

			//
			// type wrapMyServiceMyRPCClient struct
			//
			g.FService().P("type ", wrapRPCClientName, " struct {")
			g.FService().In()

			g.FService().P("cli ", func_alias, ".", svcName, "_", rpc.Name, "Client")

			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			if rpc.StreamsRequest {
				//
				// func (w *wrapMyServiceMyRPCClient) Send(*MyReq) error
				//
				g.FService().P("func (w *", wrapRPCClientName, ") Send(m ", tinfo_req.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ") error {")
				g.FService().In()

				g.FService().P("var err error")

				// convert request
				g.FService().P("var wreq ", tinfo_req.Source().TypeName(g.FService(), TNT_TYPENAME, 0))

				check_error, err := tinfo_req.Converter().GenerateExport(g.FService(), "m", "wreq", "err")
				if err != nil {
					return err
				}
				if check_error {
					s.generateErrorCheck(g, "")
				}

				g.FService().P()

				g.FService().P("return w.cli.Send(wreq)")

				g.FService().Out()
				g.FService().P("}")
				g.FService().P()
			}

			// Always have Recv or CloseAndRecv
			if rpc.StreamsResponse {
				//
				// func (w *wrapMyServiceMyRPCClient) Recv() (*MyResp, error)
				//
				g.FService().P("func (w *", wrapRPCClientName, ") Recv() (", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", error) {")
				g.FService().In()
				g.FService().P("resp, err := w.cli.Recv()")
			} else {
				//
				// func (w *wrapMyServiceMyRPCClient) CloseAndRecv() (*MyResp, error)
				//
				g.FService().P("func (w *", wrapRPCClientName, ") CloseAndRecv() (", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", error) {")
				g.FService().In()
				g.FService().P("resp, err := w.cli.CloseAndRecv()")
			}

			// get and convert response
			s.generateErrorCheck(g, defretnilvalue)
			g.FService().P()

			g.FService().P("var wresp ", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0))

			check_error, err = tinfo_resp.Converter().GenerateImport(g.FService(), "resp", "wresp", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheck(g, defretnilvalue)
			}

			g.FService().P()

			g.FService().P("return wresp, nil")

			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			// Generate grpc.ClientStream methods
			g.FService().P("// grpc.ClientStream")

			metadata_alias := g.FService().DeclDep("google.golang.org/grpc/metadata", "metadata")

			g.FService().P("func (w *", wrapRPCClientName, ") Header() (", metadata_alias, ".MD, error) {")
			g.FService().In()
			g.FService().P("return w.cli.Header()")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			g.FService().P("func (w *", wrapRPCClientName, ") Trailer() ", metadata_alias, ".MD {")
			g.FService().In()
			g.FService().P("return w.cli.Trailer()")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			g.FService().P("func (w *", wrapRPCClientName, ") CloseSend() error {")
			g.FService().In()
			g.FService().P("return w.cli.CloseSend()")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			// Generate grpc.Stream methods
			g.FService().P("// grpc.Stream")

			g.FService().P("func (w *", wrapRPCClientName, ") Context() ", ctx_alias, ".Context {")
			g.FService().In()
			g.FService().P("return w.cli.Context()")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			g.FService().P("func (w *", wrapRPCClientName, ") SendMsg(m interface{}) error {")
			g.FService().In()
			g.FService().P("return w.cli.SendMsg(m)")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			g.FService().P("func (w *", wrapRPCClientName, ") RecvMsg(m interface{}) error {")
			g.FService().In()
			g.FService().P("return w.cli.RecvMsg(m)")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()
		}
	}

	//
	// SERVER
	//

	//
	// type MyServiceServer interface
	//

	if !g.FService().GenerateComment(svc.Comment) {
		g.FService().P()
		g.FService().P("// Server API for ", svcName, " service")
		g.FService().P()
	}

	g.FService().P("type ", svcName, "Server interface {")
	g.FService().In()

	for _, rpc := range svc.RPCs {
		tinfo_req, err := g.GetTypeInfoFromParent(tp_svc, rpc.RequestType)
		if err != nil {
			return err
		}
		tinfo_resp, err := g.GetTypeInfoFromParent(tp_svc, rpc.ResponseType)
		if err != nil {
			return err
		}

		g.FService().GenerateComment(rpc.Comment)

		if !rpc.StreamsRequest && rpc.StreamsResponse {
			//
			// MyRPC(*MyReq, MyService_MyRRPCServer) error
			//
			g.FService().P(rpc.Name, "(", tinfo_req.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", ", svcName, "_", rpc.Name, "Server) error")
		} else if rpc.StreamsRequest || rpc.StreamsResponse {
			//
			// MyRPC(MyService_MyRRPCServer) error
			//
			g.FService().P(rpc.Name, "(", svcName, "_", rpc.Name, "Server) error")
		} else {
			//
			// MyRPC(ctx.Context, *MyReq) (*MyResp, error)
			//
			g.FService().P(rpc.Name, "(", ctx_alias, ".Context, ", tinfo_req.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ") (", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", error)")
		}
	}

	g.FService().Out()
	g.FService().P("}")
	g.FService().P()

	//
	// type wrapMyServiceServer struct
	//

	wrapServerName := "wrap" + svcName + "Server"

	g.FService().P("type ", wrapServerName, " struct {")
	g.FService().In()

	g.FService().P("srv ", svcName, "Server")
	g.FService().P("opts ", util_alias, ".RegServerOptions")
	g.FService().Out()
	g.FService().P("}")
	g.FService().P()

	//
	// func NewWrapMyServiceServer(srv MyServiceServer, opts ...fproto_gowrap_util.RegServerOption) *wrapMyServiceServer
	//

	g.FService().P("func NewWrap", svcName, "Server(srv ", svcName, "Server, opts ...", util_alias, ".RegServerOption) *wrap", svcName, "Server {")
	g.FService().In()

	g.FService().P("w := &", wrapServerName, "{srv: srv}")
	g.FService().P("for _, o := range opts {")
	g.FService().In()
	g.FService().P("o(&w.opts)")
	g.FService().Out()
	g.FService().P("}")

	g.FService().P("return w")

	g.FService().Out()
	g.FService().P("}")
	g.FService().P()

	var errVar string
	if s.WrapErrors {
		//
		// func (w *wrapMyServiceServer) wrapError(ServerErrorType, error) error
		//
		g.FService().P("func (w *", wrapServerName, ") wrapError(errorType ", util_alias, ".ServerErrorType, err error) error {")
		g.FService().In()

		g.FService().P("if w.opts.ErrorWrapper != nil {")
		g.FService().In()
		g.FService().P("return w.opts.ErrorWrapper.WrapError(errorType, err)")
		g.FService().Out()
		g.FService().P("} else {")
		g.FService().In()
		g.FService().P("return err")
		g.FService().Out()
		g.FService().P("}")

		g.FService().Out()
		g.FService().P("}")
		g.FService().P()

		errVar = "w.wrapError(" + util_alias + ".%%ERROR_TYPE%%, err)"
	} else {
		errVar = "err"
	}

	// Generate RPCs
	for _, rpc := range svc.RPCs {
		tinfo_req, err := g.GetTypeInfoFromParent(tp_svc, rpc.RequestType)
		if err != nil {
			return err
		}
		tinfo_resp, err := g.GetTypeInfoFromParent(tp_svc, rpc.ResponseType)
		if err != nil {
			return err
		}

		if !rpc.StreamsRequest && rpc.StreamsResponse {
			//
			// func (w *wrapMyServiceServer) MyRPC(*myapp.MyReq, myapp.MyService_MyRRPCServer) error
			//
			g.FService().P("func (w *", wrapServerName, ") ", rpc.Name, "(req ", tinfo_req.Source().TypeName(g.FService(), TNT_TYPENAME, 0), ", stream ", func_alias, ".", svcName, "_", rpc.Name, "Server) error {")
		} else if rpc.StreamsRequest || rpc.StreamsResponse {
			//
			// func (w *wrapMyServiceServer) MyRPC(stream myapp.MyService_MyRRPCServer) error
			//
			g.FService().P("func (w *", wrapServerName, ") ", rpc.Name, "(stream ", func_alias, ".", svcName, "_", rpc.Name, "Server) error {")
		} else {
			//
			// func (w *wrapMyServiceServer) MyRPC(ctx context.Context, req *myapp.MyReq) (*myapp.MyResp, error)
			//
			g.FService().P("func (w *", wrapServerName, ") ", rpc.Name, "(ctx ", ctx_alias, ".Context, req ", tinfo_req.Source().TypeName(g.FService(), TNT_TYPENAME, 0), ") (", tinfo_resp.Source().TypeName(g.FService(), TNT_TYPENAME, 0), ", error) {")
		}

		g.FService().In()
		g.FService().P("var err error")

		g.FService().P()

		// default return value
		defretvalue := tinfo_resp.Source().TypeName(g.FService(), TNT_EMPTYVALUE, 0)
		send_defretvalue := defretvalue
		if rpc.StreamsRequest || rpc.StreamsResponse {
			defretvalue = ""
		}

		// default return value
		defretnilvalue := tinfo_resp.Source().TypeName(g.FService(), TNT_EMPTYORNILVALUE, 0)

		if !rpc.StreamsRequest {
			// convert request
			g.FService().P("var wreq ", tinfo_req.Converter().TypeName(g.FService(), TNT_TYPENAME, 0))

			check_error, err := tinfo_req.Converter().GenerateImport(g.FService(), "req", "wreq", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheckCustomError(g, defretvalue, strings.Replace(errVar, "%%ERROR_TYPE%%", "SET_IMPORT", -1))
			}

			g.FService().P()
		}

		// call
		if !rpc.StreamsRequest && rpc.StreamsResponse {
			g.FService().P("err = w.srv.", rpc.Name, "(wreq, &wrap", svcName, "_", rpc.Name, "Server{srv: stream})")
		} else if rpc.StreamsRequest || rpc.StreamsResponse {
			g.FService().P("err = w.srv.", rpc.Name, "(&wrap", svcName, "_", rpc.Name, "Server{srv: stream})")
		} else {
			g.FService().P("resp, err := w.srv.", rpc.Name, "(ctx, wreq)")
		}

		s.generateErrorCheckCustomError(g, defretvalue, strings.Replace(errVar, "%%ERROR_TYPE%%", "SET_CALL", -1))
		g.FService().P()

		// convert response
		if !rpc.StreamsRequest && !rpc.StreamsResponse {
			// Allows returning nil from server
			if tinfo_resp.Converter().IsPointer() && defretvalue != "" {
				g.FService().P("if resp == nil {")
				g.FService().In()
				g.FService().P("return ", defretvalue, ", nil")
				g.FService().Out()
				g.FService().P("}")
			}

			g.FService().P("var wresp ", tinfo_resp.Source().TypeName(g.FService(), TNT_TYPENAME, 0))

			check_error, err := tinfo_resp.Converter().GenerateExport(g.FService(), "resp", "wresp", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheckCustomError(g, defretvalue, strings.Replace(errVar, "%%ERROR_TYPE%%", "SET_EXPORT", -1))
			}

			g.FService().P()
		}

		// return response
		if defretvalue != "" {
			g.FService().P("return wresp, nil")
		} else {
			g.FService().P("return nil")
		}

		g.FService().Out()
		g.FService().P("}")
		g.FService().P()

		// Generate stream request / response
		if rpc.StreamsRequest || rpc.StreamsResponse {
			//
			// type MyService_MyRPCServer interface
			//

			g.FService().P("type ", svcName, "_", rpc.Name, "Server interface {")
			g.FService().In()

			if rpc.StreamsRequest {
				//
				// Recv() (*MyReq, error)
				//
				g.FService().P("Recv() (", tinfo_req.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", error)")
			}
			if rpc.StreamsResponse {
				//
				// Send(*MyResp) error
				//
				g.FService().P("Send(", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ") error")
			}
			if rpc.StreamsRequest && !rpc.StreamsResponse {
				//
				// SendAndClose(*MyResp) error
				//
				g.FService().P("SendAndClose(", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ") error")
			}

			g.FService().P("grpc.ServerStream")
			g.FService().Out()
			g.FService().P("}")

			g.FService().P()

			//
			// type wrapMyService_MyRPCServer struct
			//
			wrapRPCServerName := "wrap" + svcName + "_" + rpc.Name + "Server"

			g.FService().P("type ", wrapRPCServerName, " struct {")
			g.FService().In()

			g.FService().P("srv ", func_alias, ".", svcName, "_", rpc.Name, "Server")

			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			if rpc.StreamsRequest {
				//
				// func (w *wrapMyServiceMyRPCServer) Send(*MyReq) error
				//
				g.FService().P("func (w *", wrapRPCServerName, ") Recv() (", tinfo_req.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ", error) {")
				g.FService().In()

				g.FService().P("var err error")

				g.FService().P("req, err := w.srv.Recv()")
				s.generateErrorCheck(g, defretnilvalue)
				g.FService().P()

				// convert request
				g.FService().P("var wreq ", tinfo_req.Converter().TypeName(g.FService(), TNT_TYPENAME, 0))

				check_error, err := tinfo_req.Converter().GenerateImport(g.FService(), "req", "wreq", "err")
				if err != nil {
					return err
				}
				if check_error {
					s.generateErrorCheck(g, defretnilvalue)
				}

				g.FService().P()
				g.FService().P("return wreq, nil")
				g.FService().Out()
				g.FService().P("}")
				g.FService().P()
			}

			// Always have Send or SendAndClose
			if rpc.StreamsResponse {
				//
				// func (w *wrapMyServiceMyRPCServer) Send(*MyResp)  error
				//
				g.FService().P("func (w *", wrapRPCServerName, ") Send(resp ", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ") error {")
			} else {
				//
				// func (w *wrapMyServiceMyRPCServer) CloseAndRecv() (*MyResp, error)
				//
				g.FService().P("func (w *", wrapRPCServerName, ") SendAndClose(resp ", tinfo_resp.Converter().TypeName(g.FService(), TNT_TYPENAME, 0), ") error {")
			}

			g.FService().In()
			g.FService().P("var err error")
			g.FService().P("var wresp ", tinfo_resp.Source().TypeName(g.FService(), TNT_TYPENAME, 0))

			// Allows returning nil from server
			if tinfo_resp.Converter().IsPointer() && send_defretvalue != "" {
				g.FService().P("if resp == nil {")
				g.FService().In()
				g.FService().P("wresp = ", send_defretvalue)
				g.FService().Out()
				g.FService().P("} else {")
				g.FService().In()
			}

			check_error, err := tinfo_resp.Converter().GenerateExport(g.FService(), "resp", "wresp", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheck(g, "")
			}

			// Allows returning nil from server
			if tinfo_resp.Converter().IsPointer() && send_defretvalue != "" {
				g.FService().Out()
				g.FService().P("}")
			}

			g.FService().P()

			if rpc.StreamsResponse {
				g.FService().P("err = w.srv.Send(wresp)")
			} else {
				g.FService().P("err = w.srv.SendAndClose(wresp)")
			}

			// get and convert response
			s.generateErrorCheck(g, "")
			g.FService().P()
			g.FService().P("return nil")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			// Generate grpc.ServerStream methods
			g.FService().P("// grpc.ServerStream")

			metadata_alias := g.FService().DeclDep("google.golang.org/grpc/metadata", "metadata")

			g.FService().P("func (w *", wrapRPCServerName, ") SetHeader(md ", metadata_alias, ".MD) error {")
			g.FService().In()
			g.FService().P("return w.srv.SetHeader(md)")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			g.FService().P("func (w *", wrapRPCServerName, ") SendHeader(md ", metadata_alias, ".MD) error {")
			g.FService().In()
			g.FService().P("return w.srv.SendHeader(md)")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			g.FService().P("func (w *", wrapRPCServerName, ") SetTrailer(md ", metadata_alias, ".MD) {")
			g.FService().In()
			g.FService().P("w.srv.SetTrailer(md)")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			// Generate grpc.Stream methods
			g.FService().P("// grpc.Stream")

			g.FService().P("func (w *", wrapRPCServerName, ") Context() ", ctx_alias, ".Context {")
			g.FService().In()
			g.FService().P("return w.srv.Context()")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			g.FService().P("func (w *", wrapRPCServerName, ") SendMsg(m interface{}) error {")
			g.FService().In()
			g.FService().P("return w.srv.SendMsg(m)")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

			g.FService().P("func (w *", wrapRPCServerName, ") RecvMsg(m interface{}) error {")
			g.FService().In()
			g.FService().P("return w.srv.RecvMsg(m)")
			g.FService().Out()
			g.FService().P("}")
			g.FService().P()

		}
	}

	//
	// func RegisterMyServiceServer(s *grpc.Server, srv MyServiceServer)
	//

	g.FService().P("func Register", svcName, "Server(s *", grpc_alias, ".Server, srv ", svcName, "Server, opts ...", util_alias, ".RegServerOption) {")
	g.FService().In()

	// myapp.RegisterMyServiceServer(s, NewWrapMyServiceServer(srv))
	g.FService().P(func_alias, ".Register", svcName, "Server(s, NewWrap", svcName, "Server(srv, opts...))")

	g.FService().Out()
	g.FService().P("}")

	g.FService().P()

	return nil
}

func (s *ServiceGen_gRPC) generateErrorCheck(g *Generator, extraRetVal string) {
	g.FService().P("if err != nil {")
	g.FService().In()
	if extraRetVal != "" {
		g.FService().P("return ", extraRetVal, ", err")
	} else {
		g.FService().P("return err")
	}
	g.FService().Out()
	g.FService().P("}")
}

func (s *ServiceGen_gRPC) generateErrorCheckCustomError(g *Generator, extraRetVal string, err string) {
	g.FService().P("if err != nil {")
	g.FService().In()
	if extraRetVal != "" {
		g.FService().P("return ", extraRetVal, ", ", err)
	} else {
		g.FService().P("return ", err)
	}
	g.FService().Out()
	g.FService().P("}")
}
