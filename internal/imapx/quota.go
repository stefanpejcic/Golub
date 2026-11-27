// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// QuotaInfo is storage usage in bytes.
type QuotaInfo struct {
	Used  int64 `json:"used"`
	Limit int64 `json:"limit"`
}

// Quota returns INBOX storage usage, or nil when the server has no quota.
func Quota(c *imapclient.Client) (*QuotaInfo, error) {
	if !c.Caps().Has(imap.CapQuota) {
		return nil, nil
	}
	data, err := c.GetQuotaRoot("INBOX").Wait()
	if err != nil {
		return nil, err
	}
	return storageFromQuota(data), nil
}

// RFC 9208 counts STORAGE in units of 1024 octets.
func storageFromQuota(data []imapclient.QuotaData) *QuotaInfo {
	for _, d := range data {
		if r, ok := d.Resources[imap.QuotaResourceStorage]; ok && r.Limit > 0 {
			return &QuotaInfo{Used: r.Usage * 1024, Limit: r.Limit * 1024}
		}
	}
	return nil
}
