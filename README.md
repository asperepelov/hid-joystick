# hid-joystick

Go-пакет для чтения сырых HID-репортов от любого джойстика, подключённого по USB.  
Не привязан к конкретной модели устройства — всю интерпретацию байтов вы делаете сами.

## Структура проекта

```
hid-joystick/
├── hidjoystick/          # пакет: Windows HID API, Report, Controller
│   ├── hid.go
│   ├── report.go
│   └── controller.go
├── examples/
│   └── tx12/             # пример: RadioMaster TX12
│       └── main.go
├── go.mod
└── README.md
```

## Требования

- Go 1.21+
- Windows 10 / 11
- Джойстик в режиме **HID Joystick** (для TX12: *System Settings → USB Mode → HID Joystick*)

## Быстрый старт

```go
import "github.com/yourname/hid-joystick/hidjoystick"

keywords := []string{"RadioMaster", "TX12"}

ctrl, err := hidjoystick.WaitForDevice(keywords, time.Second)
if err != nil {
    log.Fatal(err)
}
defer ctrl.Close()

ctrl.Start()

for r := range ctrl.Reports() {
    v := r.U16LE(4) // прочитать uint16 по офсету 4
    fmt.Println(v)
}
```

## API пакета `hidjoystick`

### Открытие устройства

```go
hidjoystick.IsAvailable(keywords []string) bool
hidjoystick.Open(keywords []string) (*Controller, error)
hidjoystick.WaitForDevice(keywords []string, interval time.Duration) (*Controller, error)
```

### Controller

```go
ctrl.Start()                    // запустить фоновое чтение
ctrl.Reports() <-chan Report    // канал репортов
ctrl.Errors()  <-chan error     // канал ошибок
ctrl.Poll() (Report, bool)      // последний репорт без блокировки (для game loop)
ctrl.ReadOnce() (Report, error) // одно блокирующее чтение
ctrl.Info() Info                // имя, VID, PID устройства
ctrl.Close()                    // закрыть соединение
```

### Report

```go
r.Len()                    int    // длина репорта
r.Byte(offset)             byte
r.U16LE(offset)            uint16 // little-endian
r.U16BE(offset)            uint16 // big-endian
r.Bit(offset, bit)         bool   // бит в байте
r.BitU16(offset, bit)      bool   // бит в uint16 LE
```

## Пример: RadioMaster TX12

```
cd examples/tx12
go run .
```

Смотрите `examples/tx12/main.go` — там TX12-специфичный маппинг байтов,
структура `JoystickState` и отображение всех каналов в терминале.
