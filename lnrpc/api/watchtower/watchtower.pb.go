// Code generated by protoc-gen-go. DO NOT EDIT.
// source: watchtower/watchtower.proto

package watchtower

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
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

type GetInfoRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetInfoRequest) Reset()         { *m = GetInfoRequest{} }
func (m *GetInfoRequest) String() string { return proto.CompactTextString(m) }
func (*GetInfoRequest) ProtoMessage()    {}
func (*GetInfoRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2c65982db7c0adcb, []int{0}
}

func (m *GetInfoRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetInfoRequest.Unmarshal(m, b)
}
func (m *GetInfoRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetInfoRequest.Marshal(b, m, deterministic)
}
func (m *GetInfoRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetInfoRequest.Merge(m, src)
}
func (m *GetInfoRequest) XXX_Size() int {
	return xxx_messageInfo_GetInfoRequest.Size(m)
}
func (m *GetInfoRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetInfoRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetInfoRequest proto.InternalMessageInfo

type GetInfoResponse struct {
	/// The public key of the watchtower.
	Pubkey []byte `protobuf:"bytes,1,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
	/// The listening addresses of the watchtower.
	Listeners []string `protobuf:"bytes,2,rep,name=listeners,proto3" json:"listeners,omitempty"`
	/// The URIs of the watchtower.
	Uris                 []string `protobuf:"bytes,3,rep,name=uris,proto3" json:"uris,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetInfoResponse) Reset()         { *m = GetInfoResponse{} }
func (m *GetInfoResponse) String() string { return proto.CompactTextString(m) }
func (*GetInfoResponse) ProtoMessage()    {}
func (*GetInfoResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_2c65982db7c0adcb, []int{1}
}

func (m *GetInfoResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetInfoResponse.Unmarshal(m, b)
}
func (m *GetInfoResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetInfoResponse.Marshal(b, m, deterministic)
}
func (m *GetInfoResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetInfoResponse.Merge(m, src)
}
func (m *GetInfoResponse) XXX_Size() int {
	return xxx_messageInfo_GetInfoResponse.Size(m)
}
func (m *GetInfoResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetInfoResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetInfoResponse proto.InternalMessageInfo

func (m *GetInfoResponse) GetPubkey() []byte {
	if m != nil {
		return m.Pubkey
	}
	return nil
}

func (m *GetInfoResponse) GetListeners() []string {
	if m != nil {
		return m.Listeners
	}
	return nil
}

func (m *GetInfoResponse) GetUris() []string {
	if m != nil {
		return m.Uris
	}
	return nil
}

func init() {
	proto.RegisterType((*GetInfoRequest)(nil), "watchtower.GetInfoRequest")
	proto.RegisterType((*GetInfoResponse)(nil), "watchtower.GetInfoResponse")
}

func init() { proto.RegisterFile("watchtower/watchtower.proto", fileDescriptor_2c65982db7c0adcb) }

var fileDescriptor_2c65982db7c0adcb = []byte{
	// 213 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x90, 0x41, 0x4b, 0x85, 0x40,
	0x14, 0x85, 0x79, 0xbd, 0x78, 0xf1, 0x2e, 0x51, 0x31, 0x8b, 0x10, 0x6d, 0x21, 0xae, 0x5c, 0x39,
	0x50, 0xd1, 0x0f, 0x70, 0x13, 0xed, 0xc2, 0x4d, 0x50, 0x2b, 0xb5, 0x9b, 0x0e, 0x4e, 0x77, 0xa6,
	0x99, 0x3b, 0x48, 0xff, 0x3e, 0x30, 0x51, 0x83, 0x76, 0xf7, 0x7e, 0x67, 0x71, 0x3e, 0x0e, 0x24,
	0x63, 0xcd, 0x6d, 0xcf, 0x66, 0x44, 0x27, 0xd7, 0xb3, 0xb0, 0xce, 0xb0, 0x11, 0xb0, 0x92, 0xec,
	0x0a, 0x2e, 0x1e, 0x91, 0x9f, 0xe8, 0xc3, 0x54, 0xf8, 0x15, 0xd0, 0x73, 0xf6, 0x06, 0x97, 0x0b,
	0xf1, 0xd6, 0x90, 0x47, 0x71, 0x0d, 0x07, 0x1b, 0x9a, 0x01, 0xbf, 0xa3, 0x5d, 0xba, 0xcb, 0xcf,
	0xab, 0xf9, 0x13, 0x37, 0x70, 0xd4, 0xca, 0x33, 0x12, 0x3a, 0x1f, 0x9d, 0xa4, 0xfb, 0xfc, 0x58,
	0xad, 0x40, 0x08, 0x38, 0x0d, 0x4e, 0xf9, 0x68, 0x3f, 0x05, 0xd3, 0x7d, 0xfb, 0x0c, 0xf0, 0xb2,
	0x94, 0x8b, 0x12, 0xce, 0xe6, 0x2a, 0x11, 0x17, 0x1b, 0xcd, 0xbf, 0x46, 0x71, 0xf2, 0x6f, 0xf6,
	0xeb, 0x56, 0x3e, 0xbc, 0xde, 0x77, 0x8a, 0xfb, 0xd0, 0x14, 0xad, 0xf9, 0x94, 0x5a, 0x75, 0x3d,
	0x93, 0xa2, 0x8e, 0x90, 0x47, 0xe3, 0x06, 0xa9, 0xe9, 0x5d, 0x6a, 0x72, 0xb6, 0x95, 0xb5, 0x55,
	0x9b, 0x29, 0x9a, 0xc3, 0xb4, 0xc5, 0xdd, 0x4f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x61, 0x60, 0x24,
	0xab, 0x2a, 0x01, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// WatchtowerClient is the client API for Watchtower service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type WatchtowerClient interface {
	//* lncli: tower info
	//GetInfo returns general information concerning the companion watchtower
	//including it's public key and URIs where the server is currently
	//listening for clients.
	GetInfo(ctx context.Context, in *GetInfoRequest, opts ...grpc.CallOption) (*GetInfoResponse, error)
}

type watchtowerClient struct {
	cc grpc.ClientConnInterface
}

func NewWatchtowerClient(cc grpc.ClientConnInterface) WatchtowerClient {
	return &watchtowerClient{cc}
}

func (c *watchtowerClient) GetInfo(ctx context.Context, in *GetInfoRequest, opts ...grpc.CallOption) (*GetInfoResponse, error) {
	out := new(GetInfoResponse)
	err := c.cc.Invoke(ctx, "/watchtower.Watchtower/GetInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WatchtowerServer is the server API for Watchtower service.
type WatchtowerServer interface {
	//* lncli: tower info
	//GetInfo returns general information concerning the companion watchtower
	//including it's public key and URIs where the server is currently
	//listening for clients.
	GetInfo(context.Context, *GetInfoRequest) (*GetInfoResponse, error)
}

// UnimplementedWatchtowerServer can be embedded to have forward compatible implementations.
type UnimplementedWatchtowerServer struct {
}

func (*UnimplementedWatchtowerServer) GetInfo(ctx context.Context, req *GetInfoRequest) (*GetInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetInfo not implemented")
}

func RegisterWatchtowerServer(s *grpc.Server, srv WatchtowerServer) {
	s.RegisterService(&_Watchtower_serviceDesc, srv)
}

func _Watchtower_GetInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WatchtowerServer).GetInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/watchtower.Watchtower/GetInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WatchtowerServer).GetInfo(ctx, req.(*GetInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Watchtower_serviceDesc = grpc.ServiceDesc{
	ServiceName: "watchtower.Watchtower",
	HandlerType: (*WatchtowerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetInfo",
			Handler:    _Watchtower_GetInfo_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "watchtower/watchtower.proto",
}
