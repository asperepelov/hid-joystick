// Package tx12 предоставляет готовую реализацию для работы с пультом
// RadioMaster TX12, подключённым по USB в режиме HID Joystick.
//
// Построен поверх пакета hidjoystick и инкапсулирует TX12-специфичный
// маппинг байтов, декодирование каналов и переключателей.
package tx12

import (
	"time"

	"github.com/asperepelov/hid-joystick/hidjoystick"
)

// Ключевые слова для поиска TX12 среди HID-устройств.
var Keywords = []string{"RadioMaster", "TX12", "EdgeTX", "OpenTX"}

// Офсеты байтов в HID-репорте TX12.
// Определены опытным путём; могут отличаться в зависимости от версии прошивки.
const (
	offButtons = 1  // uint16 LE: биты кнопок
	offCH1     = 4  // Roll     uint16 LE
	offCH2     = 6  // Pitch    uint16 LE
	offCH3     = 8  // Throttle uint16 LE
	offCH4     = 10 // Yaw      uint16 LE
	offCH5     = 12 // uint16 LE (3-поз переключатель)
	offCH6     = 14
	offCH7     = 16
	offCH8     = 18

	ValMIN uint16 = 0    // Минимальное значение
	ValMID uint16 = 1024 // Среднее значение
	ValMAX uint16 = 2047 // Максимальное значение

	// Биты кнопок
	bitBtn1 = 0 // SA
	bitBtn2 = 1 // SD
	bitBtn3 = 2 // S1
	bitBtn4 = 3 // S2

	// Минимальная длина валидного репорта
	minReportLen = 20
)

// SwitchPos — позиция 3-позиционного переключателя.
type SwitchPos int8

const (
	SwitchDown SwitchPos = -1
	SwitchMid  SwitchPos = 0
	SwitchUp   SwitchPos = 1
)

func (s SwitchPos) String() string {
	switch s {
	case SwitchUp:
		return "UP"
	case SwitchDown:
		return "DOWN"
	default:
		return "MID"
	}
}

// State — состояние всех органов управления TX12.
// Значения каналов — сырые uint16 из HID-репорта без нормализации.
type State struct {
	CH1  uint16 // Обычно Roll (правый стик, горизонталь)
	CH2  uint16 // Обычно Pitch (правый стик, вертикаль)
	CH3  uint16 // Обычно Throttle (левый стик, вертикаль)
	CH4  uint16 // Обычно Yaw (левый стик, горизонталь)
	CH5  uint16
	CH6  uint16
	CH7  uint16
	CH8  uint16
	CH9  uint16
	CH10 uint16
	CH11 uint16
	CH12 uint16

	Btn1 bool // Button интерпретация, offset 0
	Btn2 bool // Button интерпретация, offset 1
	Btn3 bool // Button интерпретация, offset 2
	Btn4 bool // Button интерпретация, offset 3

	SW5 SwitchPos // CH5 интерпретирован как 3-поз переключатель
	SW6 SwitchPos // CH6 интерпретирован как 3-поз переключатель
	SW7 SwitchPos // CH7 интерпретирован как 3-поз переключатель
	SW8 SwitchPos // CH8 интерпретирован как 3-поз переключатель

	Raw []byte // сырые байты репорта
}

// TX12 управляет подключением и чтением данных с пульта RadioMaster TX12.
type TX12 struct {
	ctrl    *hidjoystick.Controller
	stateCh chan *State
	errCh   chan error
	stopCh  chan struct{}
}

// Open находит TX12 среди HID-устройств и открывает соединение.
func Open() (*TX12, error) {
	ctrl, err := hidjoystick.Open(Keywords)
	if err != nil {
		return nil, err
	}
	return newTX12(ctrl), nil
}

// WaitForDevice блокирует до появления TX12, проверяя каждые interval.
func WaitForDevice(interval time.Duration) (*TX12, error) {
	ctrl, err := hidjoystick.WaitForDevice(Keywords, interval)
	if err != nil {
		return nil, err
	}
	return newTX12(ctrl), nil
}

// IsAvailable проверяет, подключён ли TX12 прямо сейчас.
func IsAvailable() bool {
	return hidjoystick.IsAvailable(Keywords)
}

