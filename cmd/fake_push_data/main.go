package main

import (
	"flag"
	"log"
	"net"
)

var raw = `{"rxpk":[
	{
		"time":"2019-11-27T16:21:17.530974Z",
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
		"data":"VEVTVF9QQUNLRVRfMTIzNA=="
	}
]}`

var (
	addr = flag.String("addr", "localhost:1700", "Addr to sent the packet to")
)

func main() {
	flag.Parse()

	raddr, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return
	}

	//  Bytes  | Function
	//:------:|---------------------------------------------------------------------
	// 0      | protocol version = 2
	// 1-2    | random token
	// 3      | PUSH_DATA identifier 0x00
	// 4-11   | Gateway unique identifier (MAC address)
	// 12-end | JSON object, starting with {, ending with }, see section 4

	p := []byte{'2', 'A', 'B', 0x00, 0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0xDE, 0xAD, 0xBE}

	p = append(p, []byte(raw)...)
	_, err = conn.Write(p)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("sent", p)
}
