package util

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"unicode"

	"github.com/bytedance/sonic"
	"github.com/gocarina/gocsv"
	"github.com/rs/zerolog/log"
)

// In slice
func In[T string | int](mem T, arr []T) bool {
	for i := range arr {
		if mem == arr[i] {
			return true
		}
	}
	return false
}

// expressions
func Exp[T string | int | float64](isTrue bool, yes T, no T) T {
	if isTrue {
		return yes
	} else {
		return no
	}
}

// http get and read
func GetAndRead(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// tushare api
func TushareApi(apiName string, params any, fields any, val any) error {
	// set params
	req := map[string]any{
		"api_name": apiName,
		"token":    "8dbaa93be7f8d09210ca9cb0843054417e2820203201c0f3f7643410",
	}
	if params != nil {
		req["params"] = params
	}
	if fields != nil {
		req["fields"] = fields
	}
	param, _ := sonic.Marshal(req)

	// post request
	res, err := http.Post("https://api.tushare.pro", "application/json", bytes.NewReader(param))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var data struct {
		Data struct {
			Head  []string   `json:"fields"`
			Items [][]string `json:"items"`
		} `json:"data"`
		Msg string `json:"msg"`
	}

	if err = sonic.Unmarshal(body, &data); err != nil {
		return err
	}
	if data.Msg != "" {
		log.Warn().Msg(err.Error())
		return errors.New(data.Msg)
	}

	// read csv data
	var src strings.Builder
	src.WriteString(strings.Join(data.Data.Head, ","))

	for _, i := range data.Data.Items {
		// valid
		t := strings.Join(i, "")
		if strings.Contains(t, ",") || strings.Contains(t, "\"") {
			continue
		}
		// write
		src.WriteByte('\n')
		src.WriteString(strings.Join(i, ","))
	}

	return gocsv.Unmarshal(strings.NewReader(src.String()), val)
}

func Md5Code(code string) string {
	sig := md5.Sum([]byte(code))
	sum := byte(0)
	for _, c := range sig {
		sum += c
	}
	return fmt.Sprintf("%d", sum%128)
}

func IsChinese(str string) bool {
	for i, r := range str {
		// only check first character
		if unicode.Is(unicode.Han, r) {
			return true
		}
		if i > 0 {
			break
		}
	}
	return false
}
