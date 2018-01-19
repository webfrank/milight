package milight

import (
	"testing"
	"time"
)

var m = New()

func TestOff(t *testing.T) {
	m.Off()
}

func TestOn(t *testing.T) {
	time.Sleep(1 * time.Second)
	m.On()
}

func TestMode(t *testing.T) {
	m.Alert()
}

/*
func TestColor(t *testing.T) {
	for i := 0; i < 256; i++ {
		time.Sleep(100 * time.Millisecond)
		Color(byte(i))
	}

}
*/
/*
func TestWhite(t *testing.T) {
	time.Sleep(1 * time.Second)
	m.White()
}

func TestBrightness(t *testing.T) {
	for i := 0; i < 100; i = i + 10 {
		time.Sleep(500 * time.Millisecond)
		m.Brightness(byte(i))
	}

}
*/
