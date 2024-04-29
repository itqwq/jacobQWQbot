package tools

/*
这段代码定义了一个名为 TrailingStop 的类型，它代表了一个动态追踪止损的功能。动态追踪止损是一种投资策略，它允许投资者根据资产价格的变化来调整止损水平，以便保护已实现的利润。
*/
type TrailingStop struct {
	current float64 // 当前价格
	stop    float64 // 止损价格
	active  bool    // 追踪止损功能是否激活的标志
}

// NewTrailingStop 创建一个新的 TrailingStop 实例
func NewTrailingStop() *TrailingStop {
	return &TrailingStop{}
}

// Start 启动追踪止损功能，并设置当前价格和止损价格
func (t *TrailingStop) Start(current, stop float64) {
	t.stop = stop
	t.current = current
	t.active = true
}

// Stop 停止追踪止损功能
func (t *TrailingStop) Stop() {
	t.active = false
}

// Active 返回追踪止损功能是否激活的状态
func (t TrailingStop) Active() bool {
	return t.active
}

// Update 更新当前价格，并根据当前价格与止损价格的关系来判断是否需要触发止损
func (t *TrailingStop) Update(current float64) bool {
	if !t.active {
		// 如果止损没有激活，则直接返回false
		return false
	}
	//例如新的当前价格 (current) 是110元。旧的当前价格 (t.current) 是100元，价格差额 = 110 - 100 = 10元，原止损价格 (t.stop) 是95元。，新的止损价格 = 旧止损价格 + 价格差额 = 95 + 10 = 105元。将 t.current 更新为新的当前价格，即110元。
	if current > t.current {
		// 当当前价格高于之前的价格时，更新止损价格
		// 止损价格上调为之前的止损价格加上当前价格与之前价格的差额
		t.stop = t.stop + (current - t.current)
		t.current = current
		// 返回false表示不触发止损
		// 当当前价格 (current) 高于之前记录的价格 (t.current) 时，意味着投资表现良好，价格在上涨。此时，将止损价格 (t.stop) 上调，是为了保护已经获得的利润。
		return false
	}

	// 当当前价格不高于之前的价格时，更新当前价格
	t.current = current
	// 在运算中比较符号都是布尔表达式，current <= t.stop 如果当前价格小于或者等于止损价，就返回true，否则则返回false，表示触发止损价
	return current <= t.stop
}
