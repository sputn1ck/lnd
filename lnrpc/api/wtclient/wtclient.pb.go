// Code generated by protoc-gen-go. DO NOT EDIT.
// source: wtclient/wtclient.proto

package wtclient

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

type AddTowerRequest struct {
	// The identifying public key of the watchtower to add.
	Pubkey []byte `protobuf:"bytes,1,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
	// A network address the watchtower is reachable over.
	Address              string   `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AddTowerRequest) Reset()         { *m = AddTowerRequest{} }
func (m *AddTowerRequest) String() string { return proto.CompactTextString(m) }
func (*AddTowerRequest) ProtoMessage()    {}
func (*AddTowerRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{0}
}

func (m *AddTowerRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AddTowerRequest.Unmarshal(m, b)
}
func (m *AddTowerRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AddTowerRequest.Marshal(b, m, deterministic)
}
func (m *AddTowerRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AddTowerRequest.Merge(m, src)
}
func (m *AddTowerRequest) XXX_Size() int {
	return xxx_messageInfo_AddTowerRequest.Size(m)
}
func (m *AddTowerRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_AddTowerRequest.DiscardUnknown(m)
}

var xxx_messageInfo_AddTowerRequest proto.InternalMessageInfo

func (m *AddTowerRequest) GetPubkey() []byte {
	if m != nil {
		return m.Pubkey
	}
	return nil
}

func (m *AddTowerRequest) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

type AddTowerResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AddTowerResponse) Reset()         { *m = AddTowerResponse{} }
func (m *AddTowerResponse) String() string { return proto.CompactTextString(m) }
func (*AddTowerResponse) ProtoMessage()    {}
func (*AddTowerResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{1}
}

func (m *AddTowerResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AddTowerResponse.Unmarshal(m, b)
}
func (m *AddTowerResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AddTowerResponse.Marshal(b, m, deterministic)
}
func (m *AddTowerResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AddTowerResponse.Merge(m, src)
}
func (m *AddTowerResponse) XXX_Size() int {
	return xxx_messageInfo_AddTowerResponse.Size(m)
}
func (m *AddTowerResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_AddTowerResponse.DiscardUnknown(m)
}

var xxx_messageInfo_AddTowerResponse proto.InternalMessageInfo

type RemoveTowerRequest struct {
	// The identifying public key of the watchtower to remove.
	Pubkey []byte `protobuf:"bytes,1,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
	//
	//If set, then the record for this address will be removed, indicating that is
	//is stale. Otherwise, the watchtower will no longer be used for future
	//session negotiations and backups.
	Address              string   `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RemoveTowerRequest) Reset()         { *m = RemoveTowerRequest{} }
func (m *RemoveTowerRequest) String() string { return proto.CompactTextString(m) }
func (*RemoveTowerRequest) ProtoMessage()    {}
func (*RemoveTowerRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{2}
}

func (m *RemoveTowerRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RemoveTowerRequest.Unmarshal(m, b)
}
func (m *RemoveTowerRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RemoveTowerRequest.Marshal(b, m, deterministic)
}
func (m *RemoveTowerRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RemoveTowerRequest.Merge(m, src)
}
func (m *RemoveTowerRequest) XXX_Size() int {
	return xxx_messageInfo_RemoveTowerRequest.Size(m)
}
func (m *RemoveTowerRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_RemoveTowerRequest.DiscardUnknown(m)
}

var xxx_messageInfo_RemoveTowerRequest proto.InternalMessageInfo

func (m *RemoveTowerRequest) GetPubkey() []byte {
	if m != nil {
		return m.Pubkey
	}
	return nil
}

func (m *RemoveTowerRequest) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

type RemoveTowerResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RemoveTowerResponse) Reset()         { *m = RemoveTowerResponse{} }
func (m *RemoveTowerResponse) String() string { return proto.CompactTextString(m) }
func (*RemoveTowerResponse) ProtoMessage()    {}
func (*RemoveTowerResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{3}
}

func (m *RemoveTowerResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RemoveTowerResponse.Unmarshal(m, b)
}
func (m *RemoveTowerResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RemoveTowerResponse.Marshal(b, m, deterministic)
}
func (m *RemoveTowerResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RemoveTowerResponse.Merge(m, src)
}
func (m *RemoveTowerResponse) XXX_Size() int {
	return xxx_messageInfo_RemoveTowerResponse.Size(m)
}
func (m *RemoveTowerResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_RemoveTowerResponse.DiscardUnknown(m)
}

var xxx_messageInfo_RemoveTowerResponse proto.InternalMessageInfo

