package fproto_gowrap

import (
	"strings"

	"github.com/RangelReale/fproto"
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
	ctx_alias := g.FService().Dep("golang.org/x/net/context", "context")
	grpc_alias := g.FService().Dep("google.golang.org/grpc", "grpc")
	var util_alias string
	util_alias = g.FService().Dep("github.com/RangelReale/fproto-wrap/gowrap/util", "fproto_gowrap_util")
	func_alias := g.FService().FileDep(nil, "", false)

	svcName := CamelCase(svc.Name)

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
		tc_req, err := g.GetGowrapType("", rpc.RequestType)
		if err != nil {
			return err
		}
		tc_resp, err := g.GetGowrapType("", rpc.ResponseType)
		if err != nil {
			return err
		}

		g.FService().GenerateComment(rpc.Comment)

		if rpc.StreamsResponse && !rpc.StreamsRequest {
			//
			// MyRPC(ctx context.Context, in *MyReq, opts ...grpc.CallOption) (MyService_MyRPCClient, error)
			//
			g.FService().P(rpc.Name, "(ctx ", ctx_alias, ".Context, in ", tc_req.TypeName(g.FService(), TNT_TYPENAME), ", opts ...", grpc_alias, ".CallOption) (", svcName, "_", rpc.Name, "Client, error)")

		} else if rpc.StreamsResponse || rpc.StreamsRequest {
			//
			// MyRPC(ctx context.Context, opts ...grpc.CallOption) (MyService_MyRPCClient, error)
			//
			g.FService().P(rpc.Name, "(ctx ", ctx_alias, ".Context, opts ...", grpc_alias, ".CallOption) (", svcName, "_", rpc.Name, "Client, error)")

		} else {
			//
			// MyRPC(ctx context.Context, in *MyReq, opts ...grpc.CallOption) (*MyResp, error)
			//
			g.FService().P(rpc.Name, "(ctx ", ctx_alias, ".Context, in ", tc_req.TypeName(g.FService(), TNT_TYPENAME), ", opts ...", grpc_alias, ".CallOption) (", tc_resp.TypeName(g.FService(), TNT_TYPENAME), ", error)")
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

	g.FService().P("w := &", wrapClientName, "{cli: ", func_alias, ".New", svcName, "Client(cc)}")
	g.FService().P("return w")

	g.FService().Out()
	g.FService().P("}")
	g.FService().P()

	// Implement each RPC wrapper

	for _, rpc := range svc.RPCs {
		tc_req, tcgo_req, err := g.GetBothTypes("", rpc.RequestType)
		if err != nil {
			return err
		}
		tc_resp, err := g.GetGowrapType("", rpc.ResponseType)
		if err != nil {
			return err
		}

		if rpc.StreamsResponse && !rpc.StreamsRequest {
			//
			// func (w *wrapMyServiceClient) MyRPC(ctx context.Context, in *MyReq, opts ...grpc.CallOption) (MyService_MyRPCClient, error)
			//
			g.FService().P("func (w *", wrapClientName, ") ", rpc.Name, "(ctx ", ctx_alias, ".Context, in ", tc_req.TypeName(g.FService(), TNT_TYPENAME), ", opts ...", grpc_alias, ".CallOption) (", svcName, "_", rpc.Name, "Client, error) {")

		} else if rpc.StreamsResponse || rpc.StreamsRequest {
			//
			// func (w *wrapMyServiceClient) MyRPC(ctx context.Context, opts ...grpc.CallOption) (MyService_MyRPCClient, error)
			//
			g.FService().P("func (w *", wrapClientName, ") ", rpc.Name, "(ctx ", ctx_alias, ".Context, opts ...", grpc_alias, ".CallOption) (", svcName, "_", rpc.Name, "Client, error) {")

		} else {
			//
			// func (w *wrapMyServiceClient) MyRPC(ctx context.Context, in *MyReq, opts ...grpc.CallOption) (*MyResp, error)
			//
			g.FService().P("func (w *", wrapClientName, ") ", rpc.Name, "(ctx ", ctx_alias, ".Context, in ", tc_req.TypeName(g.FService(), TNT_TYPENAME), ", opts ...", grpc_alias, ".CallOption) (", tc_resp.TypeName(g.FService(), TNT_TYPENAME), ", error) {")
		}

		g.FService().In()
		g.FService().P("var err error")

		g.FService().P()

		// default return value
		defretvalue := tc_resp.TypeName(g.FService(), TNT_EMPTYVALUE)
		// default return value
		defretnilvalue := tc_resp.TypeName(g.FService(), TNT_EMPTYORNILVALUE)

		// if stream request or response, return is always an interface
		if rpc.StreamsResponse || rpc.StreamsRequest {
			defretvalue = "nil"
		}

		var check_error bool

		// convert request
		if !rpc.StreamsRequest {
			g.FService().P("var wreq ", tcgo_req.TypeName(g.FService(), TNT_TYPENAME))

			check_error, err = tc_req.GenerateExport(g.FService(), "in", "wreq", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheck(g, defretvalue)
			}

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
			g.FService().P("var wresp ", tc_resp.TypeName(g.FService(), TNT_TYPENAME))

			check_error, err = tc_resp.GenerateImport(g.FService(), "resp", "wresp", "err")
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
			g.FService().P("return &wrap", svcName, rpc.Name, "Client{cli: resp}, nil")
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
				g.FService().P("Send(", tc_req.TypeName(g.FService(), TNT_TYPENAME), ") error")
			}
			if rpc.StreamsResponse {
				//
				// Recv() (*MyResp, error)
				//
				g.FService().P("Recv() (", tc_resp.TypeName(g.FService(), TNT_TYPENAME), ", error)")
			}
			if rpc.StreamsRequest && !rpc.StreamsResponse {
				//
				// CloseAndRecv() (*MyResp, error)
				//
				g.FService().P("CloseAndRecv() (", tc_resp.TypeName(g.FService(), TNT_TYPENAME), ", error)")
			}

			g.FService().Out()
			g.FService().P("}")

			g.FService().P()

			wrapRPCClientName := "wrap" + svcName + rpc.Name + "Client"

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
				g.FService().P("func (w *", wrapRPCClientName, ") Send(m ", tc_req.TypeName(g.FService(), TNT_TYPENAME), ") error {")
				g.FService().In()

				g.FService().P("var err error")

				// convert request
				g.FService().P("var wreq ", tcgo_req.TypeName(g.FService(), TNT_TYPENAME))

				check_error, err := tc_req.GenerateExport(g.FService(), "m", "wreq", "err")
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
				g.FService().P("func (w *", wrapRPCClientName, ") Recv() (", tc_resp.TypeName(g.FService(), TNT_TYPENAME), ", error) {")
				g.FService().In()
				g.FService().P("resp, err := w.cli.Recv()")
			} else {
				//
				// func (w *wrapMyServiceMyRPCClient) CloseAndRecv() (*MyResp, error)
				//
				g.FService().P("func (w *", wrapRPCClientName, ") CloseAndRecv() (", tc_resp.TypeName(g.FService(), TNT_TYPENAME), ", error) {")
				g.FService().In()
				g.FService().P("resp, err := w.cli.CloseAndRecv()")
			}

			// get and convert response
			s.generateErrorCheck(g, defretnilvalue)
			g.FService().P()

			g.FService().P("var wresp ", tc_resp.TypeName(g.FService(), TNT_TYPENAME))

			check_error, err = tc_resp.GenerateImport(g.FService(), "resp", "wresp", "err")
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
		tc_req, err := g.GetGowrapType("", rpc.RequestType)
		if err != nil {
			return err
		}
		tc_resp, err := g.GetGowrapType("", rpc.ResponseType)
		if err != nil {
			return err
		}

		g.FService().GenerateComment(rpc.Comment)

		if !rpc.StreamsRequest && rpc.StreamsResponse {
			//
			// MyRPC(*MyReq, MyService_MyRRPCServer) error
			//
			g.FService().P(rpc.Name, "(", tc_req.TypeName(g.FService(), TNT_TYPENAME), ", ", svcName, "_", rpc.Name, "Server) error")
		} else if rpc.StreamsRequest || rpc.StreamsResponse {
			//
			// MyRPC(MyService_MyRRPCServer) error
			//
			g.FService().P(rpc.Name, "(", svcName, "_", rpc.Name, "Server) error")
		} else {
			//
			// MyRPC(ctx.Context, *MyReq) (*MyResp, error)
			//
			g.FService().P(rpc.Name, "(", ctx_alias, ".Context, ", tc_req.TypeName(g.FService(), TNT_TYPENAME), ") (", tc_resp.TypeName(g.FService(), TNT_TYPENAME), ", error)")
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
	// func newWrapMyServiceServer(srv MyServiceServer, opts ...fproto_gowrap_util.RegServerOption) *wrapMyServiceServer
	//

	g.FService().P("func newWrap", svcName, "Server(srv ", svcName, "Server, opts ...", util_alias, ".RegServerOption) *wrap", svcName, "Server {")
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
		tc_req, tcgo_req, err := g.GetBothTypes("", rpc.RequestType)
		if err != nil {
			return err
		}
		tc_resp, tcgo_resp, err := g.GetBothTypes("", rpc.ResponseType)
		if err != nil {
			return err
		}

		if !rpc.StreamsRequest && rpc.StreamsResponse {
			//
			// func (w *wrapMyServiceServer) MyRPC(*myapp.MyReq, myapp.MyService_MyRRPCServer) error
			//
			g.FService().P("func (w *", wrapServerName, ") ", rpc.Name, "(req ", tcgo_req.TypeName(g.FService(), TNT_TYPENAME), ", stream ", func_alias, ".", svcName, "_", rpc.Name, "Server) error {")
		} else if rpc.StreamsRequest || rpc.StreamsResponse {
			//
			// func (w *wrapMyServiceServer) MyRPC(stream myapp.MyService_MyRRPCServer) error
			//
			g.FService().P("func (w *", wrapServerName, ") ", rpc.Name, "(stream ", func_alias, ".", svcName, "_", rpc.Name, "Server) error {")
		} else {
			//
			// func (w *wrapMyServiceServer) MyRPC(ctx context.Context, req *myapp.MyReq) (*myapp.MyResp, error)
			//
			g.FService().P("func (w *", wrapServerName, ") ", rpc.Name, "(ctx ", ctx_alias, ".Context, req ", tcgo_req.TypeName(g.FService(), TNT_TYPENAME), ") (", tcgo_resp.TypeName(g.FService(), TNT_TYPENAME), ", error) {")
		}

		g.FService().In()
		g.FService().P("var err error")

		g.FService().P()

		// default return value
		defretvalue := tcgo_resp.TypeName(g.FService(), TNT_EMPTYVALUE)
		send_defretvalue := defretvalue
		if rpc.StreamsRequest || rpc.StreamsResponse {
			defretvalue = ""
		}

		// default return value
		defretnilvalue := tcgo_resp.TypeName(g.FService(), TNT_EMPTYORNILVALUE)

		if !rpc.StreamsRequest {
			// convert request
			g.FService().P("var wreq ", tc_req.TypeName(g.FService(), TNT_TYPENAME))

			check_error, err := tc_req.GenerateImport(g.FService(), "req", "wreq", "err")
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
			g.FService().P("err = w.srv.", rpc.Name, "(wreq, &wrap", svcName, rpc.Name, "Server{srv: stream})")
		} else if rpc.StreamsRequest || rpc.StreamsResponse {
			g.FService().P("err = w.srv.", rpc.Name, "(&wrap", svcName, rpc.Name, "Server{srv: stream})")
		} else {
			g.FService().P("resp, err := w.srv.", rpc.Name, "(ctx, wreq)")
		}

		s.generateErrorCheckCustomError(g, defretvalue, strings.Replace(errVar, "%%ERROR_TYPE%%", "SET_CALL", -1))
		g.FService().P()

		// convert response
		if !rpc.StreamsRequest && !rpc.StreamsResponse {
			// Allows returning nil from server
			if tc_resp.IsPointer() && defretvalue != "" {
				g.FService().P("if resp == nil {")
				g.FService().In()
				g.FService().P("return ", defretvalue, ", nil")
				g.FService().Out()
				g.FService().P("}")
			}

			g.FService().P("var wresp ", tcgo_resp.TypeName(g.FService(), TNT_TYPENAME))

			check_error, err := tc_resp.GenerateExport(g.FService(), "resp", "wresp", "err")
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
				g.FService().P("Recv() (", tc_req.TypeName(g.FService(), TNT_TYPENAME), ", error)")
			}
			if rpc.StreamsResponse {
				//
				// Send(*MyResp) error
				//
				g.FService().P("Send(", tc_resp.TypeName(g.FService(), TNT_TYPENAME), ") error")
			}
			if rpc.StreamsRequest && !rpc.StreamsResponse {
				//
				// SendAndClose(*MyResp) error
				//
				g.FService().P("SendAndClose(", tc_resp.TypeName(g.FService(), TNT_TYPENAME), ") error")
			}

			g.FService().Out()
			g.FService().P("}")

			g.FService().P()

			//
			// type wrapMyServiceMyRPCServer struct
			//
			wrapRPCServerName := "wrap" + svcName + rpc.Name + "Server"

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
				g.FService().P("func (w *", wrapRPCServerName, ") Recv() (", tc_req.TypeName(g.FService(), TNT_TYPENAME), ", error) {")
				g.FService().In()

				g.FService().P("var err error")

				g.FService().P("req, err := w.srv.Recv()")
				s.generateErrorCheck(g, defretnilvalue)
				g.FService().P()

				// convert request
				g.FService().P("var wreq ", tc_req.TypeName(g.FService(), TNT_TYPENAME))

				check_error, err := tc_req.GenerateImport(g.FService(), "req", "wreq", "err")
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
				g.FService().P("func (w *", wrapRPCServerName, ") Send(resp ", tc_resp.TypeName(g.FService(), TNT_TYPENAME), ") error {")
			} else {
				//
				// func (w *wrapMyServiceMyRPCServer) CloseAndRecv() (*MyResp, error)
				//
				g.FService().P("func (w *", wrapRPCServerName, ") SendAndClose(resp ", tc_resp.TypeName(g.FService(), TNT_TYPENAME), ") error {")
			}

			g.FService().In()
			g.FService().P("var err error")
			g.FService().P("var wresp ", tcgo_resp.TypeName(g.FService(), TNT_TYPENAME))

			// Allows returning nil from server
			if tc_resp.IsPointer() && send_defretvalue != "" {
				g.FService().P("if resp == nil {")
				g.FService().In()
				g.FService().P("wresp = ", send_defretvalue)
				g.FService().Out()
				g.FService().P("} else {")
				g.FService().In()
			}

			check_error, err := tc_resp.GenerateExport(g.FService(), "resp", "wresp", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheck(g, "")
			}

			// Allows returning nil from server
			if tc_resp.IsPointer() && send_defretvalue != "" {
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
		}
	}

	//
	// func RegisterMyServiceServer(s *grpc.Server, srv MyServiceServer)
	//

	g.FService().P("func Register", svcName, "Server(s *", grpc_alias, ".Server, srv ", svcName, "Server, opts ...", util_alias, ".RegServerOption) {")
	g.FService().In()

	// myapp.RegisterMyServiceServer(s, newWrapMyServiceServer(srv))
	g.FService().P(func_alias, ".Register", svcName, "Server(s, newWrap", svcName, "Server(srv, opts...))")

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
