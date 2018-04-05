# fproto-gowrap

[![GoDoc](https://godoc.org/github.com/RangelReale/fproto-wrap/gowrap?status.svg)](https://godoc.org/github.com/RangelReale/fproto-wrap/gowrap)

Package for generating wrappers to the default Go protobuf generated structs and interfaces. 

### abstract

The Go generated protobuf source code isn't very developer-friendly. Well-known types that could use standard Go types without too much effort returns structs that are hard to use and requires a lot of manual parsing.

This package creates objects that wrap the generated Go types into easier to use ones, and converts between them automatically.

Type converters can be plugged in to support converting any type, like UUIDs, time, or your application internal types.

Customizers can be used to inject your own generated code into any file, using the same tools that the library uses for its processing.

Service types can be generated using plugins, and a gRPC one is provided.

The gRPC service wrapper creates new structs with the same name as the original ones that uses the new wrapped types, and automatically calls the original Go generated ones, autmatically converting the structs between the formats.

There is a sample wrapper generation executable at [fproto-gen-go](https://github.com/RangelReale/fproto-wrap/tree/master/gowrap/fproto-gen-go).
If you need to use type converters or customizers (which should be most of the time), is is recommended that you create your own generation executable.

### example

Given this proto file:

```protobuf
syntax = "proto3";
package gw_sample;
option go_package = "gwsample/core";

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "github.com/RangelReale/fproto-wrap/uuid.proto";

message User {
    fproto_wrap.UUID id = 1;
    string name = 2;
    string email = 3;
    google.protobuf.Timestamp dt_created = 4;

    message Address {
        enum AddressType {
            HOME = 0;
            MOBILE = 1;
            WORK = 2;
        }

        AddressType address_type = 20;
        string address = 21;
    }

    Address address = 5;
}

service UserSvc {
    rpc List(google.protobuf.Empty) returns (UserListResponse);
    rpc Get(fproto_wrap.UUID) returns (User);
    rpc Add(User) returns (fproto_wrap.UUID);
    rpc Modify(User) returns (google.protobuf.Empty);
    rpc Delete(fproto_wrap.UUID) returns (google.protobuf.Empty);
}

message UserListResponse {
    repeated User list = 1;
}
```

The standard Go proto generator generates this (file ending omitted for brevity):

```go
package core

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/golang/protobuf/ptypes/timestamp"
import google_protobuf1 "github.com/golang/protobuf/ptypes/empty"
import fproto_wrap "github.com/RangelReale/fproto-wrap-std/gowrap/gwproto"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type User_Address_AddressType int32

const (
	User_Address_HOME   User_Address_AddressType = 0
	User_Address_MOBILE User_Address_AddressType = 1
	User_Address_WORK   User_Address_AddressType = 2
)

var User_Address_AddressType_name = map[int32]string{
	0: "HOME",
	1: "MOBILE",
	2: "WORK",
}
var User_Address_AddressType_value = map[string]int32{
	"HOME":   0,
	"MOBILE": 1,
	"WORK":   2,
}

func (x User_Address_AddressType) String() string {
	return proto.EnumName(User_Address_AddressType_name, int32(x))
}
func (User_Address_AddressType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor0, []int{0, 0, 0}
}

type User struct {
	Id        *fproto_wrap.UUID          `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Name      string                     `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	Email     string                     `protobuf:"bytes,3,opt,name=email" json:"email,omitempty"`
	DtCreated *google_protobuf.Timestamp `protobuf:"bytes,4,opt,name=dt_created,json=dtCreated" json:"dt_created,omitempty"`
	Address   *User_Address              `protobuf:"bytes,5,opt,name=address" json:"address,omitempty"`
}

func (m *User) Reset()                    { *m = User{} }
func (m *User) String() string            { return proto.CompactTextString(m) }
func (*User) ProtoMessage()               {}
func (*User) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *User) GetId() *fproto_wrap.UUID {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *User) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *User) GetEmail() string {
	if m != nil {
		return m.Email
	}
	return ""
}

func (m *User) GetDtCreated() *google_protobuf.Timestamp {
	if m != nil {
		return m.DtCreated
	}
	return nil
}

func (m *User) GetAddress() *User_Address {
	if m != nil {
		return m.Address
	}
	return nil
}

type User_Address struct {
	AddressType User_Address_AddressType `protobuf:"varint,20,opt,name=address_type,json=addressType,enum=gw_sample.User_Address_AddressType" json:"address_type,omitempty"`
	Address     string                   `protobuf:"bytes,21,opt,name=address" json:"address,omitempty"`
}

func (m *User_Address) Reset()                    { *m = User_Address{} }
func (m *User_Address) String() string            { return proto.CompactTextString(m) }
func (*User_Address) ProtoMessage()               {}
func (*User_Address) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

func (m *User_Address) GetAddressType() User_Address_AddressType {
	if m != nil {
		return m.AddressType
	}
	return User_Address_HOME
}

func (m *User_Address) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

type UserListResponse struct {
	List []*User `protobuf:"bytes,1,rep,name=list" json:"list,omitempty"`
}

func (m *UserListResponse) Reset()                    { *m = UserListResponse{} }
func (m *UserListResponse) String() string            { return proto.CompactTextString(m) }
func (*UserListResponse) ProtoMessage()               {}
func (*UserListResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *UserListResponse) GetList() []*User {
	if m != nil {
		return m.List
	}
	return nil
}

func init() {
	proto.RegisterType((*User)(nil), "gw_sample.User")
	proto.RegisterType((*User_Address)(nil), "gw_sample.User.Address")
	proto.RegisterType((*UserListResponse)(nil), "gw_sample.UserListResponse")
	proto.RegisterEnum("gw_sample.User_Address_AddressType", User_Address_AddressType_name, User_Address_AddressType_value)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for UserSvc service

type UserSvcClient interface {
	List(ctx context.Context, in *google_protobuf1.Empty, opts ...grpc.CallOption) (*UserListResponse, error)
	Get(ctx context.Context, in *fproto_wrap.UUID, opts ...grpc.CallOption) (*User, error)
	Add(ctx context.Context, in *User, opts ...grpc.CallOption) (*fproto_wrap.UUID, error)
	Modify(ctx context.Context, in *User, opts ...grpc.CallOption) (*google_protobuf1.Empty, error)
	Delete(ctx context.Context, in *fproto_wrap.UUID, opts ...grpc.CallOption) (*google_protobuf1.Empty, error)
}

type userSvcClient struct {
	cc *grpc.ClientConn
}

func NewUserSvcClient(cc *grpc.ClientConn) UserSvcClient {
	return &userSvcClient{cc}
}

func (c *userSvcClient) List(ctx context.Context, in *google_protobuf1.Empty, opts ...grpc.CallOption) (*UserListResponse, error) {
	out := new(UserListResponse)
	err := grpc.Invoke(ctx, "/gw_sample.UserSvc/List", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userSvcClient) Get(ctx context.Context, in *fproto_wrap.UUID, opts ...grpc.CallOption) (*User, error) {
	out := new(User)
	err := grpc.Invoke(ctx, "/gw_sample.UserSvc/Get", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userSvcClient) Add(ctx context.Context, in *User, opts ...grpc.CallOption) (*fproto_wrap.UUID, error) {
	out := new(fproto_wrap.UUID)
	err := grpc.Invoke(ctx, "/gw_sample.UserSvc/Add", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userSvcClient) Modify(ctx context.Context, in *User, opts ...grpc.CallOption) (*google_protobuf1.Empty, error) {
	out := new(google_protobuf1.Empty)
	err := grpc.Invoke(ctx, "/gw_sample.UserSvc/Modify", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userSvcClient) Delete(ctx context.Context, in *fproto_wrap.UUID, opts ...grpc.CallOption) (*google_protobuf1.Empty, error) {
	out := new(google_protobuf1.Empty)
	err := grpc.Invoke(ctx, "/gw_sample.UserSvc/Delete", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for UserSvc service

type UserSvcServer interface {
	List(context.Context, *google_protobuf1.Empty) (*UserListResponse, error)
	Get(context.Context, *fproto_wrap.UUID) (*User, error)
	Add(context.Context, *User) (*fproto_wrap.UUID, error)
	Modify(context.Context, *User) (*google_protobuf1.Empty, error)
	Delete(context.Context, *fproto_wrap.UUID) (*google_protobuf1.Empty, error)
}

func RegisterUserSvcServer(s *grpc.Server, srv UserSvcServer) {
	s.RegisterService(&_UserSvc_serviceDesc, srv)
}
```


The generator source code:

```go
package main

import (
	"log"
	
	"github.com/RangelReale/fproto-wrap/gowrap"
	"github.com/RangelReale/fproto-wrap-std/gowrap/cz/jsontag"
	"github.com/RangelReale/fproto-wrap-std/gowrap/tc/time"
	"github.com/RangelReale/fproto-wrap-std/gowrap/tc/uuid"
	"github.com/RangelReale/fdep"
)

func main() {
	// Creates the proto file files and dependencies parser
	parsedep := fdep.NewDep()
	
	// Add the google.protobuf files
	err := parsedep.AddIncludeDir(`C:\protoc-3.5.1-win32\include`)
	if err != nil {
		log.Fatal(err)
	}
	
	// Add the fproto-wrap types
	err = parsedep.AddPathWithRoot("github.com/RangelReale/fproto-wrap-std", `C:\go\src\github.com\RangelReale\fproto-wrap-std`, fdep.DepType_Imported)
	if err != nil {
		log.Fatal(err)
	}

	// Add your application source code
	err = parsedep.AddPathWithRoot("gwsample", `c:\app\src\gwsample\proto`, fdep.DepType_Own)
	if err != nil {
		log.Fatal(err)
	}
	
	// Creates the wrapper class
	w := fproto_gowrap.NewWrapper(parsedep)
	
	// Add the UUID and time.Time type converters
	w.TypeConvs = append(w.TypeConvs,
		&fprotostd_gowrap_uuid.TypeConverterPlugin_UUID{},
		&fprotostd_gowrap_time.TypeConverterPlugin_Time{},
	)
	
	// Add the JSON tag customizer (adds JSON tags to struct fields)
	w.Customizers = append(w.Customizers,
		&fprotostd_gowrap_jsontag.Customizer_JSONTag{},
	)
	
	// Generates services in gRPC format
	w.ServiceGen = &fproto_gowrap.NewServiceGen_gRPC()

	// Generates the go wrapper files in this path
	err = w.GenerateFiles(`c:\app\src\gwsample\src`)
	if err != nil {
		log.Fatal(err)
	}
}

```

Outputs this easier-to-use Go file (sample.gwpb.go):

```go
// Code generated by fproto-gowrap. DO NOT EDIT.
// source file: gwsample/core/user.proto
package core

import (
	fproto_wrap "github.com/RangelReale/fproto-wrap-std/gowrap/gwproto"
	fproto_gowrap_util "github.com/RangelReale/fproto-wrap/gowrap/util"
	uuid "github.com/RangelReale/go.uuid"
	pb_types "github.com/golang/protobuf/ptypes"
	google_protobuf "github.com/golang/protobuf/ptypes/empty"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	core "gwsample/core"
	time "time"
)

//
// ENUM: User.Address.AddressType
//
type User_Address_AddressType = core.User_Address_AddressType

const (
	User_Address_HOME   User_Address_AddressType = core.User_Address_HOME
	User_Address_MOBILE User_Address_AddressType = core.User_Address_MOBILE
	User_Address_WORK   User_Address_AddressType = core.User_Address_WORK
)

var User_Address_AddressType_name = core.User_Address_AddressType_name
var User_Address_AddressType_value = core.User_Address_AddressType_value

//
// MESSAGE: User
//
type User struct {
	Id        uuid.UUID     `json:"id,omitempty"`
	Name      string        `json:"name,omitempty"`
	Email     string        `json:"email,omitempty"`
	DtCreated time.Time     `json:"dt_created,omitempty"`
	Address   *User_Address `json:"address,omitempty"`
}

//
// IMPORT: User
//
func User_Import(s *core.User) (*User, error) {
	if s == nil {
		return nil, nil
	}

	var err error
	ret := &User{}
	// User.id
	if s.Id != nil {
		ret.Id, err = uuid.FromString(s.Id.Value)
	}
	if err != nil {
		return &User{}, err
	}
	// User.name
	ret.Name = s.Name
	// User.email
	ret.Email = s.Email
	// User.dt_created
	if s.DtCreated != nil {
		ret.DtCreated, err = pb_types.Timestamp(s.DtCreated)
	}
	if err != nil {
		return &User{}, err
	}
	// User.address
	ret.Address, err = User_Address_Import(s.Address)
	if err != nil {
		return &User{}, err
	}
	return ret, err
}

//
// EXPORT: User
//
func (m *User) Export() (*core.User, error) {
	if m == nil {
		return nil, nil
	}

	var err error
	ret := &core.User{}
	// User.id
	ret.Id = &fproto_wrap.UUID{}
	ret.Id.Value = m.Id.String()
	// User.name
	ret.Name = m.Name
	// User.email
	ret.Email = m.Email
	// User.dt_created
	ret.DtCreated, err = pb_types.TimestampProto(m.DtCreated)
	if err != nil {
		return &core.User{}, err
	}
	// User.address
	ret.Address, err = m.Address.Export()
	if err != nil {
		return &core.User{}, err
	}
	return ret, err
}

//
// MESSAGE: User.Address
//
type User_Address struct {
	AddressType User_Address_AddressType `json:"address_type,omitempty"`
	Address     string                   `json:"address,omitempty"`
}

//
// IMPORT: User.Address
//
func User_Address_Import(s *core.User_Address) (*User_Address, error) {
	if s == nil {
		return nil, nil
	}

	var err error
	ret := &User_Address{}
	// User.Address.address_type
	ret.AddressType = s.AddressType
	// User.Address.address
	ret.Address = s.Address
	return ret, err
}

//
// EXPORT: User.Address
//
func (m *User_Address) Export() (*core.User_Address, error) {
	if m == nil {
		return nil, nil
	}

	var err error
	ret := &core.User_Address{}
	// User.Address.address_type
	ret.AddressType = m.AddressType
	// User.Address.address
	ret.Address = m.Address
	return ret, err
}

//
// MESSAGE: UserListResponse
//
type UserListResponse struct {
	List []*User `json:"list,omitempty"`
}

//
// IMPORT: UserListResponse
//
func UserListResponse_Import(s *core.UserListResponse) (*UserListResponse, error) {
	if s == nil {
		return nil, nil
	}

	var err error
	ret := &UserListResponse{}
	// UserListResponse.list
	for _, ms := range s.List {
		var msi *User
		msi, err = User_Import(ms)
		if err != nil {
			return &UserListResponse{}, err
		}
		ret.List = append(ret.List, msi)
	}
	return ret, err
}

//
// EXPORT: UserListResponse
//
func (m *UserListResponse) Export() (*core.UserListResponse, error) {
	if m == nil {
		return nil, nil
	}

	var err error
	ret := &core.UserListResponse{}
	// UserListResponse.list
	for _, ms := range m.List {
		var msi *core.User
		msi, err = ms.Export()
		if err != nil {
			return &core.UserListResponse{}, err
		}
		ret.List = append(ret.List, msi)
	}
	return ret, err
}

// Client API for UserSvc service

type UserSvcClient interface {
	List(ctx context.Context, in *google_protobuf.Empty, opts ...grpc.CallOption) (*UserListResponse, error)
	Get(ctx context.Context, in uuid.UUID, opts ...grpc.CallOption) (*User, error)
	Add(ctx context.Context, in *User, opts ...grpc.CallOption) (uuid.UUID, error)
	Modify(ctx context.Context, in *User, opts ...grpc.CallOption) (*google_protobuf.Empty, error)
	Delete(ctx context.Context, in uuid.UUID, opts ...grpc.CallOption) (*google_protobuf.Empty, error)
}

type wrapUserSvcClient struct {
	cli core.UserSvcClient
}

func NewUserSvcClient(cc *grpc.ClientConn) UserSvcClient {
	w := &wrapUserSvcClient{cli: core.NewUserSvcClient(cc)}
	return w
}

func (w *wrapUserSvcClient) List(ctx context.Context, in *google_protobuf.Empty, opts ...grpc.CallOption) (*UserListResponse, error) {
	var err error

	var wreq *google_protobuf.Empty
	wreq = in
	if wreq == nil {
		wreq = &google_protobuf.Empty{}
	}

	resp, err := w.cli.List(ctx, wreq, opts...)
	if err != nil {
		return &UserListResponse{}, err
	}

	var wresp *UserListResponse
	wresp, err = UserListResponse_Import(resp)
	if err != nil {
		return &UserListResponse{}, err
	}

	return wresp, nil
}

func (w *wrapUserSvcClient) Get(ctx context.Context, in uuid.UUID, opts ...grpc.CallOption) (*User, error) {
	var err error

	var wreq *fproto_wrap.UUID
	wreq = &fproto_wrap.UUID{}
	wreq.Value = in.String()
	if wreq == nil {
		wreq = &fproto_wrap.UUID{}
	}

	resp, err := w.cli.Get(ctx, wreq, opts...)
	if err != nil {
		return &User{}, err
	}

	var wresp *User
	wresp, err = User_Import(resp)
	if err != nil {
		return &User{}, err
	}

	return wresp, nil
}

func (w *wrapUserSvcClient) Add(ctx context.Context, in *User, opts ...grpc.CallOption) (uuid.UUID, error) {
	var err error

	var wreq *core.User
	wreq, err = in.Export()
	if err != nil {
		return uuid.UUID{}, err
	}
	if wreq == nil {
		wreq = &core.User{}
	}

	resp, err := w.cli.Add(ctx, wreq, opts...)
	if err != nil {
		return uuid.UUID{}, err
	}

	var wresp uuid.UUID
	if resp != nil {
		wresp, err = uuid.FromString(resp.Value)
	}
	if err != nil {
		return uuid.UUID{}, err
	}

	return wresp, nil
}

func (w *wrapUserSvcClient) Modify(ctx context.Context, in *User, opts ...grpc.CallOption) (*google_protobuf.Empty, error) {
	var err error

	var wreq *core.User
	wreq, err = in.Export()
	if err != nil {
		return &google_protobuf.Empty{}, err
	}
	if wreq == nil {
		wreq = &core.User{}
	}

	resp, err := w.cli.Modify(ctx, wreq, opts...)
	if err != nil {
		return &google_protobuf.Empty{}, err
	}

	var wresp *google_protobuf.Empty
	wresp = resp

	return wresp, nil
}

func (w *wrapUserSvcClient) Delete(ctx context.Context, in uuid.UUID, opts ...grpc.CallOption) (*google_protobuf.Empty, error) {
	var err error

	var wreq *fproto_wrap.UUID
	wreq = &fproto_wrap.UUID{}
	wreq.Value = in.String()
	if wreq == nil {
		wreq = &fproto_wrap.UUID{}
	}

	resp, err := w.cli.Delete(ctx, wreq, opts...)
	if err != nil {
		return &google_protobuf.Empty{}, err
	}

	var wresp *google_protobuf.Empty
	wresp = resp

	return wresp, nil
}

// Server API for UserSvc service

type UserSvcServer interface {
	List(context.Context, *google_protobuf.Empty) (*UserListResponse, error)
	Get(context.Context, uuid.UUID) (*User, error)
	Add(context.Context, *User) (uuid.UUID, error)
	Modify(context.Context, *User) (*google_protobuf.Empty, error)
	Delete(context.Context, uuid.UUID) (*google_protobuf.Empty, error)
}

type wrapUserSvcServer struct {
	srv  UserSvcServer
	opts fproto_gowrap_util.RegServerOptions
}

func newWrapUserSvcServer(srv UserSvcServer, opts ...fproto_gowrap_util.RegServerOption) *wrapUserSvcServer {
	w := &wrapUserSvcServer{srv: srv}
	for _, o := range opts {
		o(&w.opts)
	}
	return w
}

func (w *wrapUserSvcServer) wrapError(errorType fproto_gowrap_util.ServerErrorType, err error) error {
	if w.opts.ErrorWrapper != nil {
		return w.opts.ErrorWrapper.WrapError(errorType, err)
	} else {
		return err
	}
}

func (w *wrapUserSvcServer) List(ctx context.Context, req *google_protobuf.Empty) (*core.UserListResponse, error) {
	var err error

	var wreq *google_protobuf.Empty
	wreq = req

	resp, err := w.srv.List(ctx, wreq)
	if err != nil {
		return &core.UserListResponse{}, w.wrapError(fproto_gowrap_util.SET_CALL, err)
	}

	if resp == nil {
		return &core.UserListResponse{}, nil
	}
	var wresp *core.UserListResponse
	wresp, err = resp.Export()
	if err != nil {
		return &core.UserListResponse{}, w.wrapError(fproto_gowrap_util.SET_EXPORT, err)
	}

	return wresp, nil
}

func (w *wrapUserSvcServer) Get(ctx context.Context, req *fproto_wrap.UUID) (*core.User, error) {
	var err error

	var wreq uuid.UUID
	if req != nil {
		wreq, err = uuid.FromString(req.Value)
	}
	if err != nil {
		return &core.User{}, w.wrapError(fproto_gowrap_util.SET_IMPORT, err)
	}

	resp, err := w.srv.Get(ctx, wreq)
	if err != nil {
		return &core.User{}, w.wrapError(fproto_gowrap_util.SET_CALL, err)
	}

	if resp == nil {
		return &core.User{}, nil
	}
	var wresp *core.User
	wresp, err = resp.Export()
	if err != nil {
		return &core.User{}, w.wrapError(fproto_gowrap_util.SET_EXPORT, err)
	}

	return wresp, nil
}

func (w *wrapUserSvcServer) Add(ctx context.Context, req *core.User) (*fproto_wrap.UUID, error) {
	var err error

	var wreq *User
	wreq, err = User_Import(req)
	if err != nil {
		return &fproto_wrap.UUID{}, w.wrapError(fproto_gowrap_util.SET_IMPORT, err)
	}

	resp, err := w.srv.Add(ctx, wreq)
	if err != nil {
		return &fproto_wrap.UUID{}, w.wrapError(fproto_gowrap_util.SET_CALL, err)
	}

	var wresp *fproto_wrap.UUID
	wresp = &fproto_wrap.UUID{}
	wresp.Value = resp.String()

	return wresp, nil
}

func (w *wrapUserSvcServer) Modify(ctx context.Context, req *core.User) (*google_protobuf.Empty, error) {
	var err error

	var wreq *User
	wreq, err = User_Import(req)
	if err != nil {
		return &google_protobuf.Empty{}, w.wrapError(fproto_gowrap_util.SET_IMPORT, err)
	}

	resp, err := w.srv.Modify(ctx, wreq)
	if err != nil {
		return &google_protobuf.Empty{}, w.wrapError(fproto_gowrap_util.SET_CALL, err)
	}

	if resp == nil {
		return &google_protobuf.Empty{}, nil
	}
	var wresp *google_protobuf.Empty
	wresp = resp

	return wresp, nil
}

func (w *wrapUserSvcServer) Delete(ctx context.Context, req *fproto_wrap.UUID) (*google_protobuf.Empty, error) {
	var err error

	var wreq uuid.UUID
	if req != nil {
		wreq, err = uuid.FromString(req.Value)
	}
	if err != nil {
		return &google_protobuf.Empty{}, w.wrapError(fproto_gowrap_util.SET_IMPORT, err)
	}

	resp, err := w.srv.Delete(ctx, wreq)
	if err != nil {
		return &google_protobuf.Empty{}, w.wrapError(fproto_gowrap_util.SET_CALL, err)
	}

	if resp == nil {
		return &google_protobuf.Empty{}, nil
	}
	var wresp *google_protobuf.Empty
	wresp = resp

	return wresp, nil
}

func RegisterUserSvcServer(s *grpc.Server, srv UserSvcServer, opts ...fproto_gowrap_util.RegServerOption) {
	core.RegisterUserSvcServer(s, newWrapUserSvcServer(srv, opts...))
}
```

### related

 * [https://github.com/RangelReale/fproto](https://github.com/RangelReale/fproto)
    The protobuf file parser used in this package.
 * [https://github.com/RangelReale/fproto-wrap-validator](https://github.com/RangelReale/fproto-wrap-validator)
    Validator generation customizer.

### author

Rangel Reale (rangelspam@gmail.com)
