# hid-joystick

hidjoystick - Go-пакет для чтения сырых HID-репортов от любого джойстика, подключённого по USB.  
Не привязан к конкретной модели устройства — всю интерпретацию байтов вы делаете сами.

tx12 - Go-пакет конкретная реализация для RadioMaster TX12 на базе hidjoystick
## Структура проекта

```
hid-joystick/
├── hidjoystick/          # пакет: Windows HID API, Report, Controller
│   ├── hid.go
│   ├── report.go
│   └── controller.go
├── tx12/                 # пакет: Работа с RadioMaster TX12
│   └── tx12.go
├── examples/
│   └── tx12/             # примеры: RadioMaster TX12
│       ├── tx12_monitor.go
│       └── tx12_simple.go
├── go.mod
└── README.md
```

## Требования

- Go 1.21+
- Windows 10 / 11
- Джойстик в режиме **HID Joystick** (для TX12: *System Settings → USB Mode → HID Joystick*)

## Быстрый старт

```
cd examples\tx12
go run .\tx12_simple.go
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

## Пример: Монитор RadioMaster TX12

```
cd examples\tx12
go run .\tx12_monitor.go
```