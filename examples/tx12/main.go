// Пример использования пакета hidjoystick с пультом RadioMaster TX12.
//
// Запуск:
//
//	go run .
//
// Предварительно на TX12: System Settings → USB Mode → HID Joystick
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asperepelov/hid-joystick"
	"github.com/asperepelov/hid-joystick/hidjoystick"
)

// defaultKeywords — ключевые слова для поиска TX12 среди HID-устройств.
var defaultKeywords = []string{"RadioMaster", "TX12", "EdgeTX", "OpenTX"}

// ── TX12 маппинг репорта ──────────────────────────────────────────────────────
//
// Определено опытным путём через диагностику сырого репорта.
// При необходимости скорректируйте офсеты под вашу версию прошивки.

const (
	offButtons = 1  // uint16 LE: биты Btn1..Btn4
	offCH1     = 4  // Roll     uint16 LE
	offCH2     = 6  // Pitch    uint16 LE
	offCH3     = 8  // Throttle uint16 LE
	offCH4     = 10 // Yaw      uint16 LE
	offCH5     = 12 // uint16 LE (3-поз переключатель)
	offCH6     = 14 // uint16 LE (3-поз переключатель)
	offCH7     = 16 // uint16 LE (3-поз переключатель)
	offCH8     = 18 // uint16 LE (3-поз переключатель)

	// Значения 3-поз переключателей (определено опытным путём)
	swDOWN uint16 = 0
	swMID  uint16 = 1024
	swUP   uint16 = 2047

	// Биты кнопок в offButtons
	bitBtn1 = 0
	bitBtn2 = 1
	bitBtn3 = 2
	bitBtn4 = 3
)

// SwitchPos — позиция 3-позиционного переключателя.
type SwitchPos int

const (
	SwitchDown SwitchPos = -1
	SwitchMid  SwitchPos = 0
	SwitchUp   SwitchPos = 1
)

func (s SwitchPos) String() string {
	switch s {
	case SwitchUp:
		return "↑UP  "
	case SwitchDown:
		return "↓DOWN"
	default:
		return "─MID "
	}
}

// JoystickState — состояние всех органов управления TX12.
// Значения каналов — сырые uint16 из HID-репорта, без нормализации.
type JoystickState struct {
	CH1  uint16 // Обычно Roll (правый стик, горизонталь)
	CH2  uint16 // Обычно Pitch (правый стик, вертикаль)
	CH3  uint16 // Обычно Throttle (левый стик, вертикаль)
	CH4  uint16 // Обычно Yaw (левый стик, горизонталь)
	CH5  uint16 // 3-поз переключатель (сырое значение)
	CH6  uint16 // 3-поз переключатель (сырое значение)
	CH7  uint16 // 3-поз переключатель (сырое значение)
	CH8  uint16 // 3-поз переключатель (сырое значение)
	CH9  uint16
	CH10 uint16
	CH11 uint16
	CH12 uint16

	// Интерпретированные позиции переключателей
	SW5 SwitchPos
	SW6 SwitchPos
	SW7 SwitchPos
	SW8 SwitchPos

	Btn1 bool // SA
	Btn2 bool // SD
	Btn3 bool // S1
	Btn4 bool // S2

	Raw []byte // сырые байты репорта для отладки
}

// ── Парсинг ───────────────────────────────────────────────────────────────────

func absDiff(a, b uint16) uint16 {
	if a > b {
		return a - b
	}
	return b - a
}

func valToSwitch(v uint16) SwitchPos {
	dUp := absDiff(v, swUP)
	dMid := absDiff(v, swMID)
	dDown := absDiff(v, swDOWN)
	if dUp <= dMid && dUp <= dDown {
		return SwitchUp
	}
	if dMid <= dDown {
		return SwitchMid
	}
	return SwitchDown
}

