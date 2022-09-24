package job

import (
	"fmt"
	"fund/db"
	"fund/model"
	"fund/util"
	"math"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

func getRealStock(m *model.Market) {
	url := fmt.Sprintf("https://xueqiu.com/service/v5/stock/screener/quote/list?size=5000&order_by=amount&type=%s", m.StrType)

	for {
		freq := m.Freq()

		if freq == 2 {
			log.Info().Msgf("update stock[%s]", m.StrType)
		}

		body, err := util.GetAndRead(url)
		if err != nil {
			continue
		}

		var data []*model.Stock
		util.UnmarshalJSON(body, &data, "data", "list")

		bulk := db.Stock.Bulk()

		for _, s := range data {
			s.CalData(m)

			if s.Price > 0 {
				// update db
				bulk.UpdateId(s.Id, bson.M{"$set": s})

				// insert db
				if freq == 2 {
					db.Stock.InsertOne(ctx, s)
				}
			}
		}

		bulk.Run(ctx)
		go updateMinute(data, m)
		go getIndustry(m)

		if freq >= 1 {
			go getDistribution(m)

			if m.Market == util.MARKET_CN {
				go getMainFlow()
				go getNorthMoney()
			}
		}
		Cond.Broadcast()
		m.Incr()

		for !m.Status {
			time.Sleep(time.Millisecond * 100)
			m.ReSet()
		}
		time.Sleep(time.Millisecond * 500)
	}
}

func updateMinute(s []*model.Stock, m *model.Market) {
	tradeTime := m.TradeTime.Format("2006/01/02 15:04")
	date := strings.Split(tradeTime, " ")[0]

	newTime, _ := time.Parse("2006/01/02 15:04", tradeTime)

	coll := db.MinuteDB.Collection(date)
	if m.Freq() == 2 {
		coll.EnsureIndexes(ctx, nil, []string{"code,minute"})
	}

	a := time.Now()
	if a.Second() > 15 && a.Second() < 45 {
		return
	}

	bulk := coll.Bulk()

	for _, i := range s {
		id := fmt.Sprintf("%s-%s", i.Id, tradeTime)
		bulk.UpsertId(
			id,
			bson.M{"_id": id, "code": i.Id, "time": newTime.Unix(),
				"price": i.Price, "pct_chg": i.PctChg, "vol": i.Vol,
				"avg": i.Avg, "main_net": i.MainNet, "minute": newTime.Minute()},
		)
	}
	go bulk.Run(ctx)
}

func getNews() {
	var stocks []*struct {
		Code string `bson:"_id"`
		Name string `bson:"name"`
	}
	db.Stock.Find(ctx, bson.M{}).All(&stocks)

	var news []*struct {
		Datetime string `csv:"datetime"`
		Content  string `csv:"content"`
		Title    string `csv:"title"`
	}

	// 中文名去除后缀
	for _, s := range stocks {
		if util.IsChinese(s.Name) {
			s.Name = strings.Split(s.Name, "-")[0]
		}
	}

	loc, _ := time.LoadLocation("Asia/Shanghai")

	if err := util.TushareApi("news", bson.M{"src": "eastmoney"}, "datetime,title,content", &news); err != nil {
		log.Error().Msg(err.Error())
		return
	}

	for _, n := range news {
		// 匹配
		codes := make([]string, 0)
		for _, s := range stocks {
			if strings.Contains(n.Title, s.Name) && s.Name != "证券" {
				if len(codes) < 5 {
					codes = append(codes, s.Code)
				}
			}
		}

		t, _ := time.ParseInLocation("2006-01-02 15:04:05", n.Datetime, loc)
		db.Article.InsertOne(ctx, &model.Article{
			Title:    n.Title,
			Content:  n.Content,
			CreateAt: t,
			Tag:      codes,
		})
	}
}

func getCNStocks() []string {
	var id []string
	db.Stock.Find(ctx, bson.M{
		"marketType": util.MARKET_CN, "type": util.TYPE_STOCK, "mc": bson.M{"$gt": 500 * math.Pow(10, 8)},
	}).Distinct("_id", &id)
	return id
}
