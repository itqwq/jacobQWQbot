package indicator // 定义指标相关功能的包

import (
	"time" // 导入时间处理的包

	"github.com/rodrigo-brito/ninjabot/model" // 导入 ninjabot 的模型包，用于访问交易数据结构
	"github.com/rodrigo-brito/ninjabot/plot"  // 导入 ninjabot 的绘图包，用于实现指标的图形化

	"github.com/markcheno/go-talib" // 导入 go-talib 包，用于计算各种技术指标
)

// OBV 函数创建并返回一个 OBV 指标对象
// 趋势确认：当价格和OBV同时上升，表示上升趋势得到了成交量的支持，认为上升趋势稳固；当价格和OBV同时下降，表示下降趋势得到成交量的支持，认为下降趋势稳固。
// OBV的计算相对简单：如果今天的收盘价高于昨天的收盘价，则将今天的成交量加到昨天的OBV上；如果今天的收盘价低于昨天的收盘价，则从昨天的OBV中减去今天的成交量；如果今天的收盘价与昨天的收盘价相等，OBV保持不变。
func OBV(color string) plot.Indicator {
	return &obv{
		Color: color, // 设置绘图时使用的颜色
	}
}

// obv 结构体定义了 OBV 指标所需的基本属性
type obv struct {
	Color  string                // 绘图时使用的颜色
	Values model.Series[float64] // 存储计算出的 OBV 值
	Time   []time.Time           // 对应每个 OBV 值的时间点
}

// Warmup 方法返回计算该指标所需的最小数据点数，对于 OBV 来说，没有预热数据点，故返回 0
// OBV 不需要预热值 他是成交量的积累
func (e obv) Warmup() int {
	return 0
}

// Name 方法返回该指标的名称，即 "OBV"
func (e obv) Name() string {
	return "OBV"
}

// Overlay 方法指示该指标是否应该覆盖在主图表上，对于 OBV，通常不覆盖在价格图表上，所以返回 false
func (e obv) Overlay() bool {
	return false
}

// Load 方法接受一个数据帧（df）作为输入，并使用 go-talib 包的 Obv 函数计算 OBV 值
func (e *obv) Load(df *model.Dataframe) {
	e.Values = talib.Obv(df.Close, df.Volume) // 计算 OBV 值
	e.Time = df.Time                          // 保存每个 OBV 值对应的时间点
}

// Metrics 方法返回一个包含该指标绘图数据的切片
func (e obv) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		{
			Color:  e.Color,  // 绘图使用的颜色
			Style:  "line",   // 绘图样式为线条
			Values: e.Values, // 要绘制的 OBV 值
			Time:   e.Time,   // 对应的时间点
		},
	}
}
