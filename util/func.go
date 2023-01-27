package util

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
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

// Http get and read
func GetAndRead(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		log.Err(err)
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
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

	body, _ := io.ReadAll(resp.Body)
	return body, nil
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
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// UnmarshalJSON
func UnmarshalJSON(body []byte, data any, path ...any) error {
	node := jsoniter.Get(body, path...)
	return sonic.UnmarshalString(node.ToString(), &data)
}

// ParseCode exp: 000001.SH 00700 AAPL
func ParseCode(code string) string {
	pre := strings.Split(code, ".")[0]
	// CN
	if len(pre) > 2 && (pre[0:2] == "SZ" || pre[0:2] == "SH") {
		return fmt.Sprintf("%s.%s", pre[2:], pre[0:2])
	}
	return code
}

// Is CN Stock
func IsCNStock(code string) bool {
	_, suf, ok := strings.Cut(code, ".")
	if ok {
		if suf == "SH" || suf == "SZ" {
			return true
		}
	}
	return false
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
