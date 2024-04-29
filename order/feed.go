package order

import (
	"github.com/rodrigo-brito/ninjabot/model"
)

/*
这段代码实现了一个订阅-发布系统，专门用于处理和分发金融交易订单信息。它允许各个组件根据特定的货币对（交易对）订阅订单数据，并在有新的订单信息时接收通知。这个系统设计用于支持并发操作，能够高效地处理和分发大量的实时交易数据。总体来说，这个系统的核心功能和目的数据流管理，动态订阅，实时数据发布，并发和异步处理，订阅者通知这段代码是构建高性能交易系统和金融应用的基础设施的一部分，特别适用于需要处理大量实时数据和支持多个数据消费者的场景。通过提供一个灵活且高效的数据订阅和发布机制，它帮助系统解耦了数据的生产者和消费者，提高了整体的数据处理能力和系统的可扩展性。
*/
// DataFeed 结构定义了一个特定的数据通道，用于传输订单数据和错误信息。
// 它包含两个通道：一个用于订单数据，另一个用于错误信息。

/*
Data 通道
确实是用来接收订单的。当用户下订单时，订单信息就发送到这个通道中，等待系统的后端部分处理这个订单。无论是自动交易系统中的订单还是任何需要被系统进一步处理的订单信息，都可以通过这个通道传递。

Err 通道
这个通道专门用来处理错误信息。当在处理订单（或任何其他操作）过程中出现错误时，错误信息就发送到这个通道中，等待系统的错误处理机制介入。
*/
type DataFeed struct {
	Data chan model.Order // 用于传输订单数据的通道
	Err  chan error       // 用于传输错误信息的通道
}

// FeedConsumer 定义了一个函数类型，它接受一个订单作为参数。
// 任何符合这个签名的函数都可以作为订单数据的消费者。
// 换句话说，FeedConsumer 定义了一个接口，任何符合这个接口（即接收一个model.Order参数并且不返回任何结果的函数）的函数都可以作为一个消费者函数来处理订单数据。
type FeedConsumer func(order model.Order)

// Feed 结构是订阅-发布系统的核心，管理着所有的订单数据流（OrderFeeds）
// 和订阅信息（SubscriptionsBySymbol）。
type Feed struct {
	//这是一个映射（map），键（key）是表示货币对的字符串（例如 "BTC/USD"），值（value）是指向 DataFeed 实例的指针。每个 DataFeed 负责流式传输订单数据和错误信息。
	OrderFeeds map[string]*DataFeed // 按货币对组织的订单数据流
	//SubscriptionsBySymbol对特定货币对的订阅者的集合，这同样是一个映射（map），其键同样是货币对的字符串表示，但值是 Subscription 结构体的切片。这表示每个货币对可以有多个订阅者。Subscription 结构体包含订阅者的具体信息，例如是否只对新订单感兴趣以及如何处理接收到的订单数据的消费者函数。这个映射使系统能够跟踪哪些消费者订阅了哪些货币对的数据流，从而在有新订单数据到来时通知它们。
	SubscriptionsBySymbol map[string][]Subscription // 按货币对组织的订阅列表
}

// Subscription 结构定义了一个订阅，包含一个标志来指示是否只对新订单感兴趣，
// 以及一个消费者函数，用于处理接收到的订单数据。
type Subscription struct {
	//onlyNewOrder: 这是一个布尔类型的字段，用来标记订阅者是否仅对新订单感兴趣。如果此字段为 true，则表示订阅者只希望接收新生成的订单的通知；如果为 false，则表示订阅者对所有订单（包括旧订单和新订单）都感兴趣。这个字段允许订阅者根据自己的需求定制他们接收订单数据的方式，从而提高了系统的灵活性和效率。
	onlyNewOrder bool // 标志：是否只订阅新订单
	//consumer: 这是一个 FeedConsumer 类型的字段，代表了一个消费者函数，用于接收并处理订单数据。FeedConsumer 是一个函数类型，其签名为 func(order model.Order)，意味着任何符合这个签名的函数都可以用作订单数据的消费者。这个消费者函数是订阅者定义的逻辑，用于对接收到的订单数据进行处理，比如执行交易策略、记录日志、更新用户界面等。
	consumer FeedConsumer // 消费者函数：接收并处理订单数据
}

// NewOrderFeed 函数初始化并返回一个新的 Feed 实例。
// 它为 OrderFeeds 和 SubscriptionsBySymbol 分别创建了空的映射，
// 准备用于存储订单数据流和订阅信息。
// 创建这个实例，可以灵活改变里面的数据
func NewOrderFeed() *Feed {
	return &Feed{
		OrderFeeds:            make(map[string]*DataFeed),      // 初始化空的订单数据流映射
		SubscriptionsBySymbol: make(map[string][]Subscription), // 初始化空的订阅列表映射
	}
}

