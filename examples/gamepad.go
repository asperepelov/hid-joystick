package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asperepelov/hid-joystick/hidjoystick"
)

// Замените на слова из имени вашего геймпада.
// Если не знаете имя — поставьте []string{""} чтобы открыть первое HID-устройство.
var defaultKeywords = []string{"Gamepad", "Controller", "Joystick"}

func printReport(r hidjoystick.Report, name string) {
	fmt.Print("\033[H")
	fmt.Printf("Device: %s\n", name)
	fmt.Printf("Report length: %d bytes    \n\n", r.Len())

	fmt.Println("── Offset / Hex / Decimal ──")
	for i := 0; i < r.Len(); i++ {
		fmt.Printf("  [%2d]  0x%02X  %3d\n", i, r.Byte(i), r.Byte(i))
	}

	fmt.Println("\n── uint16 LE pairs ───────")
	for i := 0; i+1 < r.Len(); i += 2 {
		fmt.Printf("  [%2d-%2d]  %5d\n", i, i+1, r.U16LE(i))
	}

	fmt.Println("\n── Binary ────────────────")
	for i := 0; i < r.Len(); i++ {
		fmt.Printf("  [%2d]  %08b\n", i, r.Byte(i))
	}

	fmt.Println("\nCtrl+C to exit.")
}

func main() {
	fmt.Println("=== HID Gamepad Raw Viewer ===")
	fmt.Println("Searching for device...")

	if !hidjoystick.IsAvailable(defaultKeywords) {
		fmt.Println("Device not found. Waiting...")
	}

	ctrl, err := hidjoystick.WaitForDevice(defaultKeywords, time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	defer ctrl.Close()

	info := ctrl.Info()
	fmt.Printf("Connected: %s  (VID=%04X PID=%04X)\n", info.Name, info.VendorID, info.ProductID)
	time.Sleep(300 * time.Millisecond)

	ctrl.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	fmt.Print("\033[2J")

	var last hidjoystick.Report
	var ready bool
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sig:
			fmt.Println("\nBye!")
			return
		case r := <-ctrl.Reports():
			last = r
			ready = true
		case <-ticker.C:
			if ready {
				printReport(last, info.Name)
			}
		}
	}
}
