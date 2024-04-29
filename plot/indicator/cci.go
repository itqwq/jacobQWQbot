package indicator // 定义一个包名为 indicator，用于存放交易指标相关的代码

import (
	"fmt"  // 导入 fmt 包，用于格式化输出
	"time" // 导入 time 包，用于处理时间相关的操作

	"github.com/rodrigo-brito/ninjabot/model" // 导入 ninjabot 模型包，用于处理交易数据模型
	"github.com/rodrigo-brito/ninjabot/plot"  // 导入 ninjabot 绘图包，用于绘制交易指标图表

	"github.com/markcheno/go-talib" // 导入 go-talib 包，用于计算各种交易指标
)

// CCI 函数接受一个周期长度和颜色，返回一个 plot.Indicator 类型的 CCI 指标
// 超买与超卖：当CCI大于+100时，市场被认为是超买的，这可能是卖出的信号。当CCI小于-100时，市场被认为是超卖的，这可能是买入的信号。
// 趋势变化：CCI的方向变化可以反映市场趋势的变化。例如，CCI从正向下穿-100可能表明一个强劲的下行趋势的开始。
// 商品通道指数（Commodity Channel Index，CCI）是一种技术分析指标，用于识别新趋势或警告极端条件。
func CCI(period int, color string) plot.Indicator {
	return &cci{
		Period: period, // 设置 CCI 指标的周期长度
		Color:  color,  // 设置绘图时使用的颜色
	}
}

// cci 结构体定义了 CCI 指标所需的基本属性
type cci struct {
	Period int                   // CCI 指标的周期长度
	Color  string                // 绘图时使用的颜色
	Values model.Series[float64] // 用于存储计算后的 CCI 值
	Time   []time.Time           // 用于存储对应 CCI 值的时间点
}

// Warmup 方法返回初始化该指标所需的最小数据点数量，通常为周期长度
func (c cci) Warmup() int {
	return c.Period
}

// Name 方法返回该指标的名称，格式为 "CCI(周期长度)"
func (c cci) Name() string {
	return fmt.Sprintf("CCI(%d)", c.Period)
}

// Overlay 方法返回该指标是否应该被绘制在价格图表上方。对于 CCI，通常不直接绘制在价格图上，所以返回 false
func (c cci) Overlay() bool {
	return false
}

// Load 方法接受一个 dataframe 作为输入，使用 go-talib 库计算 CCI 值，并存储结果
func (c *cci) Load(dataframe *model.Dataframe) {
	// 使用 talib.Cci 函数计算 CCI 值，根据周期裁剪结果以匹配数据帧
	c.Values = talib.Cci(dataframe.High, dataframe.Low, dataframe.Close, c.Period)[c.Period:]
	c.Time = dataframe.Time[c.Period:] // 保存计算 CCI 值对应的时间点
}

// Metrics 方法返回该指标的绘图指标，包括颜色、样式、值和时间
func (c cci) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		{
			Color:  c.Color,  // 指定绘图使用的颜色
			Style:  "line",   // 指定绘图样式为线条
			Values: c.Values, // 指定要绘制的 CCI 值
			Time:   c.Time,   // 指定对应的时间点
		},
	}
}
