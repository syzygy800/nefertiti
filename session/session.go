package session

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

const (
	RFC3339Milli = "2006-01-02T15:04:05.999Z07:00"
)

func GetSessionDir() string {
	tmp := os.TempDir()
	dir := filepath.Join(tmp, "com.cryptotrader.session")
	os.MkdirAll(dir, os.ModePerm)
	return dir
}

func GetSessionFile(name string) string {
	return filepath.Join(GetSessionDir(), name)
}

func GetTempFileName(name, ext string) string {
	return filepath.Join(GetSessionDir(), (name + ext))
}

func GetLastRequest(exchange string) (*time.Time, error) {
	data, err := ioutil.ReadFile(GetSessionFile(exchange))
	if err == nil {
		for len(data) > 0 {
			out, err := time.Parse(RFC3339Milli, string(data))
			if err == nil {
				return &out, nil
			} else {
				if string(data[len(data)-1]) == "0" {
					data = data[:len(data)-1]
				} else {
					return nil, err
				}
			}
		}
	}
	return nil, nil
}

func SetLastRequest(exchange string, value time.Time) error {
	var (
		err error
		str string
	)
	str = GetSessionFile(exchange)
	if _, err = os.Stat(str); err == nil {
		if err = os.Truncate(str, 0); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(str, []byte(value.Format(RFC3339Milli)), 0600)
}
