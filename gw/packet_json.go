package gw

import "time"

type UpstreamJSON struct {
	Rxpk []RXPacket
}

type RXPacket struct {
	Time time.Time `json:"time"` // UTC time of pkt RX, us precision, ISO 8601 'compact' format
	Tmms int       // GPS time of pkt RX, number of milliseconds since 06.Jan.1980
	Tmst int       // Internal timestamp of "RX finished" event (32b unsigned)
	Freq float64   // RX central frequency in MHz (unsigned float, Hz precision)
	Chan int       // Concentrator "IF" channel used for RX (unsigned integer)
	Rfch int       // Concentrator "RF chain" used for RX (unsigned integer)
	Stat int       // CRC status: 1 = OK, -1 = fail, 0 = no CRC
	Modu string    // Modulation identifier "LORA" or "FSK"
	// 2 fields with different data type since it's not needed here skip them
	// Datr string    // LoRa datarate identifier (eg. SF12BW500)
	// Datr int       // FSK datarate (unsigned, in bits per second)
	Codr string  // LoRa ECC coding rate identifier
	Rssi int     // RSSI in dBm (signed integer, 1 dB precision)
	Lsnr float64 // Lora SNR ratio in dB (signed float, 0.1 dB precision)
	Size int     // RF packet payload size in bytes (unsigned integer)
	Data []byte  // Base64 encoded RF packet payload, padded

	Token []byte
	GwID  []byte
}
