// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func TestStorageFromQuota(t *testing.T) {
	data := []imapclient.QuotaData{
		{Root: "User quota", Resources: map[imap.QuotaResourceType]imapclient.QuotaResourceData{
			imap.QuotaResourceMessage: {Usage: 10, Limit: 1000},
			imap.QuotaResourceStorage: {Usage: 2048, Limit: 1048576},
		}},
	}
	q := storageFromQuota(data)
	if q == nil || q.Used != 2048*1024 || q.Limit != 1048576*1024 {
		t.Fatalf("%+v", q)
	}
	if storageFromQuota(nil) != nil {
		t.Error("no roots should mean no quota")
	}
	unlimited := []imapclient.QuotaData{{Resources: map[imap.QuotaResourceType]imapclient.QuotaResourceData{imap.QuotaResourceStorage: {Usage: 5, Limit: 0}}}}
	if storageFromQuota(unlimited) != nil {
		t.Error("zero limit means unlimited")
	}
}

func TestQuotaWithoutServerSupport(t *testing.T) {
	c := seededClient(t, 0)
	q, err := Quota(c)
	if err != nil || q != nil {
		t.Fatalf("%+v %v", q, err)
	}
}
