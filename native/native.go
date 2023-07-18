package native

//#include <Windows.h>
import "C"
import (
	"errors"
	"time"
)

func SetSystemDate(newTime time.Time) error {
	date := newTime.UTC()
	updatedTime := C.SYSTEMTIME{}
	updatedTime.wYear = C.ushort(date.Year())
	updatedTime.wMonth = C.ushort(date.Month())
	updatedTime.wDay = C.ushort(date.Day())
	updatedTime.wHour = C.ushort(date.Hour())
	updatedTime.wMinute = C.ushort(date.Minute())
	updatedTime.wSecond = C.ushort(date.Second())
	res := C.SetSystemTime(&updatedTime)
	if res != 1 {
		return errors.New("failed to set system time")
	}

	return nil
}
