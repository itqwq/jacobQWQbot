package indicator // 定义了一个包名为indicator，用于存放技术分析指标的实现

import (
	"fmt"  // 导入fmt包，用于格式化字符串
	"time" // 导入time包，用于处理时间相关的功能

	"github.com/rodrigo-brito/ninjabot/model" // 导入ninjabot的model包，用于访问交易数据结构
	"github.com/rodrigo-brito/ninjabot/plot"  // 导入ninjabot的plot包，用于绘制指标图表

	"github.com/markcheno/go-talib" // 导入go-talib包，用于计算各种技术指标
)

// WillR函数创建并返回一个Williams %R指标对象
// Williams %R指标，也称为威廉姆斯百分比范围，是由Larry Williams开发的一种动量指标，用于衡量超买和超卖水平。它测量当前的收盘价相对于过去一定期间内的最高价和最低价的位置。超买和超卖区域：当Williams %R指标接近0（例如，高于-20）时，表明市场可能进入超买状态，可能会出现价格回调。当指标接近-100（例如，低于-80）时，表明市场可能进入超卖状态，价格可能会反弹。
func WillR(period int, color string) plot.Indicator {
	return &willR{
		Period: period, // 指定计算Williams %R时使用的周期
		Color:  color,  // 指定绘制指标线条时使用的颜色
	}
}

// willR结构体定义了Williams %R指标所需的基本属性
type willR struct {
	Period int                   // 计算指标所需的周期长度
	Color  string                // 绘图时使用的颜色
	Values model.Series[float64] // 存储计算出的Williams %R值
	Time   []time.Time           // 对应每个Williams %R值的时间点
}

// Warmup方法返回计算该指标所需的最小数据点数，等于指定的周期
func (w willR) Warmup() int {
	return w.Period
}

// Name方法返回该指标的名称，格式为"%R(周期长度)"
func (w willR) Name() string {
	return fmt.Sprintf("%%R(%d)", w.Period)
}

// Overlay方法指示该指标是否应该覆盖在主图表上，对于Williams %R，通常不覆盖在价格图表上，所以返回false
func (w willR) Overlay() bool {
	return false
}

// Load方法接受一个数据帧（df）作为输入，并使用go-talib包的WillR函数计算Williams %R值
func (w *willR) Load(dataframe *model.Dataframe) {
	if len(dataframe.Time) < w.Period {
		return // 如果数据帧中的数据点不足以计算Williams %R，则直接返回
	}

	// 计算Williams %R值并存储结果
	w.Values = talib.WillR(dataframe.High, dataframe.Low, dataframe.Close, w.Period)[w.Period:]
	w.Time = dataframe.Time[w.Period:] // 保存每个Williams %R值对应的时间点
}

// Metrics方法返回一个包含该指标绘图数据的切片
func (w willR) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		{
			Style:  "line",   // 绘图样式为线条
			Color:  w.Color,  // 绘图使用的颜色
			Values: w.Values, // 要绘制的Williams %R值
			Time:   w.Time,   // 对应的时间点
		},
	}
}
