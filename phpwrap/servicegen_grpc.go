package fproto_phpwrap

import (
	"errors"
	"path"

	"github.com/RangelReale/fdep"
	"github.com/RangelReale/fproto"
	"github.com/RangelReale/fproto-wrap"
)

// Generates service specifications for gRPC
type ServiceGen_gRPC struct {
}

func NewServiceGen_gRPC() *ServiceGen_gRPC {
	return &ServiceGen_gRPC{}
}

func (s *ServiceGen_gRPC) ServiceType() string {
	return "grpc"
}

func (s *ServiceGen_gRPC) BuildServiceClientName(g *Generator, svc *fproto.ServiceElement) (phpName string, protoName string) {
	// get the dep type
	tp_svc := g.dep.DepTypeFromElement(svc)
	if tp_svc == nil {
		panic("service type not found")
	}

	// Camel-cased name, with "." replaced by "_"
	phpName = fproto_wrap.CamelCaseProto(tp_svc.Name + "Client")

	protoName = tp_svc.Name

	return
}

func (s *ServiceGen_gRPC) BuildRpcName(rpc *fproto.RPCElement) (phpName string) {
	phpName = fproto_wrap.CamelCase(rpc.Name)
	return
}

func (s *ServiceGen_gRPC) GenerateService(g *Generator, svc *fproto.ServiceElement) error {
	tp_svc := g.dep.DepTypeFromElement(svc)
	if tp_svc == nil {
		return errors.New("service type not found")
	}

	sourceNS, _, wrapPath := g.PhpWrapNS(g.GetFileDep())
	svcPhpName, svcProtoName := s.BuildServiceClientName(g, svc)
	fileId := path.Join(wrapPath, svcPhpName)

	g.SetFile(fileId)

	gf := g.F(fileId)

	// class Message

	if !gf.GenerateComment(svc.Comment, nil) {
		gf.GenerateCommentLine("SERVICE: ", svcProtoName)
	}

	gf.P("class ", svcPhpName)
	gf.P("{")
	gf.In()

	gf.P("/**")
	gf.P(" * @var \\", sourceNS, "\\", svcPhpName)
	gf.P(" */")
	gf.P("public $client = null;")
	gf.P()

	// constructor
	gf.P("public function __construct($hostname, $opts, $channel = null) {")
	gf.In()
	gf.P("$this->client = new \\", sourceNS, "\\", svcPhpName, "($hostname, $opts, $channel);")
	gf.Out()
	gf.P("}")

	// RPCs
	for _, rpc := range svc.RPCs {
		tp_req, err := tp_svc.GetType(rpc.RequestType)
		if err != nil {
			return err
		}

		tp_resp, err := tp_svc.GetType(rpc.ResponseType)
		if err != nil {
			return err
		}

		_, wrapReqFieldTypeName := g.BuildTypeNSName(tp_req)
		_, wrapRespFieldTypeName := g.BuildTypeNSName(tp_resp)

		rpcName := s.BuildRpcName(rpc)

		gf.P()

		if !rpc.StreamsRequest {
			gf.P("public function ", rpcName, "(", wrapReqFieldTypeName, " $argument,")
			gf.In()
			gf.P("$metadata = [], $options = []) {")

		} else {
			gf.P("public function ", rpcName, "($metadata = [], $options = []) {")
			gf.In()
		}

		if rpc.StreamsRequest || rpc.StreamsResponse {
			isRequestObject := "false"
			if !tp_req.IsScalar() && tp_req.IsPointer() && tp_req.FileDep.DepType == fdep.DepType_Own {
				isRequestObject = "true"
			}

			if !tp_resp.IsScalar() && tp_resp.IsPointer() && tp_resp.FileDep.DepType == fdep.DepType_Own {
				gf.P("$resp_obj = new ", wrapRespFieldTypeName, "();")
			} else {
				gf.P("$resp_obj = null;")
			}

			if rpc.StreamsRequest && rpc.StreamsResponse {
				gf.P("return new \\RangelReale\\FPWrap\\BidiStreamingCall($this->client->", rpcName, "($metadata, $options), ", isRequestObject, ", $resp_obj);")
			} else if rpc.StreamsRequest {
				gf.P("return new \\RangelReale\\FPWrap\\ClientStreamingCall($this->client->", rpcName, "($metadata, $options), ", isRequestObject, ", $resp_obj);")
			} else {
				if !tp_req.IsScalar() && tp_req.IsPointer() && tp_req.FileDep.DepType == fdep.DepType_Own {
					gf.P("$rreq = $argument->export();")
				} else {
					gf.P("$rreq = $argument;")
				}
				gf.P("return new \\RangelReale\\FPWrap\\ServerStreamingCall($this->client->", rpcName, "($rreq, $metadata, $options), $resp_obj);")
			}
		} else {
			if !tp_req.IsScalar() && tp_req.IsPointer() && tp_req.FileDep.DepType == fdep.DepType_Own {
				gf.P("$rreq = $argument->export();")
			} else {
				gf.P("$rreq = $argument;")
			}

			gf.P("$resp_call = $this->client->", fproto_wrap.CamelCase(rpc.Name), "($rreq, $metadata, $options);")

			if !tp_resp.IsScalar() && tp_resp.IsPointer() && tp_resp.FileDep.DepType == fdep.DepType_Own {
				gf.P("$resp_obj = new ", wrapRespFieldTypeName, "();")
			} else {
				gf.P("$resp_obj = null;")
			}

			gf.P("return new \\RangelReale\\FPWrap\\UnaryCall($resp_call, $resp_obj);")
		}

		gf.Out()
		gf.P("}")
	}

	// finish class
	gf.Out()
	gf.P("}")

	return nil

}
