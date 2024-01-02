package log

import (
	"fmt"
	"testing"

	journal "gitee.com/oschina/sense-agent/pkg/log"
)

func TestOtelExporter(t *testing.T) {
	exporter, err := NewExporter("test", "gitee", "1.0", "127.0.0.1:4318")
	if err != nil {
		fmt.Println(err)
	}
	ch := make(chan journal.Message)
	createJournal(ch)
	provider := NewLoggerProvider(exporter, ch, ExportProverConfig{
		maxLines: 10,
		maxBytes: 102400,
	},
	)
	provider.Start()
	for {

	}
}

func createJournal(ch chan journal.Message) {
	if j, err := journal.NewJournalReader("_TRANSPORT", []string{"/proc/1/root/run/log/journal", "/proc/1/root/var/log/journal"}); err != nil {
		fmt.Println(err)
	} else {
		if err := j.Subscribe("kernel", ch); err != nil {
			fmt.Println(err)
		}
	}
}
