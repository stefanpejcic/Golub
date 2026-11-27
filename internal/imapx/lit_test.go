// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import "strings"

type literalReader struct{ *strings.Reader }

func (l literalReader) Size() int64 { return int64(l.Len()) }

func lit(s string) literalReader { return literalReader{strings.NewReader(s)} }
