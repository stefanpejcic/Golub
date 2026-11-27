// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestImportKeepsDateAndExportWalksAll(t *testing.T) {
	c := seededClient(t, 0)
	when := time.Date(2020, 5, 6, 7, 8, 9, 0, time.UTC)
	for i := range 130 {
		raw := fmt.Sprintf("From: Bob <bob@golub.test>\r\nSubject: m%d\r\nDate: %s\r\n\r\nx\r\n", i, when.Format(time.RFC1123Z))
		if err := ImportMessage(c, "Work", []byte(raw)); err != nil {
			t.Fatal(err)
		}
	}
	p, _ := ListMessages(c, "Work", 0, 1, Query{})
	if p.Total != 130 || !p.Messages[0].Date.Equal(when) || !p.Messages[0].Seen {
		t.Fatalf("%d %+v", p.Total, p.Messages[0])
	}

	n := 0
	err := ExportFolder(c, "Work", func(raw []byte, from string, date time.Time) error {
		n++
		if from != "bob@golub.test" || !strings.Contains(string(raw), "Subject: m") || date.IsZero() {
			return fmt.Errorf("bad export %q %v", from, date)
		}
		return nil
	})
	if err != nil || n != 130 {
		t.Fatalf("%v, %d messages", err, n)
	}
}
