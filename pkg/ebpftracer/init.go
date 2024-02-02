package ebpftracer

import (
	"strings"

	"k8s.io/klog/v2"

	"github.com/kwaisu/sense-agent/pkg/system"
)

type file struct {
	pid uint32
	fd  uint64
}

type sock struct {
	pid uint32
	fd  uint64
	system.Sock
}

func readFds(pids []uint32) (files []file, socks []sock) {
	nss := map[string]map[string]sock{}
	for _, pid := range pids {
		ns, err := system.GetNetNs(pid)
		if err != nil {
			continue
		}
		nsId := ns.UniqueId()
		sockets, ok := nss[nsId]
		_ = ns.Close()
		if !ok {
			sockets = map[string]sock{}
			nss[nsId] = sockets
			if ss, err := system.GetSockets(pid); err != nil {
				klog.Warningln(err)
			} else {
				for _, s := range ss {
					sockets[s.Inode] = sock{Sock: s}
				}
			}
		}

		fds, err := system.ReadFds(pid)
		if err != nil {
			continue
		}
		for _, fd := range fds {
			switch {
			case fd.SocketInode != "":
				if s, ok := sockets[fd.SocketInode]; ok {
					s.fd = fd.Fd
					s.pid = pid
					socks = append(socks, s)
				}
			case strings.HasPrefix(fd.Dest, "/"):
				files = append(files, file{pid: pid, fd: fd.Fd})
			}
		}
	}
	return
}
