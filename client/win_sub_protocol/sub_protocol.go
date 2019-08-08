package win_sub_protocol

import (
	"bard/bard"
)

var subProtocols = []bard.TCSubProtocol{

}

func WinSubProtocols() *bard.TCSubProtocols {
	ts := &bard.TCSubProtocols{}

	ts.Init()

	for _, v := range subProtocols {
		ts.Register(v)
	}

	return ts
}