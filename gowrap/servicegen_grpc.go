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
	ctx_alias := g.Dep("golang.org/x/net/context", "context")
	grpc_alias := g.Dep("google.golang.org/grpc", "grpc")
	var util_alias string
	util_alias = g.Dep("github.com/RangelReale/fproto-gowrap/util", "fproto_gowrap_util")
	func_alias := g.FileDep(nil, "", false)

	svcName := CamelCase(svc.Name)

	//
	// CLIENT
	//

	//
	// type MyServiceClient interface
	//
	if !g.GenerateComment(svc.Comment) {
		g.P()
		g.P("// Client API for ", svcName, " service")
		g.P()
	}

	g.P("type ", svcName, "Client interface {")
	g.In()

	for _, rpc := range svc.RPCs {
		tc_req, err := g.GetGowrapType("", rpc.RequestType)
		if err != nil {
			return err
		}
		tc_resp, err := g.GetGowrapType("", rpc.ResponseType)
		if err != nil {
			return err
		}

		g.GenerateComment(rpc.Comment)

		if rpc.StreamsResponse && !rpc.StreamsRequest {
			//
			// MyRPC(ctx context.Context, in *MyReq, opts ...grpc.CallOption) (MyService_MyRPCClient, error)
			//
			g.P(rpc.Name, "(ctx ", ctx_alias, ".Context, in ", tc_req.TypeName(g, TNT_TYPENAME), ", opts ...", grpc_alias, ".CallOption) (", svcName, "_", rpc.Name, "Client, error)")

		} else if rpc.StreamsResponse || rpc.StreamsRequest {
			//
			// MyRPC(ctx context.Context, opts ...grpc.CallOption) (MyService_MyRPCClient, error)
			//
			g.P(rpc.Name, "(ctx ", ctx_alias, ".Context, opts ...", grpc_alias, ".CallOption) (", svcName, "_", rpc.Name, "Client, error)")

		} else {
			//
			// MyRPC(ctx context.Context, in *MyReq, opts ...grpc.CallOption) (*MyResp, error)
			//
			g.P(rpc.Name, "(ctx ", ctx_alias, ".Context, in ", tc_req.TypeName(g, TNT_TYPENAME), ", opts ...", grpc_alias, ".CallOption) (", tc_resp.TypeName(g, TNT_TYPENAME), ", error)")
		}
	}

	g.Out()
	g.P("}")
	g.P()

	//
	// type wrapMyServiceClient struct
	//

	wrapClientName := "wrap" + svcName + "Client"

	g.P("type ", wrapClientName, " struct {")
	g.In()

	// the default Golang protobuf client
	g.P("cli ", func_alias, ".", svcName, "Client")

	g.Out()
	g.P("}")
	g.P()

	//
	// func NewMyServiceClient(cc *grpc.ClientConn, errorHandler ...wraputil.ServiceErrorHandler) MyServiceClient
	//

	g.P("func New", svcName, "Client(cc *", grpc_alias, ".ClientConn) ", svcName, "Client {")
	g.In()

	g.P("w := &", wrapClientName, "{cli: ", func_alias, ".New", svcName, "Client(cc)}")
	g.P("return w")

	g.Out()
	g.P("}")
	g.P()

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
			g.P("func (w *", wrapClientName, ") ", rpc.Name, "(ctx ", ctx_alias, ".Context, in ", tc_req.TypeName(g, TNT_TYPENAME), ", opts ...", grpc_alias, ".CallOption) (", svcName, "_", rpc.Name, "Client, error) {")

		} else if rpc.StreamsResponse || rpc.StreamsRequest {
			//
			// func (w *wrapMyServiceClient) MyRPC(ctx context.Context, opts ...grpc.CallOption) (MyService_MyRPCClient, error)
			//
			g.P("func (w *", wrapClientName, ") ", rpc.Name, "(ctx ", ctx_alias, ".Context, opts ...", grpc_alias, ".CallOption) (", svcName, "_", rpc.Name, "Client, error) {")

		} else {
			//
			// func (w *wrapMyServiceClient) MyRPC(ctx context.Context, in *MyReq, opts ...grpc.CallOption) (*MyResp, error)
			//
			g.P("func (w *", wrapClientName, ") ", rpc.Name, "(ctx ", ctx_alias, ".Context, in ", tc_req.TypeName(g, TNT_TYPENAME), ", opts ...", grpc_alias, ".CallOption) (", tc_resp.TypeName(g, TNT_TYPENAME), ", error) {")
		}

		g.In()
		g.P("var err error")

		g.P()

		// default return value
		defretvalue := tc_resp.TypeName(g, TNT_EMPTYVALUE)
		// default return value
		defretnilvalue := tc_resp.TypeName(g, TNT_EMPTYORNILVALUE)

		// if stream request or response, return is always an interface
		if rpc.StreamsResponse || rpc.StreamsRequest {
			defretvalue = "nil"
		}

		var check_error bool

		// convert request
		if !rpc.StreamsRequest {
			g.P("var wreq ", tcgo_req.TypeName(g, TNT_TYPENAME))

			check_error, err = tc_req.GenerateExport(g, "in", "wreq", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheck(g, defretvalue)
			}

			g.P()
		}

		// call
		if !rpc.StreamsRequest {
			g.P("resp, err := w.cli.", rpc.Name, "(ctx, wreq, opts...)")
		} else {
			g.P("resp, err := w.cli.", rpc.Name, "(ctx, opts...)")
		}

		s.generateErrorCheck(g, defretvalue)
		g.P()

		// convert response
		if !rpc.StreamsResponse && !rpc.StreamsRequest {
			g.P("var wresp ", tc_resp.TypeName(g, TNT_TYPENAME))

			check_error, err = tc_resp.GenerateImport(g, "resp", "wresp", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheck(g, defretvalue)
			}
			g.P()

			// Return response
			g.P("return wresp, nil")
		} else {
			// return stream wrapper
			g.P("return &wrap", svcName, rpc.Name, "Client{cli: resp}, nil")
		}

		g.Out()
		g.P("}")
		g.P()

		// Generate stream request / response
		if rpc.StreamsRequest || rpc.StreamsResponse {
			rpcClientStruct := svcName + "_" + rpc.Name + "Client"

			//
			// type MyService_MyRPCClient interface
			//
			g.P("type ", rpcClientStruct, " interface {")
			g.In()

			if rpc.StreamsRequest {
				//
				// Send(*MyReq) error
				//
				g.P("Send(", tc_req.TypeName(g, TNT_TYPENAME), ") error")
			}
			if rpc.StreamsResponse {
				//
				// Recv() (*MyResp, error)
				//
				g.P("Recv() (", tc_resp.TypeName(g, TNT_TYPENAME), ", error)")
			}
			if rpc.StreamsRequest && !rpc.StreamsResponse {
				//
				// CloseAndRecv() (*MyResp, error)
				//
				g.P("CloseAndRecv() (", tc_resp.TypeName(g, TNT_TYPENAME), ", error)")
			}

			g.Out()
			g.P("}")

			g.P()

			wrapRPCClientName := "wrap" + svcName + rpc.Name + "Client"

			//
			// type wrapMyServiceMyRPCClient struct
			//
			g.P("type ", wrapRPCClientName, " struct {")
			g.In()

			g.P("cli ", func_alias, ".", svcName, "_", rpc.Name, "Client")

			g.Out()
			g.P("}")
			g.P()

			if rpc.StreamsRequest {
				//
				// func (w *wrapMyServiceMyRPCClient) Send(*MyReq) error
				//
				g.P("func (w *", wrapRPCClientName, ") Send(m ", tc_req.TypeName(g, TNT_TYPENAME), ") error {")
				g.In()

				g.P("var err error")

				// convert request
				g.P("var wreq ", tcgo_req.TypeName(g, TNT_TYPENAME))

				check_error, err := tc_req.GenerateExport(g, "m", "wreq", "err")
				if err != nil {
					return err
				}
				if check_error {
					s.generateErrorCheck(g, "")
				}

				g.P()

				g.P("return w.cli.Send(wreq)")

				g.Out()
				g.P("}")
				g.P()
			}

			// Always have Recv or CloseAndRecv
			if rpc.StreamsResponse {
				//
				// func (w *wrapMyServiceMyRPCClient) Recv() (*MyResp, error)
				//
				g.P("func (w *", wrapRPCClientName, ") Recv() (", tc_resp.TypeName(g, TNT_TYPENAME), ", error) {")
				g.In()
				g.P("resp, err := w.cli.Recv()")
			} else {
				//
				// func (w *wrapMyServiceMyRPCClient) CloseAndRecv() (*MyResp, error)
				//
				g.P("func (w *", wrapRPCClientName, ") CloseAndRecv() (", tc_resp.TypeName(g, TNT_TYPENAME), ", error) {")
				g.In()
				g.P("resp, err := w.cli.CloseAndRecv()")
			}

			// get and convert response
			s.generateErrorCheck(g, defretnilvalue)
			g.P()

			g.P("var wresp ", tc_resp.TypeName(g, TNT_TYPENAME))

			check_error, err = tc_resp.GenerateImport(g, "resp", "wresp", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheck(g, defretnilvalue)
			}

			g.P()

			g.P("return wresp, nil")

			g.Out()
			g.P("}")
			g.P()
		}
	}

	//
	// SERVER
	//

	//
	// type MyServiceServer interface
	//

	if !g.GenerateComment(svc.Comment) {
		g.P()
		g.P("// Server API for ", svcName, " service")
		g.P()
	}

	g.P("type ", svcName, "Server interface {")
	g.In()

	for _, rpc := range svc.RPCs {
		tc_req, err := g.GetGowrapType("", rpc.RequestType)
		if err != nil {
			return err
		}
		tc_resp, err := g.GetGowrapType("", rpc.ResponseType)
		if err != nil {
			return err
		}

		g.GenerateComment(rpc.Comment)

		if !rpc.StreamsRequest && rpc.StreamsResponse {
			//
			// MyRPC(*MyReq, MyService_MyRRPCServer) error
			//
			g.P(rpc.Name, "(", tc_req.TypeName(g, TNT_TYPENAME), ", ", svcName, "_", rpc.Name, "Server) error")
		} else if rpc.StreamsRequest || rpc.StreamsResponse {
			//
			// MyRPC(MyService_MyRRPCServer) error
			//
			g.P(rpc.Name, "(", svcName, "_", rpc.Name, "Server) error")
		} else {
			//
			// MyRPC(ctx.Context, *MyReq) (*MyResp, error)
			//
			g.P(rpc.Name, "(", ctx_alias, ".Context, ", tc_req.TypeName(g, TNT_TYPENAME), ") (", tc_resp.TypeName(g, TNT_TYPENAME), ", error)")
		}
	}

	g.Out()
	g.P("}")
	g.P()

	//
	// type wrapMyServiceServer struct
	//

	wrapServerName := "wrap" + svcName + "Server"

	g.P("type ", wrapServerName, " struct {")
	g.In()

	g.P("srv ", svcName, "Server")
	g.P("opts ", util_alias, ".RegServerOptions")
	g.Out()
	g.P("}")
	g.P()

	//
	// func newWrapMyServiceServer(srv MyServiceServer, opts ...fproto_gowrap_util.RegServerOption) *wrapMyServiceServer
	//

	g.P("func newWrap", svcName, "Server(srv ", svcName, "Server, opts ...", util_alias, ".RegServerOption) *wrap", svcName, "Server {")
	g.In()

	g.P("w := &", wrapServerName, "{srv: srv}")
	g.P("for _, o := range opts {")
	g.In()
	g.P("o(&w.opts)")
	g.Out()
	g.P("}")

	g.P("return w")

	g.Out()
	g.P("}")
	g.P()

	var errVar string
	if s.WrapErrors {
		//
		// func (w *wrapMyServiceServer) wrapError(ServerErrorType, error) error
		//
		g.P("func (w *", wrapServerName, ") wrapError(errorType ", util_alias, ".ServerErrorType, err error) error {")
		g.In()

		g.P("if w.opts.ErrorWrapper != nil {")
		g.In()
		g.P("return w.opts.ErrorWrapper.WrapError(errorType, err)")
		g.Out()
		g.P("} else {")
		g.In()
		g.P("return err")
		g.Out()
		g.P("}")

		g.Out()
		g.P("}")
		g.P()

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
			g.P("func (w *", wrapServerName, ") ", rpc.Name, "(req ", tcgo_req.TypeName(g, TNT_TYPENAME), ", stream ", func_alias, ".", svcName, "_", rpc.Name, "Server) error {")
		} else if rpc.StreamsRequest || rpc.StreamsResponse {
			//
			// func (w *wrapMyServiceServer) MyRPC(stream myapp.MyService_MyRRPCServer) error
			//
			g.P("func (w *", wrapServerName, ") ", rpc.Name, "(stream ", func_alias, ".", svcName, "_", rpc.Name, "Server) error {")
		} else {
			//
			// func (w *wrapMyServiceServer) MyRPC(ctx context.Context, req *myapp.MyReq) (*myapp.MyResp, error)
			//
			g.P("func (w *", wrapServerName, ") ", rpc.Name, "(ctx ", ctx_alias, ".Context, req ", tcgo_req.TypeName(g, TNT_TYPENAME), ") (", tcgo_resp.TypeName(g, TNT_TYPENAME), ", error) {")
		}

		g.In()
		g.P("var err error")

		g.P()

		// default return value
		defretvalue := tcgo_resp.TypeName(g, TNT_EMPTYVALUE)
		send_defretvalue := defretvalue
		if rpc.StreamsRequest || rpc.StreamsResponse {
			defretvalue = ""
		}

		// default return value
		defretnilvalue := tcgo_resp.TypeName(g, TNT_EMPTYORNILVALUE)

		if !rpc.StreamsRequest {
			// convert request
			g.P("var wreq ", tc_req.TypeName(g, TNT_TYPENAME))

			check_error, err := tc_req.GenerateImport(g, "req", "wreq", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheckCustomError(g, defretvalue, strings.Replace(errVar, "%%ERROR_TYPE%%", "SET_IMPORT", -1))
			}

			g.P()
		}

		// call
		if !rpc.StreamsRequest && rpc.StreamsResponse {
			g.P("err = w.srv.", rpc.Name, "(wreq, &wrap", svcName, rpc.Name, "Server{srv: stream})")
		} else if rpc.StreamsRequest || rpc.StreamsResponse {
			g.P("err = w.srv.", rpc.Name, "(&wrap", svcName, rpc.Name, "Server{srv: stream})")
		} else {
			g.P("resp, err := w.srv.", rpc.Name, "(ctx, wreq)")
		}

		s.generateErrorCheckCustomError(g, defretvalue, strings.Replace(errVar, "%%ERROR_TYPE%%", "SET_CALL", -1))
		g.P()

		// convert response
		if !rpc.StreamsRequest && !rpc.StreamsResponse {
			// Allows returning nil from server
			if tc_resp.IsPointer() && defretvalue != "" {
				g.P("if resp == nil {")
				g.In()
				g.P("return ", defretvalue, ", nil")
				g.Out()
				g.P("}")
			}

			g.P("var wresp ", tcgo_resp.TypeName(g, TNT_TYPENAME))

			check_error, err := tc_resp.GenerateExport(g, "resp", "wresp", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheckCustomError(g, defretvalue, strings.Replace(errVar, "%%ERROR_TYPE%%", "SET_EXPORT", -1))
			}

			g.P()
		}

		// return response
		if defretvalue != "" {
			g.P("return wresp, nil")
		} else {
			g.P("return nil")
		}

		g.Out()
		g.P("}")
		g.P()

		// Generate stream request / response
		if rpc.StreamsRequest || rpc.StreamsResponse {
			//
			// type MyService_MyRPCServer interface
			//

			g.P("type ", svcName, "_", rpc.Name, "Server interface {")
			g.In()

			if rpc.StreamsRequest {
				//
				// Recv() (*MyReq, error)
				//
				g.P("Recv() (", tc_req.TypeName(g, TNT_TYPENAME), ", error)")
			}
			if rpc.StreamsResponse {
				//
				// Send(*MyResp) error
				//
				g.P("Send(", tc_resp.TypeName(g, TNT_TYPENAME), ") error")
			}
			if rpc.StreamsRequest && !rpc.StreamsResponse {
				//
				// SendAndClose(*MyResp) error
				//
				g.P("SendAndClose(", tc_resp.TypeName(g, TNT_TYPENAME), ") error")
			}

			g.Out()
			g.P("}")

			g.P()

			//
			// type wrapMyServiceMyRPCServer struct
			//
			wrapRPCServerName := "wrap" + svcName + rpc.Name + "Server"

			g.P("type ", wrapRPCServerName, " struct {")
			g.In()

			g.P("srv ", func_alias, ".", svcName, "_", rpc.Name, "Server")

			g.Out()
			g.P("}")
			g.P()

			if rpc.StreamsRequest {
				//
				// func (w *wrapMyServiceMyRPCServer) Send(*MyReq) error
				//
				g.P("func (w *", wrapRPCServerName, ") Recv() (", tc_req.TypeName(g, TNT_TYPENAME), ", error) {")
				g.In()

				g.P("var err error")

				g.P("req, err := w.srv.Recv()")
				s.generateErrorCheck(g, defretnilvalue)
				g.P()

				// convert request
				g.P("var wreq ", tc_req.TypeName(g, TNT_TYPENAME))

				check_error, err := tc_req.GenerateImport(g, "req", "wreq", "err")
				if err != nil {
					return err
				}
				if check_error {
					s.generateErrorCheck(g, defretnilvalue)
				}

				g.P()
				g.P("return wreq, nil")
				g.Out()
				g.P("}")
				g.P()
			}

			// Always have Send or SendAndClose
			if rpc.StreamsResponse {
				//
				// func (w *wrapMyServiceMyRPCServer) Send(*MyResp)  error
				//
				g.P("func (w *", wrapRPCServerName, ") Send(resp ", tc_resp.TypeName(g, TNT_TYPENAME), ") error {")
			} else {
				//
				// func (w *wrapMyServiceMyRPCServer) CloseAndRecv() (*MyResp, error)
				//
				g.P("func (w *", wrapRPCServerName, ") SendAndClose(resp ", tc_resp.TypeName(g, TNT_TYPENAME), ") error {")
			}

			g.In()
			g.P("var err error")
			g.P("var wresp ", tcgo_resp.TypeName(g, TNT_TYPENAME))

			// Allows returning nil from server
			if tc_resp.IsPointer() && send_defretvalue != "" {
				g.P("if resp == nil {")
				g.In()
				g.P("wresp = ", send_defretvalue)
				g.Out()
				g.P("} else {")
				g.In()
			}

			check_error, err := tc_resp.GenerateExport(g, "resp", "wresp", "err")
			if err != nil {
				return err
			}
			if check_error {
				s.generateErrorCheck(g, "")
			}

			// Allows returning nil from server
			if tc_resp.IsPointer() && send_defretvalue != "" {
				g.Out()
				g.P("}")
			}

			g.P()

			if rpc.StreamsResponse {
				g.P("err = w.srv.Send(wresp)")
			} else {
				g.P("err = w.srv.SendAndClose(wresp)")
			}

			// get and convert response
			s.generateErrorCheck(g, "")
			g.P()
			g.P("return nil")
			g.Out()
			g.P("}")
			g.P()
		}
	}

	//
	// func RegisterMyServiceServer(s *grpc.Server, srv MyServiceServer)
	//

	g.P("func Register", svcName, "Server(s *", grpc_alias, ".Server, srv ", svcName, "Server, opts ...", util_alias, ".RegServerOption) {")
	g.In()

	// myapp.RegisterMyServiceServer(s, newWrapMyServiceServer(srv))
	g.P(func_alias, ".Register", svcName, "Server(s, newWrap", svcName, "Server(srv, opts...))")

	g.Out()
	g.P("}")

	g.P()

	return nil
}

func (s *ServiceGen_gRPC) generateErrorCheck(g *Generator, extraRetVal string) {
	g.P("if err != nil {")
	g.In()
	if extraRetVal != "" {
		g.P("return ", extraRetVal, ", err")
	} else {
		g.P("return err")
	}
	g.Out()
	g.P("}")
}

func (s *ServiceGen_gRPC) generateErrorCheckCustomError(g *Generator, extraRetVal string, err string) {
	g.P("if err != nil {")
	g.In()
	if extraRetVal != "" {
		g.P("return ", extraRetVal, ", ", err)
	} else {
		g.P("return ", err)
	}
	g.Out()
	g.P("}")
}
