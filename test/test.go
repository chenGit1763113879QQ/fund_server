package main

import (
	"fmt"
	"fund/model"
	"fund/util"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/xgzlucario/structx"
)

func getRealStock(m *model.Market) {
	a := time.Now()
	url := fmt.Sprintf("https://xueqiu.com/service/v5/stock/screener/quote/list?size=5000&order_by=amount&type=%s", m.StrType)
	body, err := util.GetAndRead(url)
	if err != nil {
		return
	}

	fmt.Println("time1", time.Since(a))

	data := structx.NewList[*model.Stock]()

	node := jsoniter.Get(body, "data", "list")
	data.UnmarshalJSON([]byte(node.ToString()))

	data.Sort(func(s1, s2 *model.Stock) bool {
		return s1.MainNet < s2.MainNet
	})

	data.Range(func(_ int, s *model.Stock) bool {
		s.CalData(m)
		return false
	})

	fmt.Println("time2", time.Since(a))
}

func main() {
	for {
		getRealStock(&model.Market{
			Market:    util.CN,
			Type:      util.STOCK,
			StrMarket: "CN",
			StrType:   "sh_sz",
		})
	}
}
