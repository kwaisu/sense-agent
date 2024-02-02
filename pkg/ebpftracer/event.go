package ebpftracer

import (
	"inet.af/netaddr"

	"github.com/kwaisu/sense-agent/pkg/ebpftracer/l7"
)

type EventType uint32
type EventReason uint32

const (
	EventTypeProcessStart    EventType = 1
	EventTypeProcessExit     EventType = 2
	EventTypeConnectionOpen  EventType = 3
	EventTypeConnectionClose EventType = 4
	EventTypeConnectionError EventType = 5
	EventTypeListenOpen      EventType = 6
	EventTypeListenClose     EventType = 7
	EventTypeFileOpen        EventType = 8
	EventTypeTCPRetransmit   EventType = 9
	EventTypeL7Request       EventType = 10

	EventReasonNone    EventReason = 0
	EventReasonOOMKill EventReason = 1
)

type l7Event struct {
	Fd                  uint64
	ConnectionTimestamp uint64
	Pid                 uint32
	Status              uint32
	Duration            uint64
	Protocol            uint8
	Method              uint8
	Padding             uint16
	StatementId         uint32
	PayloadSize         uint64
}
type fileEvent struct {
	Type EventType
	Pid  uint32
	Fd   uint64
}

type procEvent struct {
	Type   EventType
	Pid    uint32
	Reason uint32
}

type tcpEvent struct {
	Fd        uint64
	Timestamp uint64
	Type      EventType
	Pid       uint32
	SPort     uint16
	DPort     uint16
	SAddr     [16]byte
	DAddr     [16]byte
}

type Event struct {
	Type      EventType
	Reason    EventReason
	Pid       uint32
	SrcAddr   netaddr.IPPort
	DstAddr   netaddr.IPPort
	Fd        uint64
	Timestamp uint64
	L7Request *l7.RequestData
}

type perfEventMap struct {
	name             string
	perCPUBufferSize int
	typ              perfMapType
}

type perfMapType uint8

const (
	perfMapTypeProcEvents perfMapType = 1
	perfMapTypeTCPEvents  perfMapType = 2
	perfMapTypeFileEvents perfMapType = 3
	perfMapTypeL7Events   perfMapType = 4
)

var perfEvenMaps = []perfEventMap{
	{name: "proc_events", perCPUBufferSize: 4, typ: perfMapTypeProcEvents},
	{name: "tcp_listen_events", perCPUBufferSize: 4, typ: perfMapTypeTCPEvents},
	{name: "tcp_connect_events", perCPUBufferSize: 8, typ: perfMapTypeTCPEvents},
	{name: "tcp_retransmit_events", perCPUBufferSize: 4, typ: perfMapTypeTCPEvents},
	{name: "file_events", perCPUBufferSize: 4, typ: perfMapTypeFileEvents},
	{name: "l7_events", perCPUBufferSize: 32, typ: perfMapTypeL7Events}}
