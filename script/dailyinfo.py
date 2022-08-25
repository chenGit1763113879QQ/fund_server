import time
from concurrent import futures

import pandas as pd
import progressbar
import pymongo
import tushare as ts
from pymongo import UpdateOne, InsertOne

pro = ts.pro_api(token='8dbaa93be7f8d09210ca9cb0843054417e2820203201c0f3f7643410')

# 最新日期
cal = pro.adj_factor(ts_code="600519.SH", start_date='20220201', fields="trade_date")
latest_date = cal['trade_date'].max()


class DailyInfo:
    def __init__(self):
        self.client = pymongo.MongoClient("mongodb://localhost:27017")["fund"]
        self.realStock = self.client["stock"]

        pool = futures.ThreadPoolExecutor(max_workers=8)
        pool.submit(self.stock_basic)
        pool.submit(self.index_members)
        pool.submit(self.ths_bk)
        pool.submit(self.fina_data)
        pool.submit(self.events)
        pool.submit(self.after_trade)
        pool.shutdown()

    # 基本信息
    def stock_basic(self):
        opts = []
        # 涨跌停价格
        limit = pro.stk_limit()
        for i in limit.itertuples():
            opts.append(UpdateOne({"_id": i.ts_code}, {"$set": {'up_limit': i.up_limit, 'down_limit': i.down_limit}}))

        # 北向资金持股
        hk = pro.hk_hold()
        for i in hk.itertuples():
            opts.append(UpdateOne({"_id": i.ts_code}, {"$set": {'ratio': i.ratio, 'hk_vol': i.vol}}))

        # 基本数据
        basic = pd.merge(
            # 融资融券
            pro.margin_target(is_new='Y', fields='ts_code,mg_type'),
            # 基本信息
            pro.stock_basic(fields='ts_code,is_hs,list_date,market'),
            on='ts_code',
        ).set_index('ts_code')

        for code, i in basic.iterrows():
            opts.append(UpdateOne({"_id": code}, {"$set": {'basic': i.to_dict()}}))

        self.realStock.bulk_write(opts)

    # 指数成分编制
    def index_members(self):
        index = self.realStock.find({'type': 'index', 'marketType': 'CN'})

        docs = []
        for i in list(index):
            df = pro.index_weight(index_code=i['_id'], fields="con_code")
            members = df['con_code'].to_list()
            docs.append(UpdateOne(
                {'_id': i['_id']}, {'$set': {'members': members, 'count': len(members)}}
            ))
        self.realStock.bulk_write(docs)

    # 同花顺板块编制
    def ths_bk(self):
        # I1 行业 I2 细分行业
        # 代码881：一级行业；884：为细分行业
        ids = pro.ths_index(exchange='A', type='I')
        ids = ids[~ids.name.str.contains('Ⅲ')]
        ids['type'] = ids['ts_code'].map(lambda x: 'I1' if x[0:3] == '881' else 'I2')

        # 概念
        concept = pro.ths_index(exchange='A', type='N')
        concept = concept[concept['count'] < 500]
        # 去除指定概念
        for c_name in ['成份股', '样本股']:
            concept = concept[~concept.name.str.contains(c_name)]

        # 864开头是美股概念
        concept = concept[concept.ts_code.str[0:3] != '864']
        concept['type'] = 'C'

        df = pd.concat([ids, concept]).set_index('ts_code')

        # 获取成分股
        bar = progressbar.ProgressBar(max_value=len(df), prefix='更新板块中...')

        # 清空
        self.realStock.update_many({'marketType': 'CN', 'type': 'stock'}, {"$set": {'bk': []}})

        count = 0
        for i in df.itertuples():
            member = pro.ths_member(ts_code=i.Index)['code'].tolist()

            # 写入
            self.realStock.update_one({"_id": i.Index}, {"$set": {
                'name': i.name, 'marketType': 'CN', 'members': member, 'count': len(member), 'type': i.type
            }}, upsert=True)

            # 更新成份股板块列表
            self.realStock.update_many({'_id': {'$in': member}}, {'$addToSet': {'bk': i.Index}})

            count += 1
            bar.update(count)
            time.sleep(0.21)

        bar.finish()

    # 财务数据
    def fina_data(self):
        # 最新
        df = pd.concat([
            pro.income_vip(
                fields='ts_code,end_date,end_type,revenue,operate_profit,n_income,basic_eps'
            ).drop_duplicates(subset=['ts_code'], keep='last').set_index('ts_code'),

            pro.fina_indicator_vip(
                fields='ts_code,end_date,roe,roa,eps,bps,grossprofit_margin,or_yoy,op_yoy,netprofit_yoy'
            ).drop_duplicates(subset=['ts_code'], keep='last').set_index('ts_code'),
        ], axis=1)

        self.realStock.bulk_write([
            UpdateOne({'_id': code}, {'$set': i.dropna().to_dict()}) for code, i in df.iterrows()
        ])

        # 历史
        self.client['fina'].drop()
        for year in ['2017', '2018', '2019', '2020', '2021', '2022']:
            for q in ['0331', '0630', '0930', '1231']:
                df = pd.concat([
                    pro.income_vip(period=year + q).drop_duplicates(subset=['ts_code'], keep='last').set_index(
                        'ts_code'),
                    pro.fina_indicator_vip(period=year + q).drop_duplicates(subset=['ts_code'], keep='last').set_index(
                        'ts_code'),
                    pro.cashflow_vip(period=year + q).drop_duplicates(subset=['ts_code'], keep='last').set_index(
                        'ts_code'),
                ], axis=1)

                df['ts_code'] = df.index
                self.client['fina'].insert_many([i.dropna().to_dict() for index, i in df.iterrows()])

    # 公司大事
    def events(self):
        data_range = 10
        docs = []

        # 股份回购
        df = pro.repurchase()
        for i in range(data_range):
            temp = pro.repurchase(end_date=df['ann_date'].values[-1])
            df = pd.concat([df, temp])

        df['type'] = '股份回购'
        docs += [InsertOne(i.dropna().to_dict()) for idx, i in df.iterrows()]

        # 股权变动
        df = pro.stk_holdertrade()
        for i in range(data_range):
            temp = pro.stk_holdertrade(end_date=df['ann_date'].min())
            df = pd.concat([df, temp])

        df['type'] = '股权变动'
        docs += [InsertOne(i.dropna().to_dict()) for idx, i in df.iterrows()]

        # 业绩预告
        df = pro.forecast_vip()
        for i in range(data_range):
            temp = pro.forecast_vip(end_date=df['ann_date'].min())
            df = pd.concat([df, temp])

        df = df.rename(columns={'type': 'change_type'})
        df = df.drop_duplicates(subset=['ts_code', 'end_date'])
        df['type'] = '业绩预告'
        docs += [InsertOne(i.dropna().to_dict()) for idx, i in df.iterrows()]

        # 业绩快报
        df = pro.express_vip()
        df = df.drop_duplicates(subset=['ts_code', 'end_date'])
        df['type'] = '业绩快报'
        docs += [InsertOne(i.dropna().to_dict()) for idx, i in df.iterrows()]

        # 写入数据库
        self.client.drop_collection('events')
        self.client['events'].create_index([("ts_code", 1), ("type", 1)])
        self.client['events'].bulk_write(docs)

    # 盘后数据
    def after_trade(self):
        # 大宗交易
        df = pro.block_trade(start_date='20220301')
        for i in range(10):
            temp = pro.block_trade(end_date=df['trade_date'].min())
            df = pd.concat([df, temp])

        self.client.drop_collection('blockTrade')
        self.client['blockTrade'].insert_many(df.fillna('').to_dict(orient='records'))

        # 龙虎榜
        df = pd.DataFrame()
        for date in cal['trade_date']:
            temp = pro.top_list(trade_date=date)
            df = pd.concat([df, temp])

        self.client.drop_collection('topList')
        self.client['topList'].insert_many(df.fillna('').to_dict(orient='records'))


t = DailyInfo()
