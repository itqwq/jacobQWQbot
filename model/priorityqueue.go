package model

import "sync"

/*

这段代码定义了一个线程安全的优先级队列（PriorityQueue），它通过使用 Go 语言的 sync.Mutex 来保证线程安全。优先级队列中的元素必须实现 Item 接口，该接口定义了一个 Less 方法用于比较元素间的优先级。队列支持基本操作如添加元素（Push）、删除并返回优先级最高的元素（Pop）、返回但不删除最高优先级的元素（Peek）和获取队列长度（Len）。此外，队列在添加或移除元素时可触发回调函数，例如 notifyCallbacks 在每次添加新元素时执行。这种数据结构通常用于需要快速访问最小或最大元素的应用场景，例如任务调度、事件驱动处理等。
*/

// PriorityQueue 定义了一个线程安全的优先级队列。
type PriorityQueue struct {
	sync.Mutex                   // 嵌入Mutex以提供线程安全的访问
	length          int          // 队列中当前元素的数量
	data            []Item       // 存储队列元素的切片
	notifyCallbacks []func(Item) // 每当一个新消息被加入到优先级队列中，系统可能需要立即通知管理员或特定服务这一事件，以便进行进一步处理。通过在 notifyCallbacks 中注册相关的通知函数，系统可以在添加新消息到队列时自动执行这些函数，从而触发即时的响应操作。
}

// Item 接口定义了队列中元素必须实现的方法。
// Less 方法比较两个元素的优先级，如果当前元素的优先级小于参数元素的优先级，返回 true。
type Item interface {
	Less(Item) bool
}

// NewPriorityQueue 创建并返回一个新的 PriorityQueue 实例。
// 它接收一个 Item 切片作为初始化数据，并将其构建成一个堆。
func NewPriorityQueue(data []Item) *PriorityQueue {
	q := &PriorityQueue{data: data, length: len(data)}
	// 初始化堆。从最后一个非叶子节点开始，向下调整每个节点以满足堆的性质。
	if q.length > 0 {
		for i := (q.length >> 1) - 1; i >= 0; i-- {
			q.down(i)
		}
	}
	return q
}

// Push 向优先级队列中添加一个新元素，并保持堆的性质。
func (q *PriorityQueue) Push(item Item) {
	q.Lock() // 确保线程安全
	defer q.Unlock()

	// 将新元素添加到队列的末尾，并增加长度
	q.data = append(q.data, item)
	q.length++
	// 对新添加的元素执行上浮操作，以保持堆的性质
	q.up(q.length - 1)

	// 异步执行所有注册的通知回调函数
	for _, notify := range q.notifyCallbacks {
		go notify(item)
	}
}

// PopLock 注册一个回调函数，该函数会在元素被弹出时触发。
// 返回一个 channel，用于异步接收被弹出的元素。
func (q *PriorityQueue) PopLock() <-chan Item {
	ch := make(chan Item)
	q.notifyCallbacks = append(q.notifyCallbacks, func(_ Item) {
		ch <- q.Pop()
	})
	return ch
}

// Pop 移除并返回优先级队列中优先级最高的元素。
func (q *PriorityQueue) Pop() Item {
	q.Lock() // 确保线程安全
	defer q.Unlock()

	if q.length == 0 {
		return nil // 如果队列为空，则返回 nil
	}
	// 取出并保存队列顶部元素
	top := q.data[0]
	q.length--
	if q.length > 0 {
		// 将最后一个元素移动到顶部，并执行下沉操作
		q.data[0] = q.data[q.length-1]
		q.down(0)
	}
	// 移除并返回顶部元素
	q.data = q.data[:q.length]
	return top
}

// Peek 返回但不移除优先级队列中优先级最高的元素。
func (q *PriorityQueue) Peek() Item {
	q.Lock() // 确保线程安全
	defer q.Unlock()

	if q.length == 0 {
		return nil // 如果队列为空，则返回 nil
	}
	return q.data[0] // 返回顶部元素
}

// Len 返回优先级队列中元素的数量。
func (q *PriorityQueue) Len() int {
	q.Lock() // 确保线程安全
	defer q.Unlock()

	return q.length
}

// down 是一个私有方法，用于执行下沉操作，保持堆的性质。
func (q *PriorityQueue) down(pos int) {
	data := q.data
	halfLength := q.length >> 1
	item := data[pos]
	for pos < halfLength {
		left := (pos << 1) + 1
		right := left + 1
		best := left
		if right < q.length &&
			data[right].Less(data[best]) {
			best = right
		}
		if !data[best].Less(item) {
			break
		}
		data[pos] = data[best]
		pos = best
	}
	data[pos] = item
}

// up 是一个私有方法，用于执行上浮操作，保持堆的性质。
func (q *PriorityQueue) up(pos int) {
	data := q.data
	item := data[pos]
	for pos > 0 {
		parent := (pos - 1) >> 1
		current := data[parent]
		if !item.Less(current) {
			break
		}
		data[pos] = current
		pos = parent
	}
	data[pos] = item

}
