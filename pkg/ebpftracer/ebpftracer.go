package ebpftracer

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/kwaisu/sense-agent/pkg/proc"
	"golang.org/x/mod/semver"
)

type EbpfTracer struct {
	collection       *ebpf.Collection
	readers          map[string]*perf.Reader
	links            []link.Link
	uprobes          map[string]*ebpf.Program
	disableL7Tracing bool
	subscribers      map[string]chan<- Event
	lock             sync.Mutex
}
type PerfMap struct {
	name             string
	perCPUBufferSize int
}

var perfEventMap = []PerfMap{
	{name: "proc_events", perCPUBufferSize: 4},
	{name: "tcp_listen_events", perCPUBufferSize: 4},
	{name: "tcp_connect_events", perCPUBufferSize: 8},
	{name: "tcp_retransmit_events", perCPUBufferSize: 4},
	{name: "file_events", perCPUBufferSize: 4}}

func NewTracer(kernelVersion string) (*EbpfTracer, error) {
	trace := &EbpfTracer{
		readers:     map[string]*perf.Reader{},
		uprobes:     map[string]*ebpf.Program{},
		subscribers: map[string]chan<- Event{},
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
		for _, pe := range perfEventMap {
			reader, err := perf.NewReader(collection.Maps[pe.name], pe.perCPUBufferSize)
			if err != nil {
				reader.Close()
				return nil, fmt.Errorf("failed to new  %s perfEvent Reader ", pe.name)
			}
			trace.readers[pe.name] = reader
		}
	}

	return trace, nil
}
func (t *EbpfTracer) Close() {
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
	kv := "v" + proc.KernelMajorMinor(kernelVersion)
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
func (t *EbpfTracer) Reader() {

}

func (t *EbpfTracer) Subscribe(name string, ch chan<- Event) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	if _, ok := t.subscribers[name]; ok {
		return fmt.Errorf("duplicate subscriber for group %s", name)
	}
	t.subscribers[name] = ch
	return nil
}

func (t *EbpfTracer) Run() {

}
