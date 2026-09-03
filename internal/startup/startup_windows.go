// Package startup registers SparkEnhance in:
//   - HKCU\Software\Microsoft\Windows\CurrentVersion\Run  (auto-launch at login)
//   - %APPDATA%\Microsoft\Windows\Start Menu\Programs\  (Start Menu shortcut)
//
// All operations use HKCU (no admin rights required).
package startup

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	runKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	appName   = "SparkEnhance"
	startMenu = "Microsoft\\Windows\\Start Menu\\Programs"
)

var (
	advapi32               = syscall.NewLazyDLL("advapi32.dll")
	ole32                  = syscall.NewLazyDLL("ole32.dll")
	shell32                = syscall.NewLazyDLL("shell32.dll")

	procRegOpenKeyExW     = advapi32.NewProc("RegOpenKeyExW")
	procRegCloseKey       = advapi32.NewProc("RegCloseKey")
	procRegSetValueExW    = advapi32.NewProc("RegSetValueExW")
	procRegQueryValueExW  = advapi32.NewProc("RegQueryValueExW")
	procRegDeleteValueW   = advapi32.NewProc("RegDeleteValueW")

	procCoInitialize     = ole32.NewProc("CoInitialize")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")
	procCoUninitialize   = ole32.NewProc("CoUninitialize")

	procSHGetFolderPathW = shell32.NewProc("SHGetFolderPathW")

	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleFileNameW = kernel32.NewProc("GetModuleFileNameW")
)

const (
	hkeyCurrentUser  = 0x80000001
	keyQueryValue    = 0x0001
	keySetValue      = 0x0002
	keyRead          = 0x20019
	regSz            = 1
	csidlPrograms    = 0x0002 // Start Menu\Programs
)

// CLSID_ShellLink = {00021401-0000-0000-C000-000000000046}
var clsidShellLink = syscall.GUID{
	Data1: 0x00021401, Data2: 0x0000, Data3: 0x0000,
	Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
}

// IID_IShellLinkW = {000214F9-0000-0000-C000-000000000046}
var iidIShellLinkW = syscall.GUID{
	Data1: 0x000214F9, Data2: 0x0000, Data3: 0x0000,
	Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
}

// IID_IPersistFile = {0000010B-0000-0000-C000-000000000046}
var iidIPersistFile = syscall.GUID{
	Data1: 0x0000010B, Data2: 0x0000, Data3: 0x0000,
	Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
}

// Enable registers SparkEnhance in Run + creates a Start Menu shortcut.
func Enable() error {
	if err := enableRunKey(); err != nil {
		return fmt.Errorf("run key: %w", err)
	}
	if err := enableStartMenu(); err != nil {
		return fmt.Errorf("start menu: %w", err)
	}
	return nil
}

// Disable removes both the Run key and the Start Menu shortcut.
func Disable() error {
	_ = disableRunKey()
	if err := disableStartMenu(); err != nil {
		return fmt.Errorf("start menu: %w", err)
	}
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

// StartMenuShortcutPath returns the full path to the .lnk file in Start Menu.
func StartMenuShortcutPath() (string, error) {
	dir, err := programsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appName+".lnk"), nil
}

// ---- Run key ----

func enableRunKey() error {
	exe, err := exePath()
	if err != nil {
		return err
	}
	key, err := openRunKey(keySetValue | keyQueryValue)
	if err != nil {
		return err
	}
	defer procRegCloseKey.Call(key)

	value := syscall.StringToUTF16(`"` + exe + `"`)
	return regSetValue(key, appName, &value[0])
}

func disableRunKey() error {
	key, err := openRunKey(keySetValue | keyQueryValue)
	if err != nil {
		return err
	}
	defer procRegCloseKey.Call(key)
	namePtr, _ := syscall.UTF16PtrFromString(appName)
	_, _, _ = procRegDeleteValueW.Call(key, uintptr(unsafe.Pointer(namePtr)))
	return nil
}

// ---- Start Menu shortcut ----

func enableStartMenu() error {
	shortcutPath, err := StartMenuShortcutPath()
	if err != nil {
		return err
	}

	// Make sure the Programs directory exists.
	if err := os.MkdirAll(filepath.Dir(shortcutPath), 0700); err != nil {
		return err
	}

	exe, err := exePath()
	if err != nil {
		return err
	}
	return createShortcut(shortcutPath, exe, "")
}

