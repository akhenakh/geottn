package gw

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

var raw = `{"rxpk":[
	{
		"time":"2013-03-31T16:21:17.528002Z",
		"tmst":3512348611,
		"chan":2,
		"rfch":0,
		"freq":866.349812,
		"stat":1,
		"modu":"LORA",
		"datr":"SF7BW125",
		"codr":"4/6",
		"rssi":-35,
		"lsnr":5.1,
		"size":32,
		"data":"RkFLRQo="
	},{
		"time":"2013-03-31T16:21:17.530974Z",
		"tmst":3512348514,
		"chan":9,
		"rfch":1,
		"freq":869.1,
		"stat":1,
		"modu":"FSK",
		"datr":50000,
		"rssi":-75,
		"size":16,
		"data":"VEVTVF9QQUNLRVRfMTIzNA=="
	},{
		"time":"2013-03-31T16:21:17.532038Z",
		"tmst":3316387610,
		"chan":0,
		"rfch":0,
		"freq":863.00981,
		"stat":1,
		"modu":"LORA",
		"datr":"SF10BW125",
		"codr":"4/7",
		"rssi":-38,
		"lsnr":5.5,
		"size":32,
		"data":"RkFLRQo="
	}
]}`

func TestDecodeJSON(t *testing.T) {
	ujson := &UpstreamJSON{}
	err := json.Unmarshal([]byte(raw), &ujson)
	require.NoError(t, err)
	require.Len(t, ujson.Rxpk, 3)
	for _, p := range ujson.Rxpk {
		t.Log(p)
	}
}
