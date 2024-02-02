package ebpftracer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"golang.org/x/mod/semver"
	"inet.af/netaddr"
	"k8s.io/klog/v2"

	"github.com/kwaisu/sense-agent/pkg/ebpftracer/l7"
	"github.com/kwaisu/sense-agent/pkg/system"
)

const MaxPayloadSize = 1024

type EBPFTracer struct {
	collection       *ebpf.Collection
	readers          map[string]*perfReader
	links            []link.Link
	uprobes          map[string]*ebpf.Program
	disableL7Tracing bool
	subscribers      map[EventType][]chan Event
	lock             sync.Mutex
}
type perfReader struct {
	*perf.Reader
	perfEventMap
}

func NewTracer(kernelVersion string) (*EBPFTracer, error) {
	trace := &EBPFTracer{
		readers:     map[string]*perfReader{},
		uprobes:     map[string]*ebpf.Program{},
		subscribers: map[EventType][]chan Event{},
	}
	if prog, err := getProgram(kernelVersion); err != nil {
		return nil, err
	} else {
		collectionSpec, err := ebpf.LoadCollectionSpecFromReader(bytes.NewReader(prog))
		if err != nil {
			return nil, fmt.Errorf("collection spec from reader error: %w", err)
		}
		collection, err := ebpf.NewCollection(collectionSpec)
		if err != nil {
			return nil, fmt.Errorf("failed to new collection error : %w", err)
		}
		trace.collection = collection
		var l link.Link
		for _, programSpec := range collectionSpec.Programs {
			program := collection.Programs[programSpec.Name]
			switch programSpec.Type {
			case ebpf.TracePoint:
				parts := strings.SplitN(programSpec.AttachTo, "/", 2)
				if l, err = link.Tracepoint(parts[0], parts[1], program, nil); err == nil {
					trace.links = append(trace.links, l)
				}
			case ebpf.Kprobe:
				if strings.HasPrefix(programSpec.SectionName, "uprobe/") {
					trace.uprobes[programSpec.Name] = program
					continue
				}
				if l, err = link.Kprobe(programSpec.AttachTo, program, nil); err == nil {
					trace.links = append(trace.links, l)
				}
			}
		}
		for _, pe := range perfEvenMaps {
			reader, err := perf.NewReader(collection.Maps[string(pe.name)], pe.perCPUBufferSize)
			if err != nil {
				reader.Close()
				return nil, fmt.Errorf("failed to new  %s perfEvent Reader ", pe.name)
			}
			trace.readers[string(pe.name)] = &perfReader{Reader: reader, perfEventMap: pe}
		}
	}

	return trace, nil
}

func (t *EBPFTracer) Close() {
	for _, p := range t.uprobes {
		_ = p.Close()
	}
	for _, l := range t.links {
		_ = l.Close()
	}
	for _, r := range t.readers {
		_ = r.Close()
	}
	t.collection.Close()
}

func getProgram(kernelVersion string) ([]byte, error) {
	if _, ok := ebpfProg[runtime.GOARCH]; !ok {
		return nil, fmt.Errorf("Unsupported  architecture: %s  ", runtime.GOARCH)
	}
	var prg []byte
	kv := "v" + system.KernelMajorMinor(kernelVersion)
	for _, p := range ebpfProg[runtime.GOARCH] {
		if semver.Compare(kv, p.v) >= 0 {
			prg = p.p
			break
		}
	}
	if len(prg) == 0 {
		return nil, fmt.Errorf("unsupported kernel version: %s", kernelVersion)
	}
	if _, err := os.Stat("/sys/kernel/debug/tracing"); err != nil {
		return nil, fmt.Errorf("kernel tracing is not available: %w", err)
	}
	return prg, nil
}

func (t *EBPFTracer) Reader(perfReader *perfReader) {
	for {
		record, err := perfReader.Read()
		if err != nil {
			klog.Info("perf reader read error :", err)
			continue
		}
		if record.LostSamples > 0 {
			klog.Errorf(" %s lost samples: %d", perfReader.name, record.LostSamples)
			continue
		}
		var event Event
		switch perfReader.typ {
		case perfMapTypeL7Events:
			v := &l7Event{}
			reader := bytes.NewBuffer(record.RawSample)
			if err := binary.Read(reader, binary.LittleEndian, v); err != nil {
				klog.Warningln("failed to read msg:", err)
				continue
			}
			payload := reader.Bytes()
			req := &l7.RequestData{
				Protocol:    l7.Protocol(v.Protocol),
				Status:      l7.Status(v.Status),
				Duration:    time.Duration(v.Duration),
				Method:      l7.Method(v.Method),
				StatementId: v.StatementId,
			}
			switch {
			case v.PayloadSize == 0:
			case v.PayloadSize > MaxPayloadSize:
				req.Payload = payload[:MaxPayloadSize]
			default:
				req.Payload = payload[:v.PayloadSize]
			}
			if strings.Index(string(req.Payload), "CUPS/2.4.1") < 0 {
				event = Event{Type: EventTypeL7Request, Pid: v.Pid, Fd: v.Fd, Timestamp: v.ConnectionTimestamp, L7Request: req}
			}
		case perfMapTypeFileEvents:
			v := &fileEvent{}
			if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, v); err != nil {
				klog.Warningln("failed to read msg:", err)
				continue
			}
			event = Event{Type: v.Type, Pid: v.Pid, Fd: v.Fd}
		case perfMapTypeProcEvents:
			v := &procEvent{}
			if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, v); err != nil {
				klog.Warningln("failed to read msg:", err)
				continue
			}
			event = Event{Type: v.Type, Reason: EventReason(v.Reason), Pid: v.Pid}
		case perfMapTypeTCPEvents:
			v := &tcpEvent{}
			if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, v); err != nil {
				klog.Warningln("failed to read msg:", err)
				continue
			}
			event = Event{
				Type:      v.Type,
				Pid:       v.Pid,
				SrcAddr:   ipPort(v.SAddr, v.SPort),
				DstAddr:   ipPort(v.DAddr, v.DPort),
				Fd:        v.Fd,
				Timestamp: v.Timestamp,
			}
		default:
			continue
		}
		if eventsChan, ok := t.subscribers[event.Type]; ok {
			for _, ch := range eventsChan {
				ch <- event
			}
		}

	}
}

func (t *EBPFTracer) SubscribeEvents(eventType EventType, ch chan Event) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	if eventsCh, ok := t.subscribers[eventType]; ok {
		t.subscribers[eventType] = append(eventsCh, ch)
	} else {
		t.subscribers[eventType] = []chan Event{ch}
	}
	return nil
}

func (t *EBPFTracer) Run() {
	for _, reader := range t.readers {
		go t.Reader(reader)
	}
}

func ipPort(ip [16]byte, port uint16) netaddr.IPPort {
	i, _ := netaddr.FromStdIP(ip[:])
	return netaddr.IPPortFrom(i, port)
}
