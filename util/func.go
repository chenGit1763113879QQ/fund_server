package util

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"unicode"

	"github.com/bytedance/sonic"
	"github.com/mitchellh/mapstructure"
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
		log.Error().Msg(err.Error())
		return nil, err
	}
	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)
	return body, nil
}

// Get XueQiu api
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

func Md5Code(code string) string {
	m := md5.New()
	m.Write([]byte(code))
	val := hex.EncodeToString(m.Sum(nil))
	return fmt.Sprintf("%X%c", val[0]%8, val[1])
}

// Check str is Chinese
func IsChinese(str string) bool {
	for _, r := range str {
		return unicode.Is(unicode.Han, r)
	}
	return false
}

func Mean[T int | int64 | float64](arr []T) float64 {
	var sum T
	for i := range arr {
		sum += arr[i]
	}
	return float64(sum) / float64(len(arr))
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
	node, err := sonic.Get(body, path...)
	if err != nil {
		log.Warn().Msgf("unmarshal node err: %v", err)
		return err
	}
	raw, _ := node.Raw()
	return sonic.UnmarshalString(raw, &data)
}

/*
	DeCompressJson and use mapstructure to decode
	src: `{
		"column": ["a", "b", "c"],
		"item": [
			[1, 2, 3],
			[4, 5, 6]
		]
	}`
	--> []map[string]any[
		{"a": 1, "b": 2, "c": 3},
		{"a": 4, "b": 5, "c": 6}
	]
	--> dst
*/
func DeCompressJSON(src []byte, dst any) error {
	if src == nil && dst == nil {
		log.Error().Msg("src or dst is nil")
		return errors.New("src or dst is nil")
	}

	var data struct {
		Column []string `json:"column"`
		Item   [][]any  `json:"item"`
	}
	// unmarshal
	if err := UnmarshalJSON(src, &data); err != nil {
		log.Error().Msg(err.Error())
		return err
	}

	// map
	srcMap := make([]map[string]any, len(data.Item))
	for i, item := range data.Item {
		srcMap[i] = map[string]any{}

		for c, col := range data.Column {
			srcMap[i][col] = item[c]
		}
	}

	return mapstructure.Decode(srcMap, dst)
}
