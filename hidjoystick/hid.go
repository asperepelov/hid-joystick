// Package hidjoystick предоставляет интерфейс для чтения сырых HID-репортов
// от любого джойстика, подключённого по USB как HID-устройство на Windows.
package hidjoystick

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	hid                             = windows.NewLazySystemDLL("hid.dll")
	setupapi                        = windows.NewLazySystemDLL("setupapi.dll")
	hidD_GetHidGuid                 = hid.NewProc("HidD_GetHidGuid")
	hidD_GetAttributes              = hid.NewProc("HidD_GetAttributes")
	hidD_GetProductString           = hid.NewProc("HidD_GetProductString")
	setupDiGetClassDevs             = setupapi.NewProc("SetupDiGetClassDevsW")
	setupDiEnumDeviceInterfaces     = setupapi.NewProc("SetupDiEnumDeviceInterfaces")
	setupDiGetDeviceInterfaceDetail = setupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	setupDiDestroyDeviceInfoList    = setupapi.NewProc("SetupDiDestroyDeviceInfoList")
)

const (
	digcfPresent         = 0x02
	digcfDeviceInterface = 0x10
	invalidHandleValue   = ^uintptr(0)
	fileShareRead        = 0x00000001
	fileShareWrite       = 0x00000002
	openExisting         = 3
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type spDeviceInterfaceData struct {
	CbSize             uint32
	InterfaceClassGuid guid
	Flags              uint32
	Reserved           uintptr
}

type hiddAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

func getHidGuid() guid {
	var g guid
	hidD_GetHidGuid.Call(uintptr(unsafe.Pointer(&g)))
	return g
}

func containsStr(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// openDevice перебирает HID-устройства и открывает первое,
// имя которого содержит одно из ключевых слов.
func openDevice(keywords []string) (windows.Handle, string, uint16, uint16, error) {
	g := getHidGuid()

	hDevInfo, _, _ := setupDiGetClassDevs.Call(
		uintptr(unsafe.Pointer(&g)), 0, 0,
		digcfPresent|digcfDeviceInterface,
	)
	if hDevInfo == invalidHandleValue {
		return 0, "", 0, 0, fmt.Errorf("SetupDiGetClassDevs failed")
	}
	defer setupDiDestroyDeviceInfoList.Call(hDevInfo)

	for i := 0; ; i++ {
		var ifaceData spDeviceInterfaceData
		ifaceData.CbSize = uint32(unsafe.Sizeof(ifaceData))

		ret, _, _ := setupDiEnumDeviceInterfaces.Call(
			hDevInfo, 0, uintptr(unsafe.Pointer(&g)),
			uintptr(i), uintptr(unsafe.Pointer(&ifaceData)),
		)
		if ret == 0 {
			break
		}

		var requiredSize uint32
		setupDiGetDeviceInterfaceDetail.Call(
			hDevInfo, uintptr(unsafe.Pointer(&ifaceData)),
			0, 0, uintptr(unsafe.Pointer(&requiredSize)), 0,
		)

		buf := make([]byte, requiredSize)
		cbSize := uint32(6)
		if unsafe.Sizeof(uintptr(0)) == 8 {
			cbSize = 8
		}
		*(*uint32)(unsafe.Pointer(&buf[0])) = cbSize

		ret, _, _ = setupDiGetDeviceInterfaceDetail.Call(
			hDevInfo, uintptr(unsafe.Pointer(&ifaceData)),
			uintptr(unsafe.Pointer(&buf[0])), uintptr(requiredSize),
			uintptr(unsafe.Pointer(&requiredSize)), 0,
		)
		if ret == 0 {
			continue
		}

		pathU16 := make([]uint16, (requiredSize-4)/2)
		for j := range pathU16 {
			pathU16[j] = *(*uint16)(unsafe.Pointer(&buf[4+j*2]))
		}
		path := windows.UTF16ToString(pathU16)

		h, err := windows.CreateFile(
			windows.StringToUTF16Ptr(path),
			windows.GENERIC_READ,
			fileShareRead|fileShareWrite,
			nil, openExisting, 0, 0,
		)
		if err != nil {
			continue
		}

		var attr hiddAttributes
		attr.Size = uint32(unsafe.Sizeof(attr))
		hidD_GetAttributes.Call(uintptr(h), uintptr(unsafe.Pointer(&attr)))

		var productBuf [256]uint16
		hidD_GetProductString.Call(uintptr(h), uintptr(unsafe.Pointer(&productBuf[0])), 256)
		name := windows.UTF16ToString(productBuf[:])

		for _, kw := range keywords {
			if containsStr(name, kw) {
				return h, name, attr.VendorID, attr.ProductID, nil
			}
		}
		windows.CloseHandle(h)
	}
	return 0, "", 0, 0, fmt.Errorf("device not found (keywords: %v)", keywords)
}
