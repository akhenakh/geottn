package gw

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/akhenakh/cayenne"
	"github.com/brocaar/lorawan"
	log "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"google.golang.org/grpc/health"

	"github.com/akhenakh/geottn/metrics"
	"github.com/akhenakh/geottn/storage"
)

type Server struct {
	appName string
	logger  log.Logger
	Health  *health.Server
	GeoDB   storage.Indexer
	udpConn *net.UDPConn
}

func NewServer(appName string, logger log.Logger, idx storage.Indexer) *Server {
	logger = log.With(logger, "component", "gw")
	return &Server{
		appName: appName,
		logger:  logger,
		GeoDB:   idx,
	}
}

func (s *Server) Close() {
	s.udpConn.Close()
}

func (s *Server) StartListener(ctx context.Context, addr string) error {
	ServerAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		level.Error(s.logger).Log("msg", "gw server: failed to resolve", "error", err)
		return err
	}

	/* Now listen at selected port */
	s.udpConn, err = net.ListenUDP("udp", ServerAddr)
	if err != nil {
		level.Error(s.logger).Log("msg", "gw server: failed to listen", "error", err)
		return err
	}

	level.Info(s.logger).Log("msg", fmt.Sprintf("GW UDP server listening at %s", addr))

	buf := make([]byte, 1024)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, addr, err := s.udpConn.ReadFromUDP(buf)
				if err != nil {
					level.Warn(s.logger).Log("msg", "error reading on the GW", "error", err)
					continue
				}
				err = s.handleUpstream(addr, buf[0:n])
				if err != nil {
					level.Error(s.logger).Log("msg", "error handling msg received on the GW", "error", err)
					continue
				}
			}
		}
	}()
	return nil
}

// decodeLora returns deviceID payload error
func (s *Server) decodeLora(p []byte) ([]byte, []byte, error) {
	nwkSKey := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	appSKey := [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}

	var phy lorawan.PHYPayload

	if err := phy.UnmarshalBinary(p); err != nil {
		panic(err)
	}

	ok, err := phy.ValidateUplinkDataMIC(lorawan.LoRaWAN1_0, 0, 0, 0, nwkSKey, lorawan.AES128Key{})
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, errors.New("invalid mic")
	}

	if err := phy.DecodeFOptsToMACCommands(); err != nil {
		return nil, nil, err
	}

	if err := phy.DecryptFRMPayload(appSKey); err != nil {
		return nil, nil, err
	}
	macPL, ok := phy.MACPayload.(*lorawan.MACPayload)
	if !ok {
		return nil, nil, errors.New("MACPayload expected")
	}

	pl, ok := macPL.FRMPayload[0].(*lorawan.DataPayload)
	if !ok {
		return nil, nil, errors.New("DataPayload expected")
	}

	return nil, pl.Bytes, nil
}

func (s *Server) handleUpstream(addr *net.UDPAddr, p []byte) error {
	if len(p) < 12 {
		return errors.New("invalid packet length")
	}

	//0      | protocol version = 2
	if p[0] != 2 {
		return errors.New("invalid packet protocol version")
	}

	//1-2    | random token
	token := p[1:2]

	//3      | PUSH_DATA identifier 0x00
	if p[3] != 0x00 {
		return errors.New("invalid packet not a PUSH_DATA")
	}

	//4-11   | Gateway unique identifier (MAC address)
	gwID := p[4:11]

	jsonb := p[12:]

	ujson := &UpstreamJSON{}
	err := json.Unmarshal(jsonb, &ujson)
	if err != nil {
		return err
	}

	// this could be a stat packet
	if len(ujson.Rxpk) == 0 {
		return nil
	}

	for _, p := range ujson.Rxpk {
		p.GwID = gwID
		p.Token = token

		_, lorap, err := s.decodeLora(p.Data)
		if err != nil {
			level.Info(s.logger).Log("msg", "can't decode uplink lora packet", "error", err)
			continue
		}
		r := bytes.NewReader(lorap)
		d := cayenne.NewDecoder(r)
		msg, err := d.DecodeUplink()
		if err != nil {
			level.Info(s.logger).Log("msg", "can't decode uplink cayenne packet", "error", err)
			continue
		}

		metrics.MsgReceivedCounter.WithLabelValues(metrics.ViaLabel, metrics.ReceivedViaGW)

		locKey, ok := msg.GotLocation()
		if !ok {
			level.Info(s.logger).Log("msg", "cayenne packet does not contain coordinates")
			continue
		}

		locf := msg.Values()[locKey].([]float32)

		err = s.GeoDB.Store(p.Codr, p.Data, float64(locf[0]), float64(locf[1]), p.Time)
		if err != nil {
			level.Error(s.logger).Log("msg", "can't store packet in DB", "error", err)
			continue
		}

		metrics.InsertCounter.Inc()
	}

	return nil
}
