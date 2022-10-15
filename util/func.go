package util

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/bytedance/sonic"
	jsoniter "github.com/json-iterator/go"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// In slice
func In[T string | int](n T, arr []T) bool {
	for i := range arr {
		if n == arr[i] {
			return true
		}
	}
	return false
}

// Expressions
func Exp[T string | int | float64](isTrue bool, yes T, no T) T {
	if isTrue {
		return yes
	} else {
		return no
	}
}

// Http get and read
func GetAndRead(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		log.Err(err)
		return nil, err
	}
	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)
	return body, nil
}

// XueQiu api
func XueQiuAPI(url string) ([]byte, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Add("cookie", viper.GetString("xq_token"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Err(err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	return body, nil
}

// TuShare api
func TushareApi(api string, params any, fields any, data any) error {
	// set params
	req := map[string]any{
		"api_name": api,
		"token":    viper.GetString("ts_token"),
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

	body, _ := ioutil.ReadAll(res.Body)

	var src struct {
		Data struct {
			Head  []string `json:"fields"`
			Items [][]any  `json:"items"`
		} `json:"data"`
		Msg string `json:"msg"`
	}

	if err = UnmarshalJSON(body, &src); err != nil {
		return err
	}
	if src.Msg != "" {
		log.Warn().Msgf("tushare msg: %s", src.Msg)
		return errors.New(src.Msg)
	}

	return DecodeJSONItems(src.Data.Head, src.Data.Items, &data)
}

func DecodeJSONItems(columns []string, items [][]any, data any) error {
	// decode map
	srcMap := make([]map[string]any, len(items))
	for i, item := range items {
		srcMap[i] = map[string]any{}

		for c, col := range columns {
			srcMap[i][col] = item[c]
		}
	}
	return mapstructure.Decode(srcMap, &data)
}

func Md5Code(code string) string {
	m := md5.New()
	m.Write([]byte(code))
	val := hex.EncodeToString(m.Sum(nil))
	return val[0:2]
}

// Check str is Chinese
func IsChinese(str string) bool {
	for _, r := range str {
		return unicode.Is(unicode.Han, r)
	}
	return false
}

// Mean 均值
func Mean[T int | int64 | float64](arr []T) float64 {
	var sum T
	for _, i := range arr {
		sum += i
	}
	return float64(sum) / float64(len(arr))
}

// Sum 求和
func Sum(arr ...float64) float64 {
	var sum float64
	for _, i := range arr {
		sum += i
	}
	return sum
}

// Go Job for every duration
func GoJob(f func(), duration time.Duration, delay ...time.Duration) {
	go func() {
		for _, dl := range delay {
			time.Sleep(dl)
		}
		for {
			f()
			time.Sleep(duration)
		}
	}()
}

// Unmarshal JSON
func UnmarshalJSON(body []byte, data any, path ...any) error {
	node := jsoniter.Get(body, path...)
	return sonic.UnmarshalString(node.ToString(), &data)
}

// ParseCode exp: 000001.SH 00700.HK AAPL.US
func ParseCode(code string) string {
	pre, suf, ok := strings.Cut(code, ".")
	if ok {
		// .DJI
		if pre == "" && suf != "" {
			return suf + ".US"
		}
		switch suf {
		// 600519.SS
		case "SS":
			return pre + ".SH"
		// normal
		case "SH", "SZ", "HK", "US":
			return code
		}
	}
	// US
	if len(pre) == 1 && unicode.IsLetter(rune(pre[0])) {
		return pre + ".US"
	}
	// CN
	if pre[0:2] == "SZ" || pre[0:2] == "SH" {
		return fmt.Sprintf("%s.%s", pre[2:], pre[0:2])
	}
	// HK
	if (pre[0] == '0' && len(pre) == 5) || pre[0:2] == "HK" {
		return pre + ".HK"
	}
	// US
	return pre + ".US"
}
