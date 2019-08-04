package plugin

import (
	"bard/bard-plugin/base"
)



var V = base.Plugin {
	ID:  "base",
	Ver: "0.1.0",
	Pri: 0x3111,
	DESKEY: []byte("12345678"),
	END_FLAG: []byte("\r\n\r\n"),
}

