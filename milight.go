package milight

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"strings"
	"time"
)

const (
	DISCOVER = "HF-A11ASSISTHREAD"
)

var (
	LOGIN      = []byte{0x20, 0x00, 0x00, 0x00, 0x16, 0x02, 0x62, 0x3A, 0xD5, 0xED, 0xA3, 0x01, 0xAE, 0x08, 0x2D, 0x46, 0x61, 0x41, 0xA7, 0xF6, 0xDC, 0xAF, 0xD3, 0xE6, 0x00, 0x00, 0x1E}
	HEADER     = []byte{0x80, 0x00, 0x00, 0x00, 0x11}
	ON         = []byte{0x31, 0x00, 0x00, 0x00, 0x03, 0x03, 0x00, 0x00, 0x00}
	OFF        = []byte{0x31, 0x00, 0x00, 0x00, 0x03, 0x04, 0x00, 0x00, 0x00}
	MODE       = []byte{0x31, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00} // MODE[5]=mode
	MODESLOW   = []byte{0x31, 0x00, 0x00, 0x00, 0x03, 0x01, 0x00, 0x00, 0x00}
	MODEFAST   = []byte{0x31, 0x00, 0x00, 0x00, 0x03, 0x02, 0x00, 0x00, 0x00}
	COLOR      = []byte{0x31, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00} // COLOR[5:]=color
	WHITE      = []byte{0x31, 0x00, 0x00, 0x00, 0x03, 0x05, 0x00, 0x00, 0x00}
	BRIGHTNESS = []byte{0x31, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00} // BRIGHTNESS[5]=brightness 0-100
)

type Milight struct {
	conn  *net.UDPConn
	rAddr *net.UDPAddr
	wb1   byte
	wb2   byte
	sn    byte
}

func New() *Milight {
	m := new(Milight)
	m.init()

	return m
}

func (m *Milight) Destroy() {
	m.conn.Close()
	m.conn = nil
}

func (m *Milight) Off() {
	m.retryCmd(m.buildCmd(OFF))
}

func (m *Milight) On() {
	m.retryCmd(m.buildCmd(ON))
}

func (m *Milight) Mode(mode byte) {
	cmd := MODE
	cmd[5] = mode
	m.retryCmd(m.buildCmd(cmd))
}

func (m *Milight) ModeSlow() {
	m.retryCmd(m.buildCmd(MODESLOW))
}

func (m *Milight) ModeFast() {
	m.retryCmd(m.buildCmd(MODEFAST))
}

func (m *Milight) Color(color byte) {
	cmd := COLOR
	cmd[5] = color
	cmd[6] = color
	cmd[7] = color
	cmd[8] = color
	m.retryCmd(m.buildCmd(cmd))
}

func (m *Milight) White() {
	m.retryCmd(m.buildCmd(WHITE))
}

func (m *Milight) Brightness(brightness byte) {
	cmd := BRIGHTNESS
	cmd[5] = byte(math.Min(float64(brightness), 100))
	m.retryCmd(m.buildCmd(cmd))
}

func (m *Milight) Alert() {
	m.On()
	time.Sleep(time.Millisecond * 100)
	m.Brightness(100)
	time.Sleep(time.Millisecond * 100)
	m.Mode(6)
}

// Init MiLight
func (m *Milight) init() {
	lAddr, err := net.ResolveUDPAddr("udp4", ":5987")
	checkError(err)

	m.conn, err = net.ListenUDP("udp", lAddr)
	checkError(err)

	m.login()
}

// Discover WiFi Bridge V6
func (m *Milight) discover() (string, error) {

	rAddr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:48899")
	checkError(err)

	lAddr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:48899")
	checkError(err)

	conn, err := net.ListenUDP("udp4", lAddr)
	checkError(err)
	defer conn.Close()

	_, err = conn.WriteToUDP([]byte(DISCOVER), rAddr)
	checkError(err)

	buf := make([]byte, 1024)
	deadline := time.Now().Add(1 * time.Second)
	conn.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		//fmt.Printf("Listening...\n")
		n, _, _ := conn.ReadFromUDP(buf)
		//fmt.Println(addr, n, string(buf[0:n]))
		s := strings.Split(string(buf[0:n]), ",")
		fmt.Println(s)
		if len(s) == 3 {
			return s[0], nil
		}
	}
	return "", errors.New("Discovery Timeout")
}

func (m *Milight) login() error {
	remote, err := m.discover()
	if err == nil {
		m.rAddr = new(net.UDPAddr)
		m.rAddr.IP = net.ParseIP(remote)
		m.rAddr.Port = 5987

		response := m.sendCmd(LOGIN)
		if len(response) == 22 && response[0] == 0x28 {
			m.wb1 = response[19]
			m.wb2 = response[20]

			//fmt.Printf("WifiBridge ID %x %x\n", m.wb1, m.wb2)
			return nil
		}
		return errors.New("Wrong reply")

	}

	return errors.New("Discovery Error, retry")
}

func (m *Milight) retryCmd(cmd []byte) error {
	tries := 3
	for tries > 0 {
		resp := m.sendCmd(cmd)
		if cmd[8] == resp[6] && resp[7] == 0 {
			break
		}
		m.login()
		tries--
	}

	if tries == 0 {
		return errors.New("Error sending command after retry")
	}

	return nil
}

func (m *Milight) sendCmd(cmd []byte) []byte {
	buf := make([]byte, 1024)

	_, _ = m.conn.WriteToUDP(cmd, m.rAddr)

	m.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, _, err := m.conn.ReadFromUDP(buf)
	if err != nil {
		fmt.Println("Error receiving reply", err, "login again")
		m.login()
	} else {
		//fmt.Println(addr, n, hex.EncodeToString(buf[0:n]))
		return buf[0:n]
	}

	return nil
}

func (m *Milight) buildCmd(cmd []byte) []byte {
	resp := make([]byte, 22)
	copy(resp[0:], HEADER)
	resp[5] = m.wb1
	resp[6] = m.wb2
	resp[8] = m.sn
	copy(resp[10:], cmd)
	resp[19] = 0x01 //Zone ID
	resp[21] = m.checksum(resp[10:20])

	m.sn = m.sn + 1

	return resp
}

func (m *Milight) checksum(bytes []byte) byte {
	var s byte
	for _, b := range bytes {
		s += b
	}

	return s
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error %v\n", err.Error())
		os.Exit(1)
	}
}
