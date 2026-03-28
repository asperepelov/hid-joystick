// Пример: минимальный вывод каналов, кнопок и raw-байтов TX12.
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
	fmt.Printf("Wait for TX12...\n")
	ctrl, err := tx12.WaitForDevice(time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer ctrl.Close()

	fmt.Printf("Connected: %s\n", ctrl.Info().Name)
	ctrl.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	fmt.Print("\033[2J")
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var last *tx12.State
	var ready bool

	for {
		select {
		case <-sig:
			fmt.Println("\nBye!")
			return
		case s := <-ctrl.States():
			last = s
			ready = true
		case e := <-ctrl.Errors():
			fmt.Printf("\nError TX12: %v\n", e)
			return
		case <-ticker.C:
			if !ready {
				continue
			}
			s := last
			fmt.Print("\033[H")
			fmt.Println("── Channels ────────────────────────────")
			fmt.Printf("  CH1 %5d  CH2 %5d  CH3 %5d  CH4 %5d  CH5 %5d  CH6 %5d  CH7 %5d  CH8 %5d  CH9 %5d  CH10 %5d  CH11 %5d  CH12 %5d\n",
				s.CH1, s.CH2, s.CH3, s.CH4, s.CH5, s.CH6, s.CH7, s.CH8, s.CH9, s.CH10, s.CH11, s.CH12)
			fmt.Println("── Switches ────────────────────────────")
			fmt.Printf("  CH5 %-4s  CH6 %-4s  CH7 %-4s  CH8 %-4s\n",
				s.SW5, s.SW6, s.SW7, s.SW8)
			fmt.Println("── Buttons ─────────────────────────────")
			fmt.Printf("  SA %-5v  SD %-5v  S1 %-5v  S2 %-5v\n",
				s.Btn1, s.Btn2, s.Btn3, s.Btn4)
			fmt.Println("── Raw ─────────────────────────────────")
			for i, b := range s.Raw {
				fmt.Printf("%02X ", b)
				if (i+1)%10 == 0 {
					fmt.Println()
				}
			}
			fmt.Println()
		}
	}
}
