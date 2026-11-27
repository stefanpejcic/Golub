// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import "strings"

type lit struct {
	*strings.Reader
}

func (l lit) Size() int64 { return int64(l.Len()) }

func literal(s string) lit { return lit{strings.NewReader(s)} }
