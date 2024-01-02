package log

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
)

type JournalReader struct {
	journal     *sdjournal.Journal
	subscribers map[string]chan<- Message
	lock        sync.Mutex
}

func NewJournalReader(journalFieldFilter string, journalPath []string) (j *JournalReader, err error) {
	if len(journalPath) == 0 || len(journalFieldFilter) == 0 {
		return nil, fmt.Errorf("The JournalReader can't be created because the journalFieldFilter or journalPathdoes not exist.")
	}
	j = &JournalReader{
		subscribers: map[string]chan<- Message{},
	}
	for _, path := range journalPath {
		if j.journal, err = sdjournal.NewJournalFromDir(path); err != nil {
			continue
		}
		if usage, err := j.journal.GetUsage(); err != nil || usage == 0 {
			continue
		}
		// if err = j.journal.SeekRealtimeUsec(uint64(time.Now().Add(time.Millisecond).UnixNano() / 1000)); err != nil {
		// 	return nil, err
		// }

		if err = j.journal.SeekRealtimeUsec(1000); err != nil {
			return nil, err
		}
		break
	}
	if j.journal == nil {
		return nil, fmt.Errorf("systemd journal not found in path : %s ", strings.Join(journalPath, ";"))
	}
	go j.fllow(journalFieldFilter)
	return j, nil
}

func (j *JournalReader) fllow(journalFieldFilter string) {
	for {
		if n, err := j.journal.Next(); err != nil {
			fmt.Println("faild to read journal, error: ", err)
			return
		} else {
			if n <= 0 {
				j.journal.Wait(time.Millisecond * 100)
				continue
			}
		}
		if entry, err := j.journal.GetEntry(); err != nil {
			fmt.Println("fail to read jouranl entry")
		} else {
			msg := entry.Fields[sdjournal.SD_JOURNAL_FIELD_MESSAGE]
			if len(msg) == 0 {
				continue
			}
			log := Message{Content: msg,
				Level:     priority2Levels[entry.Fields[sdjournal.SD_JOURNAL_FIELD_PRIORITY]].string(),
				Timestamp: time.UnixMicro(int64(entry.RealtimeTimestamp)),
				Meta:      attr(entry, sdjournal.SD_JOURNAL_FIELD_HOSTNAME, sdjournal.SD_JOURNAL_FIELD_MACHINE_ID, sdjournal.SD_JOURNAL_FIELD_TRANSPORT, sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT),
			}
			j.lock.Lock()
			ch, ok := j.subscribers[entry.Fields[journalFieldFilter]]
			j.lock.Unlock()
			if !ok {
				continue
			}
			ch <- log
		}
	}
}
func attr(entry *sdjournal.JournalEntry, fields ...string) map[string]string {
	attr := make(map[string]string, len(fields))
	for _, field := range fields {
		if val, ok := entry.Fields[field]; ok {
			attr[field] = val
		}
	}
	return attr
}

func (j *JournalReader) Subscribe(JournalFieldVal string, ch chan<- Message) error {
	j.lock.Lock()
	defer j.lock.Unlock()
	if _, ok := j.subscribers[JournalFieldVal]; ok {
		return fmt.Errorf("duplicate subscriber for group %s", JournalFieldVal)
	}
	j.subscribers[JournalFieldVal] = ch
	return nil
}

func (j *JournalReader) Unsubscribe(JournalFieldVal string) {
	j.lock.Lock()
	defer j.lock.Unlock()
	if _, ok := j.subscribers[JournalFieldVal]; ok {
		fmt.Printf("unknow subscribe group %s", JournalFieldVal)
	}
	delete(j.subscribers, JournalFieldVal)
}