// parseReport разбирает hidjoystick.Report в JoystickState.
func parseReport(r hidjoystick.Report) (JoystickState, bool) {
	if r.Len() < 20 {
		return JoystickState{}, false
	}

	raw := make([]byte, r.Len())
	copy(raw, r.Data)

	ch5 := r.U16LE(offCH5)
	ch6 := r.U16LE(offCH6)
	ch7 := r.U16LE(offCH7)
	ch8 := r.U16LE(offCH8)

	return JoystickState{
		CH1: r.U16LE(offCH1),
		CH2: r.U16LE(offCH2),
		CH3: r.U16LE(offCH3),
		CH4: r.U16LE(offCH4),
		CH5: ch5,
		CH6: ch6,
		CH7: ch7,
		CH8: ch8,
		// CH9..CH12 не переданы в HID-репорте TX12 (только 8 каналов)

		Btn1: r.BitU16(offButtons, bitBtn1),
		Btn2: r.BitU16(offButtons, bitBtn2),
		Btn3: r.BitU16(offButtons, bitBtn3),
		Btn4: r.BitU16(offButtons, bitBtn4),

		SW5: valToSwitch(ch5),
		SW6: valToSwitch(ch6),
		SW7: valToSwitch(ch7),
		SW8: valToSwitch(ch8),

		Raw: raw,
	}, true
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== RadioMaster TX12 — пример hidjoystick ===")

	if !hidjoystick.IsAvailable(defaultKeywords) {
		fmt.Println("TX12 не найден. Ожидание подключения...")
	}

	ctrl, err := hidjoystick.WaitForDevice(defaultKeywords, time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", err)
		os.Exit(1)
	}
	defer ctrl.Close()

	info := ctrl.Info()
	fmt.Printf("Подключено: %s  (VID=%04X PID=%04X)\n", info.Name, info.VendorID, info.ProductID)
	fmt.Println("Ctrl+C для выхода")
	time.Sleep(300 * time.Millisecond)

	ctrl.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var last JoystickState
	var hasState bool

	for {
		select {
		case <-sig:
			fmt.Print("\033[2J\033[H")
			fmt.Println("Bye!")
			return

		case err := <-ctrl.Errors():
			fmt.Fprintln(os.Stderr, "Ошибка чтения:", err)

		case r := <-ctrl.Reports():
			if s, ok := parseReport(r); ok {
				last = s
				hasState = true
			}

		case <-ticker.C:
			if hasState {
				render(last, info.Name)
			}
		}
	}
}

// ── Rendering ─────────────────────────────────────────────────────────────────

const barWidth = 20

func bar(val uint16, max uint16, w int) string {
	b := make([]rune, w)
	for i := range b {
		b[i] = '░'
	}
	center := w / 2
	pos := int(float64(val) / float64(max) * float64(w))
	if pos < 0 {
		pos = 0
	} else if pos >= w {
		pos = w - 1
	}
	b[center] = '│'
	b[pos] = '█'
	return string(b)
}

func barT(val uint16, max uint16, w int) string {
	b := make([]rune, w)
	for i := range b {
		b[i] = '░'
	}
	filled := int(float64(val) / float64(max) * float64(w))
	if filled > w {
		filled = w
	}
	for i := 0; i < filled; i++ {
		b[i] = '█'
	}
	return string(b)
}

func btn(v bool) string {
	if v {
		return "■ ON "
	}
	return "□ off"
}

func render(s JoystickState, name string) {
	const axisMax uint16 = 0xFF07

	fmt.Print("\033[H")
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║        RadioMaster TX12  ·  hidjoystick example              ║")
	fmt.Printf("║  %-60s║\n", name)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  CHANNELS  (raw uint16)                                      ║")
	fmt.Printf("║  CH1  Roll     [%s] %5d              ║\n", bar(s.CH1, axisMax, barWidth), s.CH1)
	fmt.Printf("║  CH2  Pitch    [%s] %5d              ║\n", bar(s.CH2, axisMax, barWidth), s.CH2)
	fmt.Printf("║  CH3  Throttle [%s] %5d              ║\n", barT(s.CH3, axisMax, barWidth), s.CH3)
	fmt.Printf("║  CH4  Yaw      [%s] %5d              ║\n", bar(s.CH4, axisMax, barWidth), s.CH4)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  SWITCHES                                                    ║")
	fmt.Printf("║  CH5:%s(%5d)  CH6:%s(%5d)                    ║\n",
		s.SW5, s.CH5, s.SW6, s.CH6)
	fmt.Printf("║  CH7:%s(%5d)  CH8:%s(%5d)                    ║\n",
		s.SW7, s.CH7, s.SW8, s.CH8)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  BUTTONS                                                     ║")
	fmt.Printf("║  Btn1(SA):%s  Btn2(SD):%s  Btn3(S1):%s  Btn4(S2):%s ║\n",
		btn(s.Btn1), btn(s.Btn2), btn(s.Btn3), btn(s.Btn4))
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  RAW HID REPORT                                              ║")
	hdr := "║  off:"
	hex := "║  hex:"
	dec := "║  dec:"
	for i := 0; i < len(s.Raw) && i < 20; i++ {
		hdr += fmt.Sprintf(" %3d", i)
		hex += fmt.Sprintf("  %02X", s.Raw[i])
		dec += fmt.Sprintf(" %3d", s.Raw[i])
	}
	fmt.Printf("%-65s║\n", hdr)
	fmt.Printf("%-65s║\n", hex)
	fmt.Printf("%-65s║\n", dec)
	pairs := "║  u16:"
	for i := 0; i+1 < len(s.Raw) && i < 20; i += 2 {
		v := uint16(s.Raw[i]) | uint16(s.Raw[i+1])<<8
		pairs += fmt.Sprintf(" %5d", v)
	}
	fmt.Printf("%-65s║\n", pairs)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Ctrl+C to exit                                              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}
