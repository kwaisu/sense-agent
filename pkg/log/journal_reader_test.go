package log

import (
	"fmt"
	"testing"

	"github.com/coreos/go-systemd/v22/sdjournal"
)

// timestamp:  2023-12-21 15:00:39.104844 +0800 CST --Levle:  INFO --Content:  time="2023-12-21T15:00:39.104700154+08:00" level=info msg="API listen on /run/docker.sock"
func TestCreateJournalReader(t *testing.T) {
	if j, err := NewJournalReader(sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT, []string{"/proc/1/root/run/log/journal", "/proc/1/root/var/log/journal"}); err != nil {
		fmt.Println(err)
	} else {
		ch := make(chan Message)
		if err := j.Subscribe("docker.service", ch); err != nil {
			fmt.Println(err)
		}
		for {
			select {
			case entry := <-ch:
				fmt.Println("timestamp: ", entry.Timestamp, "--Levle: ", entry.Level, "--Content: ", entry.Content)
			}
		}
	}
}

func TestJournalReadKernel(t *testing.T) {
	if j, err := NewJournalReader(sdjournal.SD_JOURNAL_FIELD_TRANSPORT, []string{"/proc/1/root/run/log/journal", "/proc/1/root/var/log/journal"}); err != nil {
		fmt.Println(err)
	} else {
		ch := make(chan Message)
		if err := j.Subscribe("kernel", ch); err != nil {
			fmt.Println(err)
		}
		for {
			select {
			case entry := <-ch:
				fmt.Println("timestamp: ", entry.Timestamp, "--Levle: ", entry.Level, "--Content: ", entry.Content)
			}
		}
	}
}
