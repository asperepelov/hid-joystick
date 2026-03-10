// Пример: полный монитор RadioMaster TX12.
// Запуск: go run .
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asperepelov/hid-joystick/tx12"
)

func main() {
	fmt.Println("=== RadioMaster TX12 Monitor ===")

	if !tx12.IsAvailable() {
		fmt.Println("TX12 не найден. Ожидание подключения...")
	}

	ctrl, err := tx12.WaitForDevice(time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", err)
		os.Exit(1)
	}
	defer ctrl.Close()

	info := ctrl.Info()
	fmt.Printf("Подключено: %s  (VID=%04X PID=%04X)\n", info.Name, info.VendorID, info.ProductID)
	time.Sleep(300 * time.Millisecond)

	ctrl.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var last tx12.State
	var ready bool

	fmt.Print("\033[2J")
	for {
		select {
		case <-sig:
			fmt.Print("\033[2J\033[H")
			fmt.Println("Bye!")
			return
		case err := <-ctrl.Errors():
			fmt.Fprintln(os.Stderr, "Ошибка:", err)
		case s := <-ctrl.States():
			last = s
			ready = true
		case <-ticker.C:
			if ready {
				render(last, info.Name)
			}
		}
	}
}

const axisMax uint16 = 2047

func bar(val, max uint16, w int) string {
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

func barT(val, max uint16, w int) string {
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

func render(s tx12.State, name string) {
	fmt.Print("\033[H")
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║        RadioMaster TX12  ·  Live Input Monitor               ║")
	fmt.Printf("║  %-60s║\n", name)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  CHANNELS                                                    ║")
	fmt.Printf("║  CH1 Roll     [%s] %5d              ║\n", bar(s.CH1, axisMax, 20), s.CH1)
	fmt.Printf("║  CH2 Pitch    [%s] %5d              ║\n", bar(s.CH2, axisMax, 20), s.CH2)
	fmt.Printf("║  CH3 Throttle [%s] %5d              ║\n", barT(s.CH3, axisMax, 20), s.CH3)
	fmt.Printf("║  CH4 Yaw      [%s] %5d              ║\n", bar(s.CH4, axisMax, 20), s.CH4)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  SWITCHES                                                    ║")
	fmt.Printf("║  CH5 %-4s (%5d)  CH6 %-4s (%5d)                      ║\n",
		s.SW5, s.CH5, s.SW6, s.CH6)
	fmt.Printf("║  CH7 %-4s (%5d)  CH8 %-4s (%5d)                      ║\n",
		s.SW7, s.CH7, s.SW8, s.CH8)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  BUTTONS                                                     ║")
	fmt.Printf("║  Btn1(SA):%s  Btn2(SD):%s  Btn3(S1):%s  Btn4(S2):%s ║\n",
		btn(s.Btn1), btn(s.Btn2), btn(s.Btn3), btn(s.Btn4))
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  RAW                                                         ║")
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
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Ctrl+C to exit                                              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}
