// Code generated by protoc-gen-go. DO NOT EDIT.
// source: bookstore_v2.proto

package endpoints_examples_bookstore_v2

import (
	context "context"
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

// A shelf resource.
type Shelf struct {
	// A unique shelf id.
	Id int64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	// A theme of the shelf (fiction, poetry, etc).
	Theme                string   `protobuf:"bytes,2,opt,name=theme,proto3" json:"theme,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Shelf) Reset()         { *m = Shelf{} }
func (m *Shelf) String() string { return proto.CompactTextString(m) }
func (*Shelf) ProtoMessage()    {}
func (*Shelf) Descriptor() ([]byte, []int) {
	return fileDescriptor_a6a8713a8b188f49, []int{0}
}

func (m *Shelf) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Shelf.Unmarshal(m, b)
}
func (m *Shelf) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Shelf.Marshal(b, m, deterministic)
}
func (m *Shelf) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Shelf.Merge(m, src)
}
func (m *Shelf) XXX_Size() int {
	return xxx_messageInfo_Shelf.Size(m)
}
func (m *Shelf) XXX_DiscardUnknown() {
	xxx_messageInfo_Shelf.DiscardUnknown(m)
}

var xxx_messageInfo_Shelf proto.InternalMessageInfo

func (m *Shelf) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Shelf) GetTheme() string {
	if m != nil {
		return m.Theme
	}
	return ""
}

// Request message for GetShelf method.
type GetShelfRequest struct {
	// The ID of the shelf resource to retrieve.
	Shelf                int64    `protobuf:"varint,1,opt,name=shelf,proto3" json:"shelf,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetShelfRequest) Reset()         { *m = GetShelfRequest{} }
func (m *GetShelfRequest) String() string { return proto.CompactTextString(m) }
func (*GetShelfRequest) ProtoMessage()    {}
func (*GetShelfRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_a6a8713a8b188f49, []int{1}
}

func (m *GetShelfRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetShelfRequest.Unmarshal(m, b)
}
func (m *GetShelfRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetShelfRequest.Marshal(b, m, deterministic)
}
func (m *GetShelfRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetShelfRequest.Merge(m, src)
}
func (m *GetShelfRequest) XXX_Size() int {
	return xxx_messageInfo_GetShelfRequest.Size(m)
}
func (m *GetShelfRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetShelfRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetShelfRequest proto.InternalMessageInfo

func (m *GetShelfRequest) GetShelf() int64 {
	if m != nil {
		return m.Shelf
	}
	return 0
}

func init() {
	proto.RegisterType((*Shelf)(nil), "endpoints.examples.bookstore.v2.Shelf")
	proto.RegisterType((*GetShelfRequest)(nil), "endpoints.examples.bookstore.v2.GetShelfRequest")
}

func init() {
	proto.RegisterFile("bookstore_v2.proto", fileDescriptor_a6a8713a8b188f49)
}

var fileDescriptor_a6a8713a8b188f49 = []byte{
	// 242 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x4a, 0xca, 0xcf, 0xcf,
	0x2e, 0x2e, 0xc9, 0x2f, 0x4a, 0x8d, 0x2f, 0x33, 0xd2, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x92,
	0x4f, 0xcd, 0x4b, 0x29, 0xc8, 0xcf, 0xcc, 0x2b, 0x29, 0xd6, 0x4b, 0xad, 0x48, 0xcc, 0x2d, 0xc8,
	0x49, 0x2d, 0xd6, 0x83, 0x2b, 0xd3, 0x2b, 0x33, 0x92, 0x92, 0x49, 0xcf, 0xcf, 0x4f, 0xcf, 0x49,
	0xd5, 0x4f, 0x2c, 0xc8, 0xd4, 0x4f, 0xcc, 0xcb, 0xcb, 0x2f, 0x49, 0x2c, 0xc9, 0xcc, 0xcf, 0x2b,
	0x86, 0x68, 0x57, 0xd2, 0xe5, 0x62, 0x0d, 0xce, 0x48, 0xcd, 0x49, 0x13, 0xe2, 0xe3, 0x62, 0xca,
	0x4c, 0x91, 0x60, 0x54, 0x60, 0xd4, 0x60, 0x0e, 0x62, 0xca, 0x4c, 0x11, 0x12, 0xe1, 0x62, 0x2d,
	0xc9, 0x48, 0xcd, 0x4d, 0x95, 0x60, 0x52, 0x60, 0xd4, 0xe0, 0x0c, 0x82, 0x70, 0x94, 0xd4, 0xb9,
	0xf8, 0xdd, 0x53, 0x4b, 0xc0, 0x3a, 0x82, 0x52, 0x0b, 0x4b, 0x53, 0x8b, 0x4b, 0x40, 0x0a, 0x8b,
	0x41, 0x7c, 0xa8, 0x5e, 0x08, 0xc7, 0xe8, 0x3f, 0x23, 0x17, 0xa7, 0x13, 0xcc, 0x19, 0x42, 0x8d,
	0x8c, 0x5c, 0x1c, 0x30, 0x7d, 0x42, 0x06, 0x7a, 0x04, 0x9c, 0xac, 0x87, 0x66, 0x85, 0x94, 0x1a,
	0x41, 0x1d, 0x60, 0xe5, 0x4a, 0xd2, 0x4d, 0x97, 0x9f, 0x4c, 0x66, 0x12, 0x15, 0x12, 0xd6, 0x2f,
	0x33, 0xd2, 0x07, 0xb9, 0xa3, 0x2c, 0xb5, 0x58, 0xbf, 0x1a, 0xec, 0xa0, 0x5a, 0xa1, 0x3c, 0x2e,
	0x01, 0x98, 0xb9, 0x8e, 0xa5, 0x25, 0xf9, 0x4e, 0x99, 0x79, 0x29, 0x34, 0x74, 0x0a, 0x43, 0x12,
	0x1b, 0x38, 0x80, 0x8d, 0x01, 0x01, 0x00, 0x00, 0xff, 0xff, 0x9c, 0xe1, 0x2d, 0x7c, 0xb5, 0x01,
	0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// BookstoreClient is the client API for Bookstore service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type BookstoreClient interface {
	// Returns a list of all shelves in the bookstore.
	GetShelf(ctx context.Context, in *GetShelfRequest, opts ...grpc.CallOption) (*Shelf, error)
	// Returns a list of all shelves in the bookstore.
	// Specifically not using google.api.http option
	// to test grpc_transcoding auto_binding feature.
	// HTTP client can call this method with:
	// POST /endpoints.examples.bookstore.v2.Bookstore/GetShelfAutoBind
	GetShelfAutoBind(ctx context.Context, in *GetShelfRequest, opts ...grpc.CallOption) (*Shelf, error)
}

type bookstoreClient struct {
	cc grpc.ClientConnInterface
}

func NewBookstoreClient(cc grpc.ClientConnInterface) BookstoreClient {
	return &bookstoreClient{cc}
}

func (c *bookstoreClient) GetShelf(ctx context.Context, in *GetShelfRequest, opts ...grpc.CallOption) (*Shelf, error) {
	out := new(Shelf)
	err := c.cc.Invoke(ctx, "/endpoints.examples.bookstore.v2.Bookstore/GetShelf", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bookstoreClient) GetShelfAutoBind(ctx context.Context, in *GetShelfRequest, opts ...grpc.CallOption) (*Shelf, error) {
	out := new(Shelf)
	err := c.cc.Invoke(ctx, "/endpoints.examples.bookstore.v2.Bookstore/GetShelfAutoBind", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BookstoreServer is the server API for Bookstore service.
type BookstoreServer interface {
	// Returns a list of all shelves in the bookstore.
	GetShelf(context.Context, *GetShelfRequest) (*Shelf, error)
	// Returns a list of all shelves in the bookstore.
	// Specifically not using google.api.http option
	// to test grpc_transcoding auto_binding feature.
	// HTTP client can call this method with:
	// POST /endpoints.examples.bookstore.v2.Bookstore/GetShelfAutoBind
	GetShelfAutoBind(context.Context, *GetShelfRequest) (*Shelf, error)
}

// UnimplementedBookstoreServer can be embedded to have forward compatible implementations.
type UnimplementedBookstoreServer struct {
}

func (*UnimplementedBookstoreServer) GetShelf(ctx context.Context, req *GetShelfRequest) (*Shelf, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetShelf not implemented")
}
func (*UnimplementedBookstoreServer) GetShelfAutoBind(ctx context.Context, req *GetShelfRequest) (*Shelf, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetShelfAutoBind not implemented")
}

func RegisterBookstoreServer(s *grpc.Server, srv BookstoreServer) {
	s.RegisterService(&_Bookstore_serviceDesc, srv)
}

func _Bookstore_GetShelf_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetShelfRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BookstoreServer).GetShelf(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/endpoints.examples.bookstore.v2.Bookstore/GetShelf",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BookstoreServer).GetShelf(ctx, req.(*GetShelfRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Bookstore_GetShelfAutoBind_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetShelfRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BookstoreServer).GetShelfAutoBind(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/endpoints.examples.bookstore.v2.Bookstore/GetShelfAutoBind",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BookstoreServer).GetShelfAutoBind(ctx, req.(*GetShelfRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Bookstore_serviceDesc = grpc.ServiceDesc{
	ServiceName: "endpoints.examples.bookstore.v2.Bookstore",
	HandlerType: (*BookstoreServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetShelf",
			Handler:    _Bookstore_GetShelf_Handler,
		},
		{
			MethodName: "GetShelfAutoBind",
			Handler:    _Bookstore_GetShelfAutoBind_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "bookstore_v2.proto",
}
