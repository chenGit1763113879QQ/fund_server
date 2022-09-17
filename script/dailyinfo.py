import pandas as pd
import pymongo
import tushare as ts
from pymongo import UpdateOne

pro = ts.pro_api(token='8dbaa93be7f8d09210ca9cb0843054417e2820203201c0f3f7643410')

# 最新日期
cal = pro.adj_factor(ts_code="600519.SH", start_date='20220901', fields="trade_date")
latest_date = cal['trade_date'].max()

MARKET_CN = 1
MARKET_HK = 2
MARKET_US = 3

TYPE_STOCK = 4
TYPE_INDEX = 5

class DailyInfo:
    def __init__(self):
        self.client = pymongo.MongoClient("mongodb://localhost:27017")["fund"]
        self.stockdb = self.client["stock"]

        self.index_members()
        self.fina_data()

    # 指数成分编制
    def index_members(self):
        index = self.stockdb.find({'marketType': MARKET_CN, 'type': TYPE_INDEX}).distinct('_id')
 
        docs = []
        for i in index:
            df = pro.index_weight(index_code=i, fields="con_code")
            members = df['con_code'].to_list()

            docs.append(UpdateOne(
                {'_id': i}, {'$set': {'members': members, 'count': len(members)}}
            ))
        self.stockdb.bulk_write(docs)

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

        self.stockdb.bulk_write([
            UpdateOne({'_id': code}, {'$set': i.dropna().to_dict()}) for code, i in df.iterrows()
        ])

        # 历史
        self.client['fina'].drop()
        for year in ['2018', '2019', '2020', '2021', '2022']:
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

DailyInfo()
