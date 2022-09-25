package util

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"unicode"

	"github.com/bytedance/sonic"
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

func Md5Code(code string) string {
	m := md5.New()
	m.Write([]byte(code))
	val := hex.EncodeToString(m.Sum(nil))
	return fmt.Sprintf("%X%c", val[0]%8, val[1])
}

func IsChinese(str string) bool {
	for _, r := range str {
		// only check first character
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
	function DeCompressJson
	src: {
		"column": ["a", "b", "c"],
		"item": [
			[1, 2, 3],
			[4, 5, 6]
		]
	}
	dst: [
		{"a": 1, "b": 2, "c": 3},
		{"a": 4, "b": 5, "c": 6}
	]
*/
func DeCompressJson(src []byte) (dst []byte, err error) {
	var data struct {
		Column []string `json:"column"`
		Item   [][]any  `json:"item"`
	}
	if err = UnmarshalJSON(src, &data); err != nil {
		log.Error().Msg(err.Error())
		return
	}

	dstMap := make([]map[string]any, len(data.Item))
	for i, item := range data.Item {
		dstMap[i] = map[string]any{}

		for c, col := range data.Column {
			dstMap[i][col] = item[c]
		}
	}

	dst, err = sonic.Marshal(&dstMap)
	return
}