func disableStartMenu() error {
	path, err := StartMenuShortcutPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func programsDir() (string, error) {
	// Use the modern known folder: FOLDERID_Programs = {A3918781-E5F2-4890-B3D9-A7E54332328C}
	// But SHGetFolderPathW with CSIDL_PROGRAMS is the simpler path.
	var buf [260]uint16
	_, _, _ = procSHGetFolderPathW.Call(
		0, // hwnd
		uintptr(csidlPrograms),
		0, // hToken
		0, // dwFlags
		uintptr(unsafe.Pointer(&buf[0])),
	)
	return syscall.UTF16ToString(buf[:]), nil
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

// ---- IShellLinkW COM ----

// IShellLinkW v-table layout (offsets in bytes):
//   0: QueryInterface
//   8: AddRef
//  16: Release
//  24: GetPath
//  32: GetIDList
//  40: SetIDList
//  48: GetDescription
//  56: SetDescription
//  64: GetArguments
//  72: SetArguments
//  80: GetWorkingDirectory
//  88: SetWorkingDirectory
//  96: GetHotkey
// 104: SetHotkey
// 112: GetShowCmd
// 120: SetShowCmd
// 128: GetIconLocation
// 136: SetIconLocation
// 144: SetRelativePath
// 152: Resolve
// 160: SetPath

// IPersistFile v-table:
// 0: QueryInterface
// 8: AddRef
// 16: Release
// 24: GetClassID
// 32: IsDirty
// 40: Load
// 48: Save
// 56: SaveCompleted
// 64: GetCurFile

const (
	clsctxInprocServer = 0x1
)

func createShortcut(lnkPath, targetPath, _ string) error {
	// CoInitialize
	hr, _, _ := procCoInitialize.Call(0)
	if hr != 0 && hr != 0x00000001 /* S_FALSE */ {
		return fmt.Errorf("CoInitialize: 0x%x", hr)
	}
	defer procCoUninitialize.Call()

	// Create IShellLinkW instance.
	var shellLink uintptr
	hr, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidShellLink)),
		0,
		uintptr(clsctxInprocServer),
		uintptr(unsafe.Pointer(&iidIShellLinkW)),
		uintptr(unsafe.Pointer(&shellLink)),
	)
	if hr != 0 {
		return fmt.Errorf("CoCreateInstance(IShellLinkW): 0x%x", hr)
	}
	defer release(shellLink)

	// Set path (vtable[20] = SetPath at offset 20*8 = 160)
	pathPtr, _ := syscall.UTF16PtrFromString(targetPath)
	hr = comCall(shellLink, 20, 1, uintptr(unsafe.Pointer(pathPtr)))
	if hresult(hr) != 0 {
		return fmt.Errorf("IShellLinkW::SetPath: 0x%x", hr)
	}

	// Set icon location to the exe itself, index 0 (vtable[17])
	exePtr, _ := syscall.UTF16PtrFromString(targetPath)
	hr = comCall(shellLink, 17, 2, uintptr(unsafe.Pointer(exePtr)), 0)
	if hresult(hr) != 0 {
		// Non-fatal: shortcut still works without icon
		fmt.Printf("warning: SetIconLocation: 0x%x\n", hr)
	}

	// Set working directory to the exe's folder.
	wd := filepath.Dir(targetPath)
	wdPtr, _ := syscall.UTF16PtrFromString(wd)
	hr = comCall(shellLink, 11, 1, uintptr(unsafe.Pointer(wdPtr)))
	if hresult(hr) != 0 {
		fmt.Printf("warning: SetWorkingDirectory: 0x%x\n", hr)
	}

	// QueryInterface for IPersistFile (offset 0 in IShellLinkW vtable)
	var persistFile uintptr
	hr = comCall(shellLink, 0, 2,
		uintptr(unsafe.Pointer(&iidIPersistFile)),
		uintptr(unsafe.Pointer(&persistFile)),
	)
	if hresult(hr) != 0 {
		return fmt.Errorf("QueryInterface(IPersistFile): 0x%x", hr)
	}
	defer release(persistFile)

	// IPersistFile::Save (vtable[6], offset 6*8 = 48)
	lnkPtr, _ := syscall.UTF16PtrFromString(lnkPath)
	hr = comCall(persistFile, 6, 2,
		uintptr(unsafe.Pointer(lnkPtr)),
		1, // fRemember = TRUE
	)
	if hresult(hr) != 0 {
		return fmt.Errorf("IPersistFile::Save: 0x%x", hr)
	}

	return nil
}

// comCall invokes a COM vtable method.
//
// layout: instance, vtable index (0-based), arg count, args...
// args are uintptr (caller is responsible for pointer marshaling).
// Returns only the first return value (HRESULT on this architecture).
func comCall(instance uintptr, vtableIdx int, argc int, args ...uintptr) uintptr {
	// Read vtable pointer from instance[0].
	vtable := *(*uintptr)(unsafe.Pointer(instance))
	// Method pointer is vtable[vtableIdx].
	method := *(*uintptr)(unsafe.Add(unsafe.Pointer(vtable), uintptr(vtableIdx)*unsafe.Sizeof(uintptr(0))))
	r1, _, _ := syscall.SyscallN(method, append([]uintptr{instance}, args...)...)
	return r1
}

// hresult extracts the HRESULT (S_OK = 0) from a COM call result.
func hresult(r uintptr) uint32 {
	return uint32(r)
}

// release calls IUnknown::Release on the COM object.
func release(obj uintptr) {
	if obj == 0 {
		return
	}
	comCall(obj, 2, 0) // IUnknown::Release is vtable[2]
}
