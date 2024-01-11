package ebpftracer

import (
	"github.com/kwaisu/sense-agent/pkg/ebpftracer/l7"
	"inet.af/netaddr"
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

type perfMapType uint8

const (
	perfMapTypeProcEvents perfMapType = 1
	perfMapTypeTCPEvents  perfMapType = 2
	perfMapTypeFileEvents perfMapType = 3
	perfMapTypeL7Events   perfMapType = 4
)