type GetTowerInfoRequest struct {
	// The identifying public key of the watchtower to retrieve information for.
	Pubkey []byte `protobuf:"bytes,1,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
	// Whether we should include sessions with the watchtower in the response.
	IncludeSessions      bool     `protobuf:"varint,2,opt,name=include_sessions,proto3" json:"include_sessions,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetTowerInfoRequest) Reset()         { *m = GetTowerInfoRequest{} }
func (m *GetTowerInfoRequest) String() string { return proto.CompactTextString(m) }
func (*GetTowerInfoRequest) ProtoMessage()    {}
func (*GetTowerInfoRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{4}
}

func (m *GetTowerInfoRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetTowerInfoRequest.Unmarshal(m, b)
}
func (m *GetTowerInfoRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetTowerInfoRequest.Marshal(b, m, deterministic)
}
func (m *GetTowerInfoRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetTowerInfoRequest.Merge(m, src)
}
func (m *GetTowerInfoRequest) XXX_Size() int {
	return xxx_messageInfo_GetTowerInfoRequest.Size(m)
}
func (m *GetTowerInfoRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetTowerInfoRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetTowerInfoRequest proto.InternalMessageInfo

func (m *GetTowerInfoRequest) GetPubkey() []byte {
	if m != nil {
		return m.Pubkey
	}
	return nil
}

func (m *GetTowerInfoRequest) GetIncludeSessions() bool {
	if m != nil {
		return m.IncludeSessions
	}
	return false
}

type TowerSession struct {
	//
	//The total number of successful backups that have been made to the
	//watchtower session.
	NumBackups uint32 `protobuf:"varint,1,opt,name=num_backups,proto3" json:"num_backups,omitempty"`
	//
	//The total number of backups in the session that are currently pending to be
	//acknowledged by the watchtower.
	NumPendingBackups uint32 `protobuf:"varint,2,opt,name=num_pending_backups,proto3" json:"num_pending_backups,omitempty"`
	// The maximum number of backups allowed by the watchtower session.
	MaxBackups uint32 `protobuf:"varint,3,opt,name=max_backups,proto3" json:"max_backups,omitempty"`
	//
	//The fee rate, in satoshis per vbyte, that will be used by the watchtower for
	//the justice transaction in the event of a channel breach.
	SweepSatPerByte      uint32   `protobuf:"varint,4,opt,name=sweep_sat_per_byte,proto3" json:"sweep_sat_per_byte,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TowerSession) Reset()         { *m = TowerSession{} }
func (m *TowerSession) String() string { return proto.CompactTextString(m) }
func (*TowerSession) ProtoMessage()    {}
func (*TowerSession) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{5}
}

func (m *TowerSession) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TowerSession.Unmarshal(m, b)
}
func (m *TowerSession) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TowerSession.Marshal(b, m, deterministic)
}
func (m *TowerSession) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TowerSession.Merge(m, src)
}
func (m *TowerSession) XXX_Size() int {
	return xxx_messageInfo_TowerSession.Size(m)
}
func (m *TowerSession) XXX_DiscardUnknown() {
	xxx_messageInfo_TowerSession.DiscardUnknown(m)
}

var xxx_messageInfo_TowerSession proto.InternalMessageInfo

func (m *TowerSession) GetNumBackups() uint32 {
	if m != nil {
		return m.NumBackups
	}
	return 0
}

func (m *TowerSession) GetNumPendingBackups() uint32 {
	if m != nil {
		return m.NumPendingBackups
	}
	return 0
}

func (m *TowerSession) GetMaxBackups() uint32 {
	if m != nil {
		return m.MaxBackups
	}
	return 0
}

func (m *TowerSession) GetSweepSatPerByte() uint32 {
	if m != nil {
		return m.SweepSatPerByte
	}
	return 0
}

type Tower struct {
	// The identifying public key of the watchtower.
	Pubkey []byte `protobuf:"bytes,1,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
	// The list of addresses the watchtower is reachable over.
	Addresses []string `protobuf:"bytes,2,rep,name=addresses,proto3" json:"addresses,omitempty"`
	// Whether the watchtower is currently a candidate for new sessions.
	ActiveSessionCandidate bool `protobuf:"varint,3,opt,name=active_session_candidate,proto3" json:"active_session_candidate,omitempty"`
	// The number of sessions that have been negotiated with the watchtower.
	NumSessions uint32 `protobuf:"varint,4,opt,name=num_sessions,proto3" json:"num_sessions,omitempty"`
	// The list of sessions that have been negotiated with the watchtower.
	Sessions             []*TowerSession `protobuf:"bytes,5,rep,name=sessions,proto3" json:"sessions,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *Tower) Reset()         { *m = Tower{} }
func (m *Tower) String() string { return proto.CompactTextString(m) }
func (*Tower) ProtoMessage()    {}
func (*Tower) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{6}
}

func (m *Tower) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Tower.Unmarshal(m, b)
}
func (m *Tower) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Tower.Marshal(b, m, deterministic)
}
func (m *Tower) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Tower.Merge(m, src)
}
func (m *Tower) XXX_Size() int {
	return xxx_messageInfo_Tower.Size(m)
}
func (m *Tower) XXX_DiscardUnknown() {
	xxx_messageInfo_Tower.DiscardUnknown(m)
}

var xxx_messageInfo_Tower proto.InternalMessageInfo

func (m *Tower) GetPubkey() []byte {
	if m != nil {
		return m.Pubkey
	}
	return nil
}

func (m *Tower) GetAddresses() []string {
	if m != nil {
		return m.Addresses
	}
	return nil
}

func (m *Tower) GetActiveSessionCandidate() bool {
	if m != nil {
		return m.ActiveSessionCandidate
	}
	return false
}

func (m *Tower) GetNumSessions() uint32 {
	if m != nil {
		return m.NumSessions
	}
	return 0
}

func (m *Tower) GetSessions() []*TowerSession {
	if m != nil {
		return m.Sessions
	}
	return nil
}

type ListTowersRequest struct {
	// Whether we should include sessions with the watchtower in the response.
	IncludeSessions      bool     `protobuf:"varint,1,opt,name=include_sessions,proto3" json:"include_sessions,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListTowersRequest) Reset()         { *m = ListTowersRequest{} }
func (m *ListTowersRequest) String() string { return proto.CompactTextString(m) }
func (*ListTowersRequest) ProtoMessage()    {}
func (*ListTowersRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{7}
}

func (m *ListTowersRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListTowersRequest.Unmarshal(m, b)
}
func (m *ListTowersRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListTowersRequest.Marshal(b, m, deterministic)
}
func (m *ListTowersRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListTowersRequest.Merge(m, src)
}
func (m *ListTowersRequest) XXX_Size() int {
	return xxx_messageInfo_ListTowersRequest.Size(m)
}
func (m *ListTowersRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListTowersRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListTowersRequest proto.InternalMessageInfo

func (m *ListTowersRequest) GetIncludeSessions() bool {
	if m != nil {
		return m.IncludeSessions
	}
	return false
}

type ListTowersResponse struct {
	// The list of watchtowers available for new backups.
	Towers               []*Tower `protobuf:"bytes,1,rep,name=towers,proto3" json:"towers,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListTowersResponse) Reset()         { *m = ListTowersResponse{} }
func (m *ListTowersResponse) String() string { return proto.CompactTextString(m) }
func (*ListTowersResponse) ProtoMessage()    {}
func (*ListTowersResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{8}
}

func (m *ListTowersResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListTowersResponse.Unmarshal(m, b)
}
func (m *ListTowersResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListTowersResponse.Marshal(b, m, deterministic)
}
func (m *ListTowersResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListTowersResponse.Merge(m, src)
}
func (m *ListTowersResponse) XXX_Size() int {
	return xxx_messageInfo_ListTowersResponse.Size(m)
}
func (m *ListTowersResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListTowersResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListTowersResponse proto.InternalMessageInfo

func (m *ListTowersResponse) GetTowers() []*Tower {
	if m != nil {
		return m.Towers
	}
	return nil
}

type StatsRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StatsRequest) Reset()         { *m = StatsRequest{} }
func (m *StatsRequest) String() string { return proto.CompactTextString(m) }
func (*StatsRequest) ProtoMessage()    {}
func (*StatsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{9}
}

func (m *StatsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StatsRequest.Unmarshal(m, b)
}
func (m *StatsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StatsRequest.Marshal(b, m, deterministic)
}
func (m *StatsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StatsRequest.Merge(m, src)
}
func (m *StatsRequest) XXX_Size() int {
	return xxx_messageInfo_StatsRequest.Size(m)
}
func (m *StatsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_StatsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_StatsRequest proto.InternalMessageInfo

type StatsResponse struct {
	//
	//The total number of backups made to all active and exhausted watchtower
	//sessions.
	NumBackups uint32 `protobuf:"varint,1,opt,name=num_backups,proto3" json:"num_backups,omitempty"`
	//
	//The total number of backups that are pending to be acknowledged by all
	//active and exhausted watchtower sessions.
	NumPendingBackups uint32 `protobuf:"varint,2,opt,name=num_pending_backups,proto3" json:"num_pending_backups,omitempty"`
	//
	//The total number of backups that all active and exhausted watchtower
	//sessions have failed to acknowledge.
	NumFailedBackups uint32 `protobuf:"varint,3,opt,name=num_failed_backups,proto3" json:"num_failed_backups,omitempty"`
	// The total number of new sessions made to watchtowers.
	NumSessionsAcquired uint32 `protobuf:"varint,4,opt,name=num_sessions_acquired,proto3" json:"num_sessions_acquired,omitempty"`
	// The total number of watchtower sessions that have been exhausted.
	NumSessionsExhausted uint32   `protobuf:"varint,5,opt,name=num_sessions_exhausted,proto3" json:"num_sessions_exhausted,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StatsResponse) Reset()         { *m = StatsResponse{} }
func (m *StatsResponse) String() string { return proto.CompactTextString(m) }
func (*StatsResponse) ProtoMessage()    {}
func (*StatsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{10}
}

func (m *StatsResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StatsResponse.Unmarshal(m, b)
}
func (m *StatsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StatsResponse.Marshal(b, m, deterministic)
}
func (m *StatsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StatsResponse.Merge(m, src)
}
func (m *StatsResponse) XXX_Size() int {
	return xxx_messageInfo_StatsResponse.Size(m)
}
func (m *StatsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_StatsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_StatsResponse proto.InternalMessageInfo

func (m *StatsResponse) GetNumBackups() uint32 {
	if m != nil {
		return m.NumBackups
	}
	return 0
}

func (m *StatsResponse) GetNumPendingBackups() uint32 {
	if m != nil {
		return m.NumPendingBackups
	}
	return 0
}

func (m *StatsResponse) GetNumFailedBackups() uint32 {
	if m != nil {
		return m.NumFailedBackups
	}
	return 0
}

func (m *StatsResponse) GetNumSessionsAcquired() uint32 {
	if m != nil {
		return m.NumSessionsAcquired
	}
	return 0
}

func (m *StatsResponse) GetNumSessionsExhausted() uint32 {
	if m != nil {
		return m.NumSessionsExhausted
	}
	return 0
}

type PolicyRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PolicyRequest) Reset()         { *m = PolicyRequest{} }
func (m *PolicyRequest) String() string { return proto.CompactTextString(m) }
func (*PolicyRequest) ProtoMessage()    {}
func (*PolicyRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{11}
}

func (m *PolicyRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PolicyRequest.Unmarshal(m, b)
}
func (m *PolicyRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PolicyRequest.Marshal(b, m, deterministic)
}
func (m *PolicyRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PolicyRequest.Merge(m, src)
}
func (m *PolicyRequest) XXX_Size() int {
	return xxx_messageInfo_PolicyRequest.Size(m)
}
func (m *PolicyRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_PolicyRequest.DiscardUnknown(m)
}

var xxx_messageInfo_PolicyRequest proto.InternalMessageInfo

type PolicyResponse struct {
	//
	//The maximum number of updates each session we negotiate with watchtowers
	//should allow.
	MaxUpdates uint32 `protobuf:"varint,1,opt,name=max_updates,proto3" json:"max_updates,omitempty"`
	//
	//The fee rate, in satoshis per vbyte, that will be used by watchtowers for
	//justice transactions in response to channel breaches.
	SweepSatPerByte      uint32   `protobuf:"varint,2,opt,name=sweep_sat_per_byte,proto3" json:"sweep_sat_per_byte,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PolicyResponse) Reset()         { *m = PolicyResponse{} }
func (m *PolicyResponse) String() string { return proto.CompactTextString(m) }
func (*PolicyResponse) ProtoMessage()    {}
func (*PolicyResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2f20ecc3ceb519a, []int{12}
}

func (m *PolicyResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PolicyResponse.Unmarshal(m, b)
}
func (m *PolicyResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PolicyResponse.Marshal(b, m, deterministic)
}
func (m *PolicyResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PolicyResponse.Merge(m, src)
}
func (m *PolicyResponse) XXX_Size() int {
	return xxx_messageInfo_PolicyResponse.Size(m)
}
func (m *PolicyResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_PolicyResponse.DiscardUnknown(m)
}

var xxx_messageInfo_PolicyResponse proto.InternalMessageInfo

func (m *PolicyResponse) GetMaxUpdates() uint32 {
	if m != nil {
		return m.MaxUpdates
	}
	return 0
}

func (m *PolicyResponse) GetSweepSatPerByte() uint32 {
	if m != nil {
		return m.SweepSatPerByte
	}
	return 0
}

func init() {
	proto.RegisterType((*AddTowerRequest)(nil), "wtclient.AddTowerRequest")
	proto.RegisterType((*AddTowerResponse)(nil), "wtclient.AddTowerResponse")
	proto.RegisterType((*RemoveTowerRequest)(nil), "wtclient.RemoveTowerRequest")
	proto.RegisterType((*RemoveTowerResponse)(nil), "wtclient.RemoveTowerResponse")
	proto.RegisterType((*GetTowerInfoRequest)(nil), "wtclient.GetTowerInfoRequest")
	proto.RegisterType((*TowerSession)(nil), "wtclient.TowerSession")
	proto.RegisterType((*Tower)(nil), "wtclient.Tower")
	proto.RegisterType((*ListTowersRequest)(nil), "wtclient.ListTowersRequest")
	proto.RegisterType((*ListTowersResponse)(nil), "wtclient.ListTowersResponse")
	proto.RegisterType((*StatsRequest)(nil), "wtclient.StatsRequest")
	proto.RegisterType((*StatsResponse)(nil), "wtclient.StatsResponse")
	proto.RegisterType((*PolicyRequest)(nil), "wtclient.PolicyRequest")
	proto.RegisterType((*PolicyResponse)(nil), "wtclient.PolicyResponse")
}

func init() { proto.RegisterFile("wtclient/wtclient.proto", fileDescriptor_a2f20ecc3ceb519a) }

var fileDescriptor_a2f20ecc3ceb519a = []byte{
	// 634 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xb4, 0x55, 0xcd, 0x6e, 0xd3, 0x4c,
	0x14, 0x95, 0x93, 0x2f, 0xf9, 0xd2, 0xdb, 0xf4, 0x87, 0x5b, 0xb5, 0x35, 0xa6, 0x48, 0x91, 0x37,
	0x44, 0x2c, 0x12, 0x14, 0x2a, 0x84, 0x40, 0x80, 0x4a, 0x25, 0x2a, 0x10, 0x0b, 0xe4, 0x22, 0x21,
	0xd8, 0x58, 0x63, 0x7b, 0x9a, 0x8c, 0xea, 0x8c, 0x5d, 0xcf, 0xb8, 0x69, 0x96, 0x3c, 0x0a, 0xef,
	0xc0, 0x8b, 0xf0, 0x46, 0xc8, 0xe3, 0xdf, 0x24, 0xb6, 0x58, 0x20, 0x76, 0x99, 0x73, 0xae, 0xcf,
	0xdc, 0xdc, 0x73, 0x7c, 0x0d, 0xc7, 0x0b, 0xe9, 0xfa, 0x8c, 0x72, 0x39, 0xce, 0x7f, 0x8c, 0xc2,
	0x28, 0x90, 0x01, 0xf6, 0xf2, 0xb3, 0x79, 0x0e, 0x7b, 0x67, 0x9e, 0xf7, 0x39, 0x58, 0xd0, 0xc8,
	0xa2, 0x37, 0x31, 0x15, 0x12, 0x8f, 0xa0, 0x1b, 0xc6, 0xce, 0x35, 0x5d, 0xea, 0xda, 0x40, 0x1b,
	0xf6, 0xad, 0xec, 0x84, 0x3a, 0xfc, 0x4f, 0x3c, 0x2f, 0xa2, 0x42, 0xe8, 0xad, 0x81, 0x36, 0xdc,
	0xb2, 0xf2, 0xa3, 0x89, 0xb0, 0x5f, 0x8a, 0x88, 0x30, 0xe0, 0x82, 0x9a, 0xef, 0x00, 0x2d, 0x3a,
	0x0f, 0x6e, 0xe9, 0x5f, 0x6a, 0x1f, 0xc2, 0xc1, 0x8a, 0x4e, 0x26, 0xff, 0x15, 0x0e, 0x2e, 0xa8,
	0x54, 0xd8, 0x7b, 0x7e, 0x15, 0xfc, 0x49, 0xff, 0x31, 0xec, 0x33, 0xee, 0xfa, 0xb1, 0x47, 0x6d,
	0x41, 0x85, 0x60, 0x01, 0x4f, 0x2f, 0xea, 0x59, 0x1b, 0xb8, 0xf9, 0x53, 0x83, 0xbe, 0x12, 0xbe,
	0x4c, 0x11, 0x1c, 0xc0, 0x36, 0x8f, 0xe7, 0xb6, 0x43, 0xdc, 0xeb, 0x38, 0x14, 0x4a, 0x79, 0xc7,
	0xaa, 0x42, 0xf8, 0x04, 0x0e, 0x92, 0x63, 0x48, 0xb9, 0xc7, 0xf8, 0xb4, 0xa8, 0x6c, 0xa9, 0xca,
	0x3a, 0x2a, 0xd1, 0x9c, 0x93, 0xbb, 0xa2, 0xb2, 0x9d, 0x6a, 0x56, 0x20, 0x1c, 0x01, 0x8a, 0x05,
	0xa5, 0xa1, 0x2d, 0x88, 0xb4, 0x43, 0x1a, 0xd9, 0xce, 0x52, 0x52, 0xfd, 0x3f, 0x55, 0x58, 0xc3,
	0x98, 0xbf, 0x34, 0xe8, 0xa8, 0xb6, 0x1b, 0x87, 0x70, 0x02, 0x5b, 0xd9, 0x54, 0x69, 0xd2, 0x5b,
	0x7b, 0xb8, 0x65, 0x95, 0x00, 0xbe, 0x00, 0x9d, 0xb8, 0x92, 0xdd, 0x16, 0x93, 0xb0, 0x5d, 0xc2,
	0x3d, 0xe6, 0x11, 0x49, 0x55, 0x7b, 0x3d, 0xab, 0x91, 0x47, 0x13, 0xfa, 0xc9, 0x9f, 0x2c, 0x46,
	0x9b, 0x76, 0xb9, 0x82, 0xe1, 0x04, 0x7a, 0x05, 0xdf, 0x19, 0xb4, 0x87, 0xdb, 0x93, 0xa3, 0x51,
	0x11, 0xcb, 0xea, 0xbc, 0xad, 0xa2, 0xce, 0x7c, 0x03, 0xf7, 0x3e, 0x32, 0x91, 0xda, 0x2c, 0x72,
	0x8f, 0xeb, 0xbc, 0xd4, 0x1a, 0xbc, 0x7c, 0x05, 0x58, 0x15, 0x48, 0xc3, 0x83, 0x8f, 0xa0, 0x2b,
	0x15, 0xa2, 0x6b, 0xaa, 0x91, 0xbd, 0xb5, 0x46, 0xac, 0x8c, 0x36, 0x77, 0xa1, 0x7f, 0x29, 0x89,
	0xcc, 0xaf, 0x36, 0xbf, 0xb7, 0x60, 0x27, 0x03, 0x32, 0xa9, 0x7f, 0x91, 0x8d, 0x11, 0x60, 0x02,
	0x5f, 0x11, 0xe6, 0x53, 0x6f, 0x2d, 0x22, 0x35, 0x0c, 0x9e, 0xc2, 0x61, 0x75, 0xd2, 0x36, 0x71,
	0x6f, 0x62, 0x16, 0x51, 0x2f, 0xb3, 0xa1, 0x9e, 0xc4, 0x67, 0x70, 0xb4, 0x42, 0xd0, 0xbb, 0x19,
	0x89, 0x85, 0xa4, 0x9e, 0xde, 0x51, 0x8f, 0x35, 0xb0, 0xe6, 0x1e, 0xec, 0x7c, 0x0a, 0x7c, 0xe6,
	0x2e, 0xf3, 0xa1, 0x38, 0xb0, 0x9b, 0x03, 0xe5, 0x50, 0x92, 0x24, 0xc7, 0x61, 0x12, 0x8e, 0x62,
	0x28, 0x15, 0xa8, 0x21, 0xdc, 0xad, 0xa6, 0x70, 0x4f, 0x7e, 0xb4, 0x61, 0xff, 0x0b, 0x91, 0xee,
	0x4c, 0x19, 0x73, 0xae, 0xbc, 0xc2, 0x33, 0xe8, 0xe5, 0x6b, 0x07, 0xef, 0x97, 0x16, 0xae, 0xed,
	0x33, 0xc3, 0xa8, 0xa3, 0xb2, 0x4e, 0x3f, 0xc0, 0x76, 0x65, 0xbb, 0xe0, 0x49, 0x59, 0xba, 0xb9,
	0xbc, 0x8c, 0x87, 0x0d, 0x6c, 0xa6, 0x75, 0x01, 0x50, 0x66, 0x0d, 0x1f, 0x94, 0xc5, 0x1b, 0x11,
	0x36, 0x4e, 0xea, 0xc9, 0x4c, 0xe8, 0x35, 0xf4, 0xab, 0xbb, 0x0d, 0x2b, 0xf7, 0xd6, 0xec, 0x3c,
	0x63, 0x3d, 0xbd, 0xf8, 0x1c, 0x3a, 0x2a, 0xa4, 0x58, 0x79, 0xc1, 0xaa, 0x31, 0x36, 0x8e, 0x37,
	0xf0, 0xec, 0xe6, 0x97, 0xd0, 0x4d, 0xad, 0xc4, 0x4a, 0xc9, 0x8a, 0xdb, 0x86, 0xbe, 0x49, 0xa4,
	0x0f, 0xbf, 0x3d, 0xfd, 0x36, 0x99, 0x32, 0x39, 0x8b, 0x9d, 0x91, 0x1b, 0xcc, 0xc7, 0x3e, 0x9b,
	0xce, 0x24, 0x67, 0x7c, 0xca, 0xa9, 0x5c, 0x04, 0xd1, 0xf5, 0xd8, 0xe7, 0xde, 0xd8, 0xe7, 0x51,
	0xe8, 0x8e, 0x49, 0xc8, 0x8a, 0x0f, 0x92, 0xd3, 0x55, 0x5f, 0xa4, 0xa7, 0xbf, 0x03, 0x00, 0x00,
	0xff, 0xff, 0x68, 0x6b, 0x32, 0x72, 0xac, 0x06, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// WatchtowerClientClient is the client API for WatchtowerClient service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type WatchtowerClientClient interface {
	//
	//AddTower adds a new watchtower reachable at the given address and
	//considers it for new sessions. If the watchtower already exists, then
	//any new addresses included will be considered when dialing it for
	//session negotiations and backups.
	AddTower(ctx context.Context, in *AddTowerRequest, opts ...grpc.CallOption) (*AddTowerResponse, error)
	//
	//RemoveTower removes a watchtower from being considered for future session
	//negotiations and from being used for any subsequent backups until it's added
	//again. If an address is provided, then this RPC only serves as a way of
	//removing the address from the watchtower instead.
	RemoveTower(ctx context.Context, in *RemoveTowerRequest, opts ...grpc.CallOption) (*RemoveTowerResponse, error)
	// ListTowers returns the list of watchtowers registered with the client.
	ListTowers(ctx context.Context, in *ListTowersRequest, opts ...grpc.CallOption) (*ListTowersResponse, error)
	// GetTowerInfo retrieves information for a registered watchtower.
	GetTowerInfo(ctx context.Context, in *GetTowerInfoRequest, opts ...grpc.CallOption) (*Tower, error)
	// Stats returns the in-memory statistics of the client since startup.
	Stats(ctx context.Context, in *StatsRequest, opts ...grpc.CallOption) (*StatsResponse, error)
	// Policy returns the active watchtower client policy configuration.
	Policy(ctx context.Context, in *PolicyRequest, opts ...grpc.CallOption) (*PolicyResponse, error)
}

type watchtowerClientClient struct {
	cc grpc.ClientConnInterface
}

func NewWatchtowerClientClient(cc grpc.ClientConnInterface) WatchtowerClientClient {
	return &watchtowerClientClient{cc}
}

func (c *watchtowerClientClient) AddTower(ctx context.Context, in *AddTowerRequest, opts ...grpc.CallOption) (*AddTowerResponse, error) {
	out := new(AddTowerResponse)
	err := c.cc.Invoke(ctx, "/wtclient.WatchtowerClient/AddTower", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *watchtowerClientClient) RemoveTower(ctx context.Context, in *RemoveTowerRequest, opts ...grpc.CallOption) (*RemoveTowerResponse, error) {
	out := new(RemoveTowerResponse)
	err := c.cc.Invoke(ctx, "/wtclient.WatchtowerClient/RemoveTower", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *watchtowerClientClient) ListTowers(ctx context.Context, in *ListTowersRequest, opts ...grpc.CallOption) (*ListTowersResponse, error) {
	out := new(ListTowersResponse)
	err := c.cc.Invoke(ctx, "/wtclient.WatchtowerClient/ListTowers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *watchtowerClientClient) GetTowerInfo(ctx context.Context, in *GetTowerInfoRequest, opts ...grpc.CallOption) (*Tower, error) {
	out := new(Tower)
	err := c.cc.Invoke(ctx, "/wtclient.WatchtowerClient/GetTowerInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *watchtowerClientClient) Stats(ctx context.Context, in *StatsRequest, opts ...grpc.CallOption) (*StatsResponse, error) {
	out := new(StatsResponse)
	err := c.cc.Invoke(ctx, "/wtclient.WatchtowerClient/Stats", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *watchtowerClientClient) Policy(ctx context.Context, in *PolicyRequest, opts ...grpc.CallOption) (*PolicyResponse, error) {
	out := new(PolicyResponse)
	err := c.cc.Invoke(ctx, "/wtclient.WatchtowerClient/Policy", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WatchtowerClientServer is the server API for WatchtowerClient service.
type WatchtowerClientServer interface {
	//
	//AddTower adds a new watchtower reachable at the given address and
	//considers it for new sessions. If the watchtower already exists, then
	//any new addresses included will be considered when dialing it for
	//session negotiations and backups.
	AddTower(context.Context, *AddTowerRequest) (*AddTowerResponse, error)
	//
	//RemoveTower removes a watchtower from being considered for future session
	//negotiations and from being used for any subsequent backups until it's added
	//again. If an address is provided, then this RPC only serves as a way of
	//removing the address from the watchtower instead.
	RemoveTower(context.Context, *RemoveTowerRequest) (*RemoveTowerResponse, error)
	// ListTowers returns the list of watchtowers registered with the client.
	ListTowers(context.Context, *ListTowersRequest) (*ListTowersResponse, error)
	// GetTowerInfo retrieves information for a registered watchtower.
	GetTowerInfo(context.Context, *GetTowerInfoRequest) (*Tower, error)
	// Stats returns the in-memory statistics of the client since startup.
	Stats(context.Context, *StatsRequest) (*StatsResponse, error)
	// Policy returns the active watchtower client policy configuration.
	Policy(context.Context, *PolicyRequest) (*PolicyResponse, error)
}

// UnimplementedWatchtowerClientServer can be embedded to have forward compatible implementations.
type UnimplementedWatchtowerClientServer struct {
}

func (*UnimplementedWatchtowerClientServer) AddTower(ctx context.Context, req *AddTowerRequest) (*AddTowerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddTower not implemented")
}
func (*UnimplementedWatchtowerClientServer) RemoveTower(ctx context.Context, req *RemoveTowerRequest) (*RemoveTowerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveTower not implemented")
}
func (*UnimplementedWatchtowerClientServer) ListTowers(ctx context.Context, req *ListTowersRequest) (*ListTowersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListTowers not implemented")
}
func (*UnimplementedWatchtowerClientServer) GetTowerInfo(ctx context.Context, req *GetTowerInfoRequest) (*Tower, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTowerInfo not implemented")
}
func (*UnimplementedWatchtowerClientServer) Stats(ctx context.Context, req *StatsRequest) (*StatsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Stats not implemented")
}
func (*UnimplementedWatchtowerClientServer) Policy(ctx context.Context, req *PolicyRequest) (*PolicyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Policy not implemented")
}

func RegisterWatchtowerClientServer(s *grpc.Server, srv WatchtowerClientServer) {
	s.RegisterService(&_WatchtowerClient_serviceDesc, srv)
}

func _WatchtowerClient_AddTower_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddTowerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WatchtowerClientServer).AddTower(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wtclient.WatchtowerClient/AddTower",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WatchtowerClientServer).AddTower(ctx, req.(*AddTowerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WatchtowerClient_RemoveTower_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveTowerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WatchtowerClientServer).RemoveTower(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wtclient.WatchtowerClient/RemoveTower",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WatchtowerClientServer).RemoveTower(ctx, req.(*RemoveTowerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WatchtowerClient_ListTowers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListTowersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WatchtowerClientServer).ListTowers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wtclient.WatchtowerClient/ListTowers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WatchtowerClientServer).ListTowers(ctx, req.(*ListTowersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WatchtowerClient_GetTowerInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTowerInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WatchtowerClientServer).GetTowerInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wtclient.WatchtowerClient/GetTowerInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WatchtowerClientServer).GetTowerInfo(ctx, req.(*GetTowerInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WatchtowerClient_Stats_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WatchtowerClientServer).Stats(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wtclient.WatchtowerClient/Stats",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WatchtowerClientServer).Stats(ctx, req.(*StatsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WatchtowerClient_Policy_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PolicyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WatchtowerClientServer).Policy(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wtclient.WatchtowerClient/Policy",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WatchtowerClientServer).Policy(ctx, req.(*PolicyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _WatchtowerClient_serviceDesc = grpc.ServiceDesc{
	ServiceName: "wtclient.WatchtowerClient",
	HandlerType: (*WatchtowerClientServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AddTower",
			Handler:    _WatchtowerClient_AddTower_Handler,
		},
		{
			MethodName: "RemoveTower",
			Handler:    _WatchtowerClient_RemoveTower_Handler,
		},
		{
			MethodName: "ListTowers",
			Handler:    _WatchtowerClient_ListTowers_Handler,
		},
		{
			MethodName: "GetTowerInfo",
			Handler:    _WatchtowerClient_GetTowerInfo_Handler,
		},
		{
			MethodName: "Stats",
			Handler:    _WatchtowerClient_Stats_Handler,
		},
		{
			MethodName: "Policy",
			Handler:    _WatchtowerClient_Policy_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "wtclient/wtclient.proto",
}