// 该方法允许组件针对特定的货币对订阅订单数据，同时指定如何处理这些数据。
// pair string：方法的第一个参数，表示货币对的字符串，如 "BTC/USD"。这是订阅者想要订阅订单数据的货币对。
// consumer FeedConsumer：第二个参数是一个类型为 FeedConsumer 的函数，这是一个处理订单的函数，订阅者用它来处理接收到的订单数据。
// onlyNewOrder bool：第三个参数是一个布尔值，指示订阅者是否只对新订单感兴趣。
func (d *Feed) Subscribe(pair string, consumer FeedConsumer, onlyNewOrder bool) {
	//这个过程确保了每个特定的货币对都有自己的数据流实例，用于后续的订单数据和错误信息的传输，如果某个货币对尚未有数据流被初始化，则会自动创建一个。
	if _, ok := d.OrderFeeds[pair]; !ok {
		d.OrderFeeds[pair] = &DataFeed{
			Data: make(chan model.Order),
			Err:  make(chan error),
		}
	}
	//这段代码的功能是为特定的货币对增加一个新的订阅者。它通过在SubscriptionsBySymbol映射中对应的货币对下的订阅列表（切片）添加一个Subscription实例来实现。这样，每当有新的订单数据发布到这个货币对时，系统就会根据这个订阅列表中的每个Subscription来决定如何处理和分发订单数据。
	d.SubscriptionsBySymbol[pair] = append(d.SubscriptionsBySymbol[pair], Subscription{
		onlyNewOrder: onlyNewOrder,
		consumer:     consumer,
	})
}

// _ bool: 这是一个匿名布尔类型的参数。匿名意味着它在函数体内不会被直接使用。这个参数可能被设计为满足某个接口，或者为未来的使用预留，但在当前的实现中它没有被使用。
func (d *Feed) Publish(order model.Order, _ bool) {
	//检查这个交易对是否有对应的频道对吗 如果有就把订单传入这个频道里面去
	//频道就是把一个整体的大项目分成很多小部分，分别分给不同的频道不同的人去完成，一旦这些小项目完成了，再合起来项目就可以进入下个阶段了,通过把订单更新事件发送到特定的频道，系统的其他部分（比如订单处理逻辑）就可以监听这个频道，一旦有更新事件发生，它们就进行相应的处理 。 相互频道独立完成工作的同时，又能协同完成整体目标 ，很好的实现了并发性
	if _, ok := d.OrderFeeds[order.Pair]; ok {
		d.OrderFeeds[order.Pair].Data <- order
	}
}

// Start 方法，它的作用是启动整个订阅-发布系统，确保每当有新的订单数据发布时，所有对应货币对的订阅者都能接收并处理这些数据。
func (d *Feed) Start() {
	//循环遍历所有的订单数据流，这个for循环确实为每一个货币对的订单流分配了一个独立的goroutine。这种方式允许每个订单流的数据并发地、独立地被处理，确保了不同货币对的订单数据可以同时且高效地分发给所有订阅了相应货币对的订阅者，而彼此之间不会相互影响。
	for pair := range d.OrderFeeds {
		// 启动一个新的 goroutine（Go 语言的并发执行单元），以异步方式监听该货币对的订单数据流。这样做可以确保系统能够同时处理来自不同货币对的订单数据。
		go func(pair string, feed *DataFeed) {
			//遍历数据流通道，里面有发送订单了可能有市价单，止损单等，就遍历出来
			for order := range feed.Data {
				// 遍历所有订阅了该货币对更新的订阅者。这些订阅者的信息（包括如何处理订单数据的consumer函数）存储在 SubscriptionsBySymbol 映射的对应条目中。
				for _, subscription := range d.SubscriptionsBySymbol[pair] {
					//对于检索到的每个订阅者，通过调用其 consumer 函数 subscription.consumer(order)，将当前订单数据 order 传递给它们。这意味着每个订阅者都会接收到每个新的订单数据，并根据自己提供的函数逻辑来处理这些数据。
					subscription.consumer(order)
				}
			}
			//pair, d.OrderFeeds[pair] 意思就是这个匿名函数传过来的参数 从外面传过来的值，pair, d.OrderFeeds[pair]  传参给pair string, feed *DataFeed
		}(pair, d.OrderFeeds[pair])
	}
}
