package strategy

import (
	log "github.com/sirupsen/logrus"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/service"
)

// Controller 结构体定义了一个策略控制器，用于管理策略的执行和数据更新。
type Controller struct {
	strategy  Strategy         // 策略对象
	dataframe *model.Dataframe // K线数据
	broker    service.Broker   // 交易所服务
	started   bool             // 标志位，表示控制器是否已启动
}

// NewStrategyController 创建一个新的策略控制器实例。
func NewStrategyController(pair string, strategy Strategy, broker service.Broker) *Controller {
	dataframe := &model.Dataframe{
		Pair:     pair,
		Metadata: make(map[string]model.Series[float64]),
	}

	return &Controller{
		dataframe: dataframe,
		strategy:  strategy,
		broker:    broker,
	}
}

// Start 方法用于启动策略控制器。
func (s *Controller) Start() {
	s.started = true
}

// OnPartialCandle 方法在每个新的部分完成的K线时被调用，用于更新K线数据并执行高频交易逻辑。
func (s *Controller) OnPartialCandle(candle model.Candle) {
	// 如果k线未完成，并且k线数据的收盘价 >=  预加载的数据(说明执行了预加载)
	if !candle.Complete && len(s.dataframe.Close) >= s.strategy.WarmupPeriod() {
		// 检查策略是否实现了HighFrequencyStrategy接口，并且如果实现了，将 s.strategy 的实际类型转换为 HighFrequencyStrategy
		if str, ok := s.strategy.(HighFrequencyStrategy); ok {
			// 更新K线数据 更新数据框架 (s.dataframe) 以包含最新的部分完成的K线数据。这一步是为了确保所有计算和决策都基于最新的市场信息。
			s.updateDataFrame(candle)
			// 计算指标为最新的数据计算交易指标。这些指标将用于生成交易信号或进行市场分析。
			str.Indicators(s.dataframe)
			// 执行部分完成K线的逻辑
			str.OnPartialCandle(s.dataframe, s.broker)
		}
	}
}

// updateDataFrame 方法用于更新K线数据。
func (s *Controller) updateDataFrame(candle model.Candle) {
	// 检查k线是否存在 是否有时间数据，并且检查传入的k线时间戳是否与dataframe最后时间戳相同s.dataframe.Time[len(s.dataframe.Time)-1])  时间戳是一个切片 拿到长度-1 得到最后一个，这意味着，我们认为传入的K线数据是对最新（最后一条）记录的更新
	if len(s.dataframe.Time) > 0 && candle.Time.Equal(s.dataframe.Time[len(s.dataframe.Time)-1]) {
		last := len(s.dataframe.Time) - 1
		s.dataframe.Close[last] = candle.Close
		s.dataframe.Open[last] = candle.Open
		s.dataframe.High[last] = candle.High
		s.dataframe.Low[last] = candle.Low
		s.dataframe.Volume[last] = candle.Volume
		s.dataframe.Time[last] = candle.Time
		for k, v := range candle.Metadata {
			s.dataframe.Metadata[k][last] = v
		}
		// 如果条件不成立就保持原有的数据不变的情况，在数据末尾添加一个一条全新的记录
	} else {
		s.dataframe.Close = append(s.dataframe.Close, candle.Close)
		s.dataframe.Open = append(s.dataframe.Open, candle.Open)
		s.dataframe.High = append(s.dataframe.High, candle.High)
		s.dataframe.Low = append(s.dataframe.Low, candle.Low)
		s.dataframe.Volume = append(s.dataframe.Volume, candle.Volume)
		s.dataframe.Time = append(s.dataframe.Time, candle.Time)
		s.dataframe.LastUpdate = candle.Time
		for k, v := range candle.Metadata {
			s.dataframe.Metadata[k] = append(s.dataframe.Metadata[k], v)
		}
	}
}

// OnCandle 方法在每个新的K线结束后被调用，用于更新K线数据并执行策略逻辑。
func (s *Controller) OnCandle(candle model.Candle) {
	// 检查接收到的K线时间是否早于已有K线数据的最新时间这表明接收到的K线数据是“过时”的，可能是因为网络延迟或数据传输问题s，如果是则记录错误日志并返回
	if len(s.dataframe.Time) > 0 && candle.Time.Before(s.dataframe.Time[len(s.dataframe.Time)-1]) {
		log.Errorf("late candle received: %#v", candle)
		return
	}

	// 更新K线数据
	s.updateDataFrame(candle)

	// 如果K线数据的长度已达到策略预热期的要求，则执行策略逻辑
	if len(s.dataframe.Close) >= s.strategy.WarmupPeriod() {
		// 从K线数据中获取最新的预热期数据样本
		sample := s.dataframe.Sample(s.strategy.WarmupPeriod())
		// 计算指标
		s.strategy.Indicators(&sample)
		// 如果控制器已启动，则执行策略的K线结束后逻辑
		if s.started {
			s.strategy.OnCandle(&sample, s.broker)
		}
	}
}
