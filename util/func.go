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

// Get XueQiu api
func XueQiuAPI(url string) ([]byte, error) {
	// add token
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Add("cookie", "xq_a_token=25916c3bfec27272745f6070d664a48d4b10d322")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Err(err)
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
		log.Warn().Msgf("unmarshal err: %v", err)
		return err
	}

	raw, _ := node.Raw()
	return sonic.UnmarshalString(raw, &data)
}
