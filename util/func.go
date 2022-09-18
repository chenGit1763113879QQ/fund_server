package util

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
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
	TYPE_IDS
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
		log.Error().Msg(err.Error())
		return nil, err
	}
	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)
	return body, nil
}

// get xueqiu api
func XueQiuAPI(url string) ([]byte, error) {
	// add token
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Add("cookie", "xq_a_token=80b283f898285a9e82e2e80cf77e5a4051435344")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Msg(err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	return body, nil
}

// tushare api
func TushareApi(api string, params any, fields any, val any) error {
	// set params
	req := map[string]any{
		"api_name": api,
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
		log.Error().Msg(err.Error())
		return err
	}
	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	var data struct {
		Data struct {
			Head  []string   `json:"fields"`
			Items [][]string `json:"items"`
		} `json:"data"`
		Msg string `json:"msg"`
	}

	if err = UnmarshalJSON(body, &data); err != nil {
		return err
	}
	if data.Msg != "" {
		log.Warn().Msgf("tushare err msg: %s", data.Msg)
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
	return hex.EncodeToString(m.Sum(nil))[0:2]
}

func IsChinese(str string) bool {
	if len(str) == 0 {
		return false
	}
	// only check first character
	r := rune(str[0])
	return unicode.Is(unicode.Han, r)
}

func UnmarshalJSON(body []byte, data any, path ...interface{}) error {
	node, err := sonic.Get(body, path...)
	if err != nil {
		log.Warn().Msgf("unmarshal get node err: %s", err.Error())
		return err
	}
	raw, err := node.Raw()
	if err != nil {
		log.Warn().Msgf("unmarshal get raw err: %s", err.Error())
		return err
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

// go func for every duration
func GoJob(f func(), duration time.Duration, delay ...time.Duration) {
	go func() {
		// delay
		if len(delay) > 0 {
			time.Sleep(delay[0])
		}
		// go func
		for {
			f()
			time.Sleep(duration)
		}
	}()
}
