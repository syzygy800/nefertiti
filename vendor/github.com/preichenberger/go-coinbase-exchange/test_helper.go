package coinbase

import (
	"errors"
	"fmt"
	"os"
	"reflect"
)

func NewTestClient() *Client {
	return NewClient(
		os.Getenv("TEST_COINBASE_SECRET"),
		os.Getenv("TEST_COINBASE_KEY"),
		os.Getenv("TEST_COINBASE_PASSPHRASE"), true)
}

func StructHasZeroValues(i interface{}) bool {
	iv := reflect.ValueOf(i)

	//values := make([]interface{}, v.NumField())

	for i := 0; i < iv.NumField(); i++ {
		field := iv.Field(i)
		if reflect.Zero(field.Type()) == field {
			return true
		}
	}

	return false
}

func CompareProperties(a, b interface{}, properties []string) (bool, error) {
	aValueOf := reflect.ValueOf(a)
	bValueOf := reflect.ValueOf(b)

	for _, property := range properties {
		aValue := reflect.Indirect(aValueOf).FieldByName(property).Interface()
		bValue := reflect.Indirect(bValueOf).FieldByName(property).Interface()

		if aValue != bValue {
			return false, errors.New(fmt.Sprintf("%s not equal", property))
		}
	}

	return true, nil
}
