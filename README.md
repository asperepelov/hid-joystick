# hid-joystick проект

#### <big>hidjoystick</big> - Go-пакет для чтения сырых HID-репортов от любого джойстика, подключённого по USB.  
Не привязан к конкретной модели устройства — всю интерпретацию байтов вы делаете сами.

#### <big>tx12</big> - Go-пакет конкретная реализация для RadioMaster TX12 на базе hidjoystick
Каналы:
- CH1 - roll
- CH2 - pitch
- CH3 - throttle
- CH4 - YAW
- CH5 - SE переключатель
- CH6 - SF переключатель
- CH7 - SB переключатель
- CH8 - SC переключатель
- CH9 - SA кнопка
- CH10 - SD кнопка
- CH11 - S1 слайдер
- CH12 - S2 слайдер

## Структура проекта

```
hid-joystick/
├── hidjoystick/                # пакет: Windows HID API, Report, Controller
│   ├── hid.go
│   ├── report.go
│   └── controller.go
├── tx12/                       # пакет: Работа с RadioMaster TX12
│   └── tx12.go
├── examples/
│   ├── tx12/                   # примеры
│   │   ├── tx12_monitor.go     # работа с tx12, монитор RadioMaster TX12
│   │   └── tx12_simple.go      # работа с tx12, простой пример
│   └── gamepad.go              # работа с hidjoystick чтение USB Gamepad
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

## Пример: Чтение USB Gamepad

```
cd examples
go run .\gamepad.go
```