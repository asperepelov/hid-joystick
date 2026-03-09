package hidjoystick

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows"
)

// Info содержит информацию об открытом HID-устройстве.
type Info struct {
	Name      string
	VendorID  uint16
	ProductID uint16
}

// Controller управляет подключением к HID-джойстику и чтением репортов.
// Не привязан ни к какой конкретной модели устройства.
type Controller struct {
	handle   windows.Handle
	info     Info
	reportCh chan Report
	errCh    chan error
	stopCh   chan struct{}
}

// Open находит HID-устройство, имя которого содержит одно из ключевых слов,
// и открывает соединение.
func Open(keywords []string) (*Controller, error) {
	h, name, vid, pid, err := openDevice(keywords)
	if err != nil {
		return nil, fmt.Errorf("hidjoystick: open: %w", err)
	}
	return &Controller{
		handle:   h,
		info:     Info{Name: name, VendorID: vid, ProductID: pid},
		reportCh: make(chan Report, 16),
		errCh:    make(chan error, 1),
		stopCh:   make(chan struct{}),
	}, nil
}

// IsAvailable проверяет, доступно ли устройство прямо сейчас,
// не открывая постоянного соединения.
func IsAvailable(keywords []string) bool {
	h, _, _, _, err := openDevice(keywords)
	if err != nil {
		return false
	}
	windows.CloseHandle(h)
	return true
}

// WaitForDevice блокирует вызывающую горутину до появления устройства,
// проверяя каждые interval. Возвращает открытый Controller.
func WaitForDevice(keywords []string, interval time.Duration) (*Controller, error) {
	for {
		c, err := Open(keywords)
		if err == nil {
			return c, nil
		}
		time.Sleep(interval)
	}
}

// Info возвращает информацию об устройстве.
func (c *Controller) Info() Info {
	return c.info
}

// Close останавливает фоновое чтение и закрывает соединение с устройством.
func (c *Controller) Close() {
	select {
	case <-c.stopCh:
		// уже закрыт
	default:
		close(c.stopCh)
	}
	windows.CloseHandle(c.handle)
}

// ReadOnce выполняет одно блокирующее чтение и возвращает Report.
func (c *Controller) ReadOnce() (Report, error) {
	buf := make([]byte, 64)
	var bytesRead uint32
	if err := windows.ReadFile(c.handle, buf, &bytesRead, nil); err != nil {
		return Report{}, fmt.Errorf("hidjoystick: read: %w", err)
	}
	data := make([]byte, bytesRead)
	copy(data, buf[:bytesRead])
	return Report{Data: data}, nil
}

// Start запускает фоновую горутину чтения репортов.
// Репорты поступают в канал Reports(), ошибки — в Errors().
// Вызовите Close() чтобы остановить.
func (c *Controller) Start() {
	go func() {
		buf := make([]byte, 64)
		for {
			select {
			case <-c.stopCh:
				return
			default:
			}

			var bytesRead uint32
			err := windows.ReadFile(c.handle, buf, &bytesRead, nil)
			if err != nil {
				select {
				case c.errCh <- fmt.Errorf("hidjoystick: read: %w", err):
				default:
				}
				time.Sleep(50 * time.Millisecond)
				continue
			}

			data := make([]byte, bytesRead)
			copy(data, buf[:bytesRead])

			select {
			case c.reportCh <- Report{Data: data}:
			default: // дроп если читатель не успевает
			}
		}
	}()
}

// Reports возвращает канал входящих HID-репортов.
// Используйте после вызова Start().
func (c *Controller) Reports() <-chan Report {
	return c.reportCh
}

// Errors возвращает канал ошибок чтения.
func (c *Controller) Errors() <-chan error {
	return c.errCh
}

// Poll возвращает последний накопившийся репорт из канала без блокировки.
// Удобно использовать в игровом цикле — дропает устаревшие данные.
func (c *Controller) Poll() (Report, bool) {
	var last Report
	var got bool
	for {
		select {
		case r := <-c.reportCh:
			last = r
			got = true
		default:
			return last, got
		}
	}
}
