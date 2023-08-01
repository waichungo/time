package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"systime/native"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
	user32          = syscall.MustLoadDLL("user32.dll")
	MUTEX           = "SysTimer"
)

type TimeInfo struct {
	Abbreviation string    `json:"abbreviation"`
	ClientIP     string    `json:"client_ip"`
	Datetime     time.Time `json:"datetime"`
	DayOfWeek    int       `json:"day_of_week"`
	DayOfYear    int       `json:"day_of_year"`
	Dst          bool      `json:"dst"`
	DstFrom      any       `json:"dst_from"`
	DstOffset    int       `json:"dst_offset"`
	DstUntil     any       `json:"dst_until"`
	RawOffset    int       `json:"raw_offset"`
	Timezone     string    `json:"timezone"`
	Unixtime     int       `json:"unixtime"`
	UtcDatetime  time.Time `json:"utc_datetime"`
	UtcOffset    string    `json:"utc_offset"`
	WeekNumber   int       `json:"week_number"`
}

func CheckErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
func LoadZonePairs() map[string]string {
	pairs := map[string]string{}
	data, err := exec.Command("tzutil.exe", "/l").Output()
	if err == nil {
		lines := strings.Split(string(data), "\n")

		for i := 1; i < len(lines); i += 3 {
			line := strings.TrimSpace(lines[i])
			key := strings.TrimSpace(lines[i-1])
			if len(line) > 0 && len(key) > 0 {
				pairs[key] = line
			}
		}
	}
	return pairs
}
func IsAdmin() bool {
	var sid *windows.SID

	// Although this looks scary, it is directly copied from the
	// official windows documentation. The Go API for this is a
	// direct wrap around the official C++ API.
	// See https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-checktokenmembership
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {

		return false
	}
	defer windows.FreeSid(sid)

	// This appears to cast a null pointer so I'm not sure why this
	// works, but this guy says it does and it Works for Me™:
	// https://github.com/golang/go/issues/28804#issuecomment-438838144
	token := windows.Token(0)

	admin, err := token.IsMember(sid)
	if err != nil {
		log.Fatalf("Token Membership Error: %s", err)
		return false
	}
	return admin
}
func CreateMutex(name string) (uintptr, error) {
	ret, _, err := procCreateMutex.Call(
		0,
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))),
	)
	switch int(err.(syscall.Errno)) {
	case 0:
		return ret, nil
	default:
		return ret, err
	}
}
func FirstInstance() bool {
	mutex, err := CreateMutex(MUTEX)
	if err != nil {
		syscall.CloseHandle(syscall.Handle(mutex))
	}
	return err == nil
}
func RerunElevated() {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)

	var showCmd int32 = 1 //SW_NORMAL

	err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		fmt.Println(err)
	}
}
func Elevate() bool {
	res := MessageBoxPlain("Permissions", "Do you need admin permissions")
	return res == 6
}
func MessageBox(hwnd uintptr, caption, title string, flags uint) int {
	ret, _, _ := user32.MustFindProc("MessageBoxW").Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(caption))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))),
		uintptr(flags))

	return int(ret)
}

// MessageBoxPlain of Win32 API.
func MessageBoxPlain(title, caption string) int {
	const (
		NULL     = 0
		MB_OK    = 0
		MB_YESNO = 4
	)
	return MessageBox(NULL, caption, title, MB_YESNO)
}
func main() {
	if IsAdmin() {
		MUTEX += "_admin"
	}
	if FirstInstance() {
		if !IsAdmin() {
			if Elevate() {
				RerunElevated()
				os.Exit(0)
			}
		}
		data, err := GetData("http://worldtimeapi.org/api/ip.json")
		CheckErr(err)

		info := TimeInfo{}
		err = json.Unmarshal(data, &info)
		//CheckErr(err)

		err = SetTimeZone(info)
		CheckErr(err)

		err = native.SetSystemDate(info.Datetime)
		CheckErr(err)
	} else {
		fmt.Println("Another instance is running")
	}

}
func SetTimeZone(tinfo TimeInfo) error {
	zone := ""
	for key, val := range WinIANA {
		if val == tinfo.Timezone {
			zone = key
			break
		}
	}
	if zone == "" {
		return errors.New("zone not found")
	}
	zones := LoadZonePairs()
	winZone := zones[zone]

	arg := fmt.Sprintf("%s", winZone)
	cmd := exec.Command("tzutil.exe", "/s", arg)
	res, err := cmd.Output()
	fmt.Println(string(res))
	return err
}
func GetData(address string) ([]byte, error) {

	res, err := http.Get(address)
	errCount := 0
	for err != nil {
		errCount++

		res, err = http.Get(address)
		if errCount > 5 {
			return []byte{}, err
		}
		time.Sleep(1500 * time.Millisecond)
	}
	defer res.Body.Close()
	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		return []byte{}, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	htmlbytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, err
	}
	return htmlbytes, nil
}