func newTX12(ctrl *hidjoystick.Controller) *TX12 {
	return &TX12{
		ctrl:    ctrl,
		stateCh: make(chan *State, 1),
		errCh:   make(chan error, 1),
		stopCh:  make(chan struct{}),
	}
}

// Info возвращает информацию об устройстве (имя, VID, PID).
func (t *TX12) Info() hidjoystick.Info {
	return t.ctrl.Info()
}

// Close останавливает чтение и закрывает соединение.
func (t *TX12) Close() {
	select {
	case <-t.stopCh:
	default:
		close(t.stopCh)
	}
	t.ctrl.Close()
}

// ReadOnce выполняет одно блокирующее чтение и возвращает State.
func (t *TX12) ReadOnce() (*State, error) {
	r, err := t.ctrl.ReadOnce()
	if err != nil {
		return nil, err
	}

	s, ok := parseReport(r)
	if !ok {
		return nil, nil
	}
	return s, nil
}

// Start запускает фоновое чтение.
// Состояния поступают в States(), ошибки — в Errors().
func (t *TX12) Start(readInterval time.Duration) {
	t.ctrl.Start(readInterval)
	go func() {
		for {
			select {
			case <-t.stopCh:
				return
			case r := <-t.ctrl.Reports():
				if state, ok := parseReport(r); ok {
					select {
					case t.stateCh <- state:
					default:
					}
				}
			case err := <-t.ctrl.Errors():
				select {
				case t.errCh <- err:
				default:
				}
			}
		}
	}()
}

// States возвращает канал состояний устройства.
func (t *TX12) States() <-chan *State {
	return t.stateCh
}

// Errors возвращает канал ошибок чтения.
func (t *TX12) Errors() <-chan error {
	return t.errCh
}

// Poll возвращает последнее актуальное состояние без блокировки.
// Удобно использовать в game loop.
func (t *TX12) Poll() (*State, bool) {
	var last *State
	var got bool
	for {
		select {
		case s := <-t.stateCh:
			last = s
			got = true
		default:
			return last, got
		}
	}
}

// ── Парсинг ───────────────────────────────────────────────────────────────────

func parseReport(r hidjoystick.Report) (*State, bool) {
	if r.Len() < minReportLen {
		return nil, false
	}

	raw := make([]byte, r.Len())
	copy(raw, r.Data)

	ch5 := r.U16LE(offCH5)
	ch6 := r.U16LE(offCH6)
	ch7 := r.U16LE(offCH7)
	ch8 := r.U16LE(offCH8)

	btn1 := r.BitU16(offButtons, bitBtn1)
	btn2 := r.BitU16(offButtons, bitBtn2)
	btn3 := r.BitU16(offButtons, bitBtn3)
	btn4 := r.BitU16(offButtons, bitBtn4)

	ch9 := uint16(ValMIN)
	if btn1 {
		ch9 = ValMAX
	}
	ch10 := uint16(ValMIN)
	if btn2 {
		ch10 = ValMAX
	}
	ch11 := uint16(ValMIN)
	if btn3 {
		ch11 = ValMAX
	}
	ch12 := uint16(ValMIN)
	if btn4 {
		ch12 = ValMAX
	}

	return &State{
		CH1:  r.U16LE(offCH1),
		CH2:  r.U16LE(offCH2),
		CH3:  r.U16LE(offCH3),
		CH4:  r.U16LE(offCH4),
		CH5:  ch5,
		CH6:  ch6,
		CH7:  ch7,
		CH8:  ch8,
		CH9:  ch9,
		CH10: ch10,
		CH11: ch11,
		CH12: ch12,

		Btn1: btn1,
		Btn2: btn2,
		Btn3: btn3,
		Btn4: btn4,

		SW5: valToSwitch(ch5),
		SW6: valToSwitch(ch6),
		SW7: valToSwitch(ch7),
		SW8: valToSwitch(ch8),

		Raw: raw,
	}, true
}

func valToSwitch(v uint16) SwitchPos {
	dUp := absDiff(v, ValMAX)
	dMid := absDiff(v, ValMID)
	dDown := absDiff(v, ValMIN)
	if dUp <= dMid && dUp <= dDown {
		return SwitchUp
	}
	if dMid <= dDown {
		return SwitchMid
	}
	return SwitchDown
}

func absDiff(a, b uint16) uint16 {
	if a > b {
		return a - b
	}
	return b - a
}
