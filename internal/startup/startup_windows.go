// Package startup registers/unregisters SparkEnhance in the Windows
// "Run" key so the app starts automatically at user login.
//
// Uses HKCU\Software\Microsoft\Windows\CurrentVersion\Run so no admin
// rights are required.
package startup

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const appName = "SparkEnhance"

var (
	advapi32                  = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyExW         = advapi32.NewProc("RegOpenKeyExW")
	procRegCloseKey           = advapi32.NewProc("RegCloseKey")
	procRegSetValueExW        = advapi32.NewProc("RegSetValueExW")
	procRegQueryValueExW      = advapi32.NewProc("RegQueryValueExW")
	procRegDeleteValueW       = advapi32.NewProc("RegDeleteValueW")
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleFileNameW    = kernel32.NewProc("GetModuleFileNameW")
)

const (
	hkeyCurrentUser = 0x80000001
	keyQueryValue   = 0x0001
	keySetValue     = 0x0002
	keyRead         = 0x20019
	regSz           = 1
)

// Enable adds SparkEnhance to the Run key.
func Enable() error {
	exePath, err := exePath()
	if err != nil {
		return err
	}
	key, err := openRunKey(keySetValue | keyQueryValue)
	if err != nil {
		return err
	}
	defer procRegCloseKey.Call(key)

	value := syscall.StringToUTF16(`"` + exePath + `"`)
	return regSetValue(key, appName, &value[0])
}

// Disable removes SparkEnhance from the Run key.
func Disable() error {
	key, err := openRunKey(keySetValue | keyQueryValue)
	if err != nil {
		return err
	}
	defer procRegCloseKey.Call(key)
	namePtr, _ := syscall.UTF16PtrFromString(appName)
	_, _, _ = procRegDeleteValueW.Call(key, uintptr(unsafe.Pointer(namePtr)))
	return nil
}

// IsEnabled reports whether the Run entry exists.
func IsEnabled() bool {
	key, err := openRunKey(keyRead)
	if err != nil {
		return false
	}
	defer procRegCloseKey.Call(key)
	namePtr, _ := syscall.UTF16PtrFromString(appName)
	var buf [1024]uint16
	var bufLen uint32 = 1024 * 2
	_, _, _ = procRegQueryValueExW.Call(
		key, uintptr(unsafe.Pointer(namePtr)), 0, 0,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bufLen)),
	)
	return bufLen > 0 && bufLen <= 2048
}

func exePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return "", err
	}
	return abs, nil
}

func openRunKey(sam uint32) (uintptr, error) {
	var hkey uintptr
	subkey, _ := syscall.UTF16PtrFromString(runKey)
	r, _, _ := procRegOpenKeyExW.Call(
		uintptr(hkeyCurrentUser),
		uintptr(unsafe.Pointer(subkey)),
		0, uintptr(sam),
		uintptr(unsafe.Pointer(&hkey)),
	)
	if r != 0 {
		return 0, syscall.Errno(r)
	}
	return hkey, nil
}

func regSetValue(key uintptr, name string, pvalue *uint16) error {
	namePtr, _ := syscall.UTF16PtrFromString(name)
	// Compute byte length including null terminator.
	var cch uint32
	for p := pvalue; *p != 0; p = (*uint16)(unsafe.Add(unsafe.Pointer(p), 2)) {
		cch++
	}
	cch++ // null
	bytes := cch * 2

	r, _, _ := procRegSetValueExW.Call(
		key,
		uintptr(unsafe.Pointer(namePtr)),
		0, uintptr(regSz),
		uintptr(unsafe.Pointer(pvalue)), uintptr(bytes),
	)
	if r != 0 {
		return syscall.Errno(r)
	}
	return nil
}
