package util

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"unicode"

	"github.com/bytedance/sonic"
	"github.com/gocarina/gocsv"
	"github.com/rs/zerolog/log"
)

const (
	MARKET_CN = iota + 1
	MARKET_HK
	MARKET_US

	TYPE_STOCK
	TYPE_INDEX
	TYPE_FUND
	TYPE_I1
	TYPE_I2
	TYPE_C
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
	m := md5.New()
	m.Write([]byte(code))
	val := hex.EncodeToString(m.Sum(nil))
	return fmt.Sprintf("%X%c", val[0]%8, val[1])
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

func UnmarshalJSON(body []byte, data any, path ...interface{}) error {
	node, err := sonic.Get(body, path...)
	if err != nil {
		log.Warn().Msg(err.Error())
	}
	raw, err := node.Raw()
	if err != nil {
		log.Warn().Msg(err.Error())
	}
	return sonic.UnmarshalString(raw, &data)
}

func Mean[T int | int64 | float64](arr []T) float64 {
	var sum T
	for i := range arr {
		sum += arr[i]
	}
	return float64(sum) / float64(len(arr))
}
