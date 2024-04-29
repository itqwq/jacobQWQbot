package indicator

import (
	"fmt"
	"time"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/plot"

	"github.com/markcheno/go-talib"
)

// EMA 创建一个指数移动平均线（EMA）指标实例。
func EMA(period int, color string) plot.Indicator {
	return &ema{
		Period: period, // EMA 的计算周期，表示在多少个数据点上计算平均值。
		Color:  color,  // 指标线的颜色。
	}
}

// ema 结构体定义了指数移动平均线（EMA）指标的内部数据结构。
type ema struct {
	Period int                   // EMA 的计算周期。
	Color  string                // 指标线的颜色。
	Values model.Series[float64] // 存储计算后的EMA值。
	Time   []time.Time           // 对应EMA值的时间序列。
}

// Warmup 返回计算指标所需的最小数据点数，即指标的周期。
func (e ema) Warmup() int {
	return e.Period
}

// Name 返回指标的名称，以格式化的字符串表示，包含其周期。
func (e ema) Name() string {
	return fmt.Sprintf("EMA(%d)", e.Period)
}

// Overlay 指示这个指标是否应该在主图表上绘制（与价格图叠加）。
func (e ema) Overlay() bool {
	return true
}

// Load 根据提供的数据加载并计算指标值。
func (e *ema) Load(dataframe *model.Dataframe) {
	if len(dataframe.Time) < e.Period {
		return // 如果提供的数据少于指标周期，则不进行计算。
	}

	// 调用 TA-Lib 的 Ema 方法计算 EMA，并将结果赋值给 e.Values。
	// 计算结果的时间段比输入的时间段短，因为部分数据用于“预热”计算。
	e.Values = talib.Ema(dataframe.Close, e.Period)[e.Period:]
	e.Time = dataframe.Time[e.Period:]
}

// Metrics 返回用于图表绘制的指标度量信息。
// 在这个例子中，它返回一个线条样式的度量，包括颜色、值和时间。
func (e ema) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		{
			Style:  "line",   // 绘制样式为线条。
			Color:  e.Color,  // 线条颜色。
			Values: e.Values, // 线条对应的值。
			Time:   e.Time,   // 线条对应的时间。
		},
	}
}
