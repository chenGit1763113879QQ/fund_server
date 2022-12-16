package job

import (
	"fmt"
	"fund/db"
	"fund/model"
	"fund/util"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

func getRealStock(m *model.Market) {
	url := fmt.Sprintf("https://xueqiu.com/service/v5/stock/screener/quote/list?size=5000&order_by=amount&market=%v", m.Market)

	for freq := m.Freq(); ; m.Incr() {
		if freq == 2 {
			log.Info().Msgf("update stock[%s]", m.Market)
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

		go getDistribution(m)

		if m.Market == util.CN {
			go getMainFlow()
			go getNorthMoney()
		}

		Cond.Broadcast()

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
