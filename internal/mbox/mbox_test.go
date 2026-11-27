// SPDX-License-Identifier: GPL-3.0-or-later

package mbox

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func readAll(t *testing.T, src string) []string {
	t.Helper()
	var out []string
	err := Read(strings.NewReader(src), func(msg []byte) error {
		out = append(out, string(msg))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestReadSplitsAndUnescapes(t *testing.T) {
	src := "From a@b Sat Jan  3 01:05:34 1996\nSubject: one\n\nhello\n>From the past\n>>From deeper\n\n" +
		"From c@d Sun Jan  4 01:05:34 1996\nSubject: two\n\nFrom here on, not a separator line because no blank line before? it is not.\nbye\n"
	got := readAll(t, src)
	if len(got) != 2 {
		t.Fatalf("%d messages: %q", len(got), got)
	}
	if got[0] != "Subject: one\r\n\r\nhello\r\nFrom the past\r\n>From deeper\r\n" {
		t.Errorf("first %q", got[0])
	}
	if !strings.HasPrefix(got[1], "Subject: two\r\n") || !strings.Contains(got[1], "\r\nFrom here on") {
		t.Errorf("second %q", got[1])
	}
}

func TestReadSingleEML(t *testing.T) {
	got := readAll(t, "Subject: lone\r\n\r\nbody\r\n")
	if len(got) != 1 || got[0] != "Subject: lone\r\n\r\nbody\r\n" {
		t.Fatalf("%q", got)
	}
}

func TestWriteRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	msgs := []string{"Subject: one\r\n\r\nFrom me\r\n>From quoted\r\n", "Subject: two\r\n\r\nno trailing newline"}
	for _, m := range msgs {
		if err := w.Write([]byte(m), "a@b.test", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.HasPrefix(buf.String(), "From a@b.test Fri Jan  2 03:04:05 2026\n") {
		t.Errorf("separator %q", buf.String()[:60])
	}
	got := readAll(t, buf.String())
	if len(got) != 2 || got[0] != msgs[0] || got[1] != msgs[1]+"\r\n" {
		t.Fatalf("%q", got)
	}
}
