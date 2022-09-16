import hashlib
from concurrent import futures
from datetime import datetime

import numpy as np
import pandas as pd
import progressbar
import pymongo
import tushare as ts
from pymongo import UpdateOne

# 设置token
pro = ts.pro_api(token='8dbaa93be7f8d09210ca9cb0843054417e2820203201c0f3f7643410')

# 连接mongo
realStock = pymongo.MongoClient("mongodb://localhost:27017")["fund"]["stock"]
klineDB = pymongo.MongoClient("mongodb://localhost:27017")["kline"]


def md5_code(code: str) -> str:
    val = hashlib.md5(code.encode('utf8')).hexdigest()
    return str(ord(val[0]) % 8) + val[1]


# 数据库更新类
class DataBase:
    def __init__(self):
        self.i = 0

    def start(self):
        self.stock_indicate()

    # 后台运行
    def background_worker(self, func=None, arg_list=None, work_name=None, work_num=16):
        self.i = 0
        bar = progressbar.ProgressBar(max_value=len(arg_list), prefix=work_name)

        def handler_func(args):
            for arg in args:
                try:
                    func(arg)
                except Exception as e:
                    print(e)
                finally:
                    self.i += 1
                    bar.update(self.i)

        arg_list = np.array_split(arg_list, work_num)

        # 初始化线程
        worker = futures.ThreadPoolExecutor(max_workers=work_num)
        [worker.submit(handler_func, x) for x in arg_list]
        worker.shutdown(wait=True)
        bar.finish()

    # 股票指标
    def stock_indicate(self):
        def func(dt: str):
            df = pd.concat([
                pro.stk_factor(trade_date=dt).set_index('ts_code'),
                pro.daily_basic(trade_date=dt,
                                fields='turnover_rate,volume_ratio,ts_code,pe_ttm,pb,ps_ttm,dv_ttm').set_index(
                    'ts_code'),
                pro.moneyflow(trade_date=dt,
                              fields="ts_code,buy_lg_amount,sell_lg_amount,buy_elg_amount,sell_elg_amount,net_mf_amount").set_index(
                    'ts_code'),
                pro.cyq_perf(trade_date=dt, fields='ts_code,weight_avg,winner_rate').set_index('ts_code'),

            ], axis=1).rename(
                columns={'net_mf_amount': 'net', 'pct_change': 'pct_chg', 'turnover_rate': 'tr', 'volume_ratio': 'vr'})

            # 大单资金流
            df['main_buy'] = df['buy_lg_amount'] + df['buy_elg_amount']
            df['main_sell'] = df['sell_lg_amount'] + df['sell_elg_amount']
            df['main_net'] = df['main_buy'] - df['main_sell']

            # 去除多余数据
            df = df.drop(columns=['trade_date'])
            for col in df.columns:
                if 'lg_amount' in col:
                    df = df.drop(columns=[col])

            # 写入数据
            df['coll_name'] = df.index.map(lambda x: md5_code(x))
            dt = datetime.strptime(dt, "%Y%m%d")

            for coll, df in df.groupby('coll_name'):
                docs = []
                df = df.drop(columns=['coll_name'])

                for code, i in df.iterrows():
                    row = i.dropna().to_dict()
                    row['code'] = code
                    row['time'] = dt
                    docs.append(UpdateOne({'code': code, 'time': dt}, {'$set': row}, upsert=True))

                # ensure index
                klineDB[coll].create_index([("code", 1), ("time", 1)], unique=True)
                # 写入
                klineDB[coll].bulk_write(docs)

        cal = pro.index_daily(ts_code='000001.SH', start_date='20120101', fields="trade_date")
        self.background_worker(func=func, arg_list=cal['trade_date'], work_name='更新股票行情')


t = DataBase()
t.start()
