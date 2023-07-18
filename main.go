package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"time"
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
func main() {
	data, err := GetData("http://worldtimeapi.org/api/ip.json")
	CheckErr(err)
	info := TimeInfo{}
	err = json.Unmarshal(data, &info)
	CheckErr(err)

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
func SetSystemDate(newTime time.Time) error {
	_, lookErr := exec.LookPath("date")
	if lookErr != nil {
		fmt.Printf("Date binary not found, cannot set system date: %s\n", lookErr.Error())
		return lookErr
	} else {
		//dateString := newTime.Format("2006-01-2 15:4:5")
		dateString := newTime.Format("2 Jan 2006 15:04:05")
		fmt.Printf("Setting system date to: %s\n", dateString)
		args := []string{"--set", dateString}
		return exec.Command("date", args...).Run()
	}
}
