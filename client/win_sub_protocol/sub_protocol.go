package win_sub_protocol

import (
	"bard/bard"
	sp_test "bard/bard-plugin/sub_protocol/test"
)

var subProtocols = []bard.TCSubProtocol{
	sp_test.T,
}

func WinSubProtocols() *bard.TCSubProtocols {
	ts := &bard.TCSubProtocols{}

	ts.Init()

	for _, v := range subProtocols {
		ts.Register(v)
	}

	return ts
}