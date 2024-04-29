package indicator

// 引入必要的包
import (
	"fmt"  // 用于格式化字符串
	"time" // 用于处理时间相关的操作

	"github.com/rodrigo-brito/ninjabot/model" // Ninjabot库的模型部分，用于处理数据模型
	"github.com/rodrigo-brito/ninjabot/plot"  // Ninjabot库的绘图部分，用于绘制图表

	"github.com/markcheno/go-talib" // Talib库，用于技术分析计算
)

// RSI 函数返回一个RSI指标的实例
// RSI（Relative Strength Index，相对强弱指数）是一种动量指标，用于衡量股票价格上涨和下跌的速度和变化的大小，以判断股票或其他金融资产的过度买入或过度卖出状态。RSI值超过70通常被认为是过度买入区域，可能是一个卖出信号。RSI值低于30则通常被视为过度卖出区域，可能是一个买入信号。
func RSI(period int, color string) plot.Indicator {
	// 创建并返回rsi结构的新实例，设定周期和颜色
	return &rsi{
		Period: period,
		Color:  color,
	}
}

// rsi 结构体定义了RSI指标需要的数据
type rsi struct {
	Period int                   // RSI计算的周期
	Color  string                // 绘图时使用的颜色
	Values model.Series[float64] // 存储计算后的RSI值
	Time   []time.Time           // 对应每个RSI值的时间序列
}

// Warmup 方法返回初始化所需的数据点数，等于RSI周期
func (e rsi) Warmup() int {
	//返回至少需要的周期数，周期可以作为参考，然后计算出rsi
	return e.Period
}

// Name 方法返回该指标的名称，包括周期
func (e rsi) Name() string {
	return fmt.Sprintf("RSI(%d)", e.Period)
}

// Overlay 方法表示该指标不需要覆盖在主图上
func (e rsi) Overlay() bool {
	return false
}

// Load 方法加载数据帧并计算RSI值
func (e *rsi) Load(dataframe *model.Dataframe) {
	// 如果数据点不足以计算RSI，则直接返回
	if len(dataframe.Time) < e.Period {
		return
	}

	// 使用Talib库计算RSI值，忽略不足周期的数据
	//	//我们有一个30天的数据序列，使用14天作为RSI的周期。意味着前14天，我们没有足够的数据来计算第一个RSI值 意思就是前14天就只有13天甚至更少的数据，我们的周期需要14天的数据，所以从15天开始才够数据
	e.Values = talib.Rsi(dataframe.Close, e.Period)[e.Period:]
	// 调整时间序列，与计算后的RSI值对应
	e.Time = dataframe.Time[e.Period:]
}

// Metrics 方法定义如何在图表上展示RSI指标
func (e rsi) Metrics() []plot.IndicatorMetric {
	// 返回一个指标度量数组，包含绘图样式和数据
	return []plot.IndicatorMetric{
		{
			Color:  e.Color,  // 指定颜色
			Style:  "line",   // 绘图样式为线形
			Values: e.Values, // RSI值
			Time:   e.Time,   // 对应的时间序列
		},
	}
}
