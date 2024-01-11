package proc

import (
	"os"
	"path"
	"strconv"
)

const PROC_PATH = "/proc"

func Path(pid uint32, subpath ...string) string {
	return path.Join(append([]string{PROC_PATH, strconv.Itoa(int(pid))}, subpath...)...)
}

func GetAllPids() ([]uint32, error) {
	var pids []uint32
	dir, err := os.Open(PROC_PATH)
	defer dir.Close()
	if err != nil {
		return nil, err
	}
	files, err := dir.ReadDir(0)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			pid, err := strconv.ParseUint(file.Name(), 10, 32)
			if err != nil {
				continue
			}
			if _, err := os.Stat(Path(uint32(pid))); err == nil {
				pids = append(pids, uint32(pid))
			}
		}
	}
	return pids, nil
}

func ProcRootSubpath(subpath ...string) string {
	return Path(1, append([]string{"root"}, subpath...)...)
}

func ProcSubpath(pid uint32, subpath ...string) string {
	return Path(pid, append([]string{"root"}, subpath...)...)
}
