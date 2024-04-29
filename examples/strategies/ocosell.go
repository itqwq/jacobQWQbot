// 引入相关的包
package strategies

import (
	"github.com/markcheno/go-talib"               // 引入技术分析库，用于计算技术指标
	"github.com/rodrigo-brito/ninjabot/indicator" // ninjabot的指标计算库
	"github.com/rodrigo-brito/ninjabot/model"     // ninjabot的数据模型库
	"github.com/rodrigo-brito/ninjabot/service"   // ninjabot的服务接口库
	"github.com/rodrigo-brito/ninjabot/strategy"  // ninjabot的策略接口库
	"github.com/rodrigo-brito/ninjabot/tools/log" // ninjabot的日志工具库
)

// OCOSell结构体，定义了一个具体的交易策略
type OCOSell struct{}

// Timeframe 返回使用的时间框架，这里是“1d”，表示每个数据点代表一天
func (e OCOSell) Timeframe() string {
	return "1d"
}

// WarmupPeriod 返回初始化这个策略需要的历史数据点数量，这里是9，指的是至少需要9根K线
func (e OCOSell) WarmupPeriod() int {
	return 9
}

// Indicators 定义并计算策略所需的技术指标，这里计算的是随机振荡指标
func (e OCOSell) Indicators(df *model.Dataframe) []strategy.ChartIndicator {
	//这个方法是不是得到一个周期为3的慢速k线，还有慢速D线当慢速%K线上穿慢速%D线时，这通常被视为买入信号。当慢速%K线下穿慢速%D线时，这通常被视为卖出信号。
	df.Metadata["stoch"], df.Metadata["stoch_signal"] = indicator.Stoch(
		df.High,   // 最高价
		df.Low,    // 最低价
		df.Close,  // 收盘价
		8,         // 随机振荡器的K周期长度，你实际上是在决定用多少天的数据来计算这个指标。这个“周期”确实指的是在计算指标时考虑的特定天数的数据范围，所以它直接影响指标的敏感度和信号的产生短周期：使用较短的周期（例如5天）会让随机振荡指标更快地反应价格变化，适用于追求快速交易的交易者。长周期：使用较长的周期（例如14天或更多）会使指标对短期波动的反应减缓，从而提供较为平滑且稳定的信号。
		3,         // 随机振荡器的平滑K，当%K或%D低于20时，市场被认为是超卖的，这可能是一个买入信号。当%K或%D高于80时，市场被认为是超买的，这可能是一个卖出信号。这个“3”实际上代表的是用来计算慢速%K线的移动平均的天数或周期数，而不是百分比。这意味着在计算出快速%K线之后，你会再对这个结果应用一个周期为3天的简单移动平均（SMA），以得到慢速%K线。例如Day 1: 20%，Day 2: 25%，Day 3: 30% 在计算三天的平均值 得到一个25\%，当快速%K线（较敏感）从下方穿越慢速%D线（较平稳）向上时，这通常被看作是买入信号。
		talib.SMA, // 使用简单移动平均作为平滑函数
		3,         // D周期长度
		talib.SMA, // 使用简单移动平均作为D的计算方法
	)

	// 返回图表指标，用于可视化
	return []strategy.ChartIndicator{
		{
			Overlay:   false,        // 指标不覆盖在主价格图上
			GroupName: "Stochastic", // 指标的组名为“Stochastic”
			Time:      df.Time,      // 时间序列
			Metrics: []strategy.IndicatorMetric{
				{
					Values: df.Metadata["stoch"],
					Name:   "K",                // K线指标
					Color:  "red",              // K线为红色
					Style:  strategy.StyleLine, // 线型显示
				},
				{
					Values: df.Metadata["stoch_signal"],
					Name:   "D",                // D线指标
					Color:  "blue",             // D线为蓝色
					Style:  strategy.StyleLine, // 线型显示
				},
			},
		},
	}
}

// 这段代码是OCOSell策略的OnCandle方法的实现，用于处理每当新的K线（蜡烛图）数据生成时的交易逻辑
func (e *OCOSell) OnCandle(df *model.Dataframe, broker service.Broker) {
	// 获取最新的收盘价格
	closePrice := df.Close.Last(0)
	// 记录日志，显示新K线的信息，包括交易对、最后更新时间和收盘价
	log.Info("New Candle = ", df.Pair, df.LastUpdate, closePrice)

	// 获取当前的资产和报价货币的持仓情况
	assetPosition, quotePosition, err := broker.Position(df.Pair)
	// 如果查询持仓信息时出现错误，记录错误并返回
	if err != nil {
		log.Error(err)
		return
	}

	// 设置买入操作的金额阈值为4000.0
	buyAmount := 4000.0
	// 检查报价货币的持仓是否足够，并且检查随机指标的K线是否穿过D线，作为买入信号
	if quotePosition > buyAmount && df.Metadata["stoch"].Crossover(df.Metadata["stoch_signal"]) {
		// 根据当前的价格和买入金额计算购买的资产数量
		size := buyAmount / closePrice
		// 尝试创建市场买入订单
		_, err := broker.CreateOrderMarket(model.SideTypeBuy, df.Pair, size)
		// 如果创建订单失败，记录错误详情
		if err != nil {
			log.WithFields(map[string]interface{}{
				"pair":  df.Pair,
				"side":  model.SideTypeBuy,
				"close": closePrice,
				"asset": assetPosition,
				"quote": quotePosition,
				"size":  size,
			}).Error(err)
		}

		// 创建OCO订单（一单成交即取消另一单），用于设置止盈和止损，止盈价格（closePrice*1.1，即当前价格的110%）、止损价格（closePrice*0.95，即当前价格的95%）、取消价格（closePrice*0.95，也是当前价格的95%）
		_, err = broker.CreateOrderOCO(model.SideTypeSell, df.Pair, size, closePrice*1.1, closePrice*0.95, closePrice*0.95)
		// 如果创建OCO订单失败，记录错误详情
		if err != nil {
			log.WithFields(map[string]interface{}{
				"pair":  df.Pair,
				"side":  model.SideTypeBuy,
				"close": closePrice,
				"asset": assetPosition,
				"quote": quotePosition,
				"size":  size,
			}).Error(err)
		}
	}
}
