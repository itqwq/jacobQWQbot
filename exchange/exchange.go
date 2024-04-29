// Package exchange defines the data feed and subscription management for the trading bot.
package exchange

import (
	// 引入必要的依赖包。
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	// 使用StudioSol/set库提供集合功能，这里用于管理字符串集合。
	"github.com/StudioSol/set"

	// 引入项目内部的模型和服务。
	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

/*
这段代码定义了一个名为 exchange 的 Go 语言包，其主要功能是管理交易所的市场数据订阅和数据推送。它允许客户端订阅特定的交易对和时间范围的蜡烛图数据，并通过回调函数对接收到的数据进行处理。核心组件包括数据订阅管理、错误处理、以及数据推送的设置和维护。此外，代码还提供了方法来启动数据订阅服务，可以选择同步或异步加载数据。整体上，这段代码是一个为交易机器人设计的数据接口管理系统，旨在从交易所获取实时的市场数据并根据设定的参数对数据进行操作和反应。
*/

// 定义一些通用的错误类型。
var (
	ErrInvalidQuantity   = errors.New("invalid quantity")             //无效数量
	ErrInsufficientFunds = errors.New("insufficient funds or locked") //当用户的账户余额不足以完成交易或者资金被锁定时返回的错误
	ErrInvalidAsset      = errors.New("invalid asset")                //无效的资产交易时返回的错误
)

// DataFeed 是市场数据的通道，包含了数据和错误两个通道。
type DataFeed struct {
	Data chan model.Candle // 数据通道，传输蜡烛图数据。
	Err  chan error        // 错误通道，用于传递数据获取过程中出现的错误。
}

// DataFeedSubscription 是数据订阅的结构体，管理各种数据订阅。
type DataFeedSubscription struct {
	exchange                service.Exchange          // 交易所的接口。
	Feeds                   *set.LinkedHashSetString  // 订阅的数据源集合{订阅的标识符可能更具体，如 "BTC/USD-1h", "EOS/USD-24h", "ETH/USD-1d" 等}。当 Feeds 字段被声明为 *set.LinkedHashSetString 类型时，它指向一个集合（Set）数据结构，这个集合专门用于存储不重复的字符串值。
	DataFeeds               map[string]*DataFeed      // 数据源的映射，意思就是键是一个键是如BTC/USD-1h"值呢这个结构体包含了实际的市场数据通道和错误通道。
	SubscriptionsByDataFeed map[string][]Subscription // 订阅信息的映射，键是如BTC/USD-1h" 通常用来指代每小时更新一次的比特币对美元的价格数据 值是Subscription 类型 。比如onCandleClose 为 true 然后把开盘价收盘价传给handleCandleClose 函数 然后打印出来。
}

// Subscription 是对数据源的具体订阅定义。比如onCandleClose 为 true 然后把开盘价收盘价传给handleCandleClose 函数 然后打印出来
// 就是拿到k线图的数据  当onCandleClose bool 为true时，在蜡烛图结束时拿到数据，形成完整的图，如果为false， 则订阅者可以在蜡烛图还未完全形成时接收数据。这意味着在蜡烛图的时间段内，任何价格更新都可能触发事件。
type Subscription struct {
	onCandleClose bool             // 是否在蜡烛图关闭时触发。
	consumer      DataFeedConsumer // 数据消费者，当新的蜡烛图数据到来时会被调用。
}

// OrderError 是订单错误的定义，包括错误信息、交易对和数量。如下订单是发现错误的交易对（"BTC/USD"）、以及试图交易的数量（0.5比特币）数量不足
type OrderError struct {
	Err      error   // 错误信息。
	Pair     string  // 交易对。
	Quantity float64 // 交易数量。
}

// Error 方法实现了error接口，返回订单错误的详细信息。例如当 （一个 OrderError 的指针）被传递给 fmt.Println 时，fmt.Println 会检查 orderErr 是否实现了 error 接口
func (o *OrderError) Error() string {
	return fmt.Sprintf("order error: %v", o.Err)
}

// DataFeedConsumer 是一个函数类型，用于处理接收到的蜡烛图数据。DataFeedConsumer不是一个具体的函数，而是一个函数类型。任何具有相同参数列表（一个model.Candle类型的参数）和相同返回类型（没有返回值）的函数都被认为是这个类型的实例
type DataFeedConsumer func(model.Candle)

// NewDataFeed 创建并返回一个新的DataFeedSubscription实例。
func NewDataFeed(exchange service.Exchange) *DataFeedSubscription {
	return &DataFeedSubscription{
		exchange:                exchange,
		Feeds:                   set.NewLinkedHashSetString(),    // 初始化字符串集合。为它的Feeds字段初始化一个空的、有序的、不允许重复的字符串集合不允许订阅重复的交易对
		DataFeeds:               make(map[string]*DataFeed),      // 初始化数据源映射。
		SubscriptionsByDataFeed: make(map[string][]Subscription), // 初始化订阅信息映射。
	}
}

// feedKey 根据交易对和时间范围生成唯一的键。提供交易对和时间范围 生成为一个键，这个键用于便是和区分不同的数据
func (d *DataFeedSubscription) feedKey(pair, timeframe string) string {
	return fmt.Sprintf("%s--%s", pair, timeframe)
}

// pairTimeframeFromKey 解析键并返回交易对和时间范围。
func (d *DataFeedSubscription) pairTimeframeFromKey(key string) (pair, timeframe string) {
	parts := strings.Split(key, "--") //它的作用是将字符串 key 按照指定的分隔符 "--" 分割成多个部分
	return parts[0], parts[1]
}

// Subscribe 方法的作用是为数据提供者的特定交易对和时间范围添加一个新的订阅。这个订阅确保当新的K线数据到来时，可以触发一个预定义的消费者函数（consumer）
func (d *DataFeedSubscription) Subscribe(pair, timeframe string, consumer DataFeedConsumer, onCandleClose bool) {
	key := d.feedKey(pair, timeframe)
	// 将生成的键值添加到数据源集合中。这可能是为了记录或管理所有活跃的数据源。
	d.Feeds.Add(key)
	// Subscribe 方法为特定的交易对和时间范围添加一个新的订阅。
	// 这将在接收到新的蜡烛图数据时通知消费者函数。
	d.SubscriptionsByDataFeed[key] = append(d.SubscriptionsByDataFeed[key], Subscription{
		onCandleClose: onCandleClose, // 是否在蜡烛图关闭时触发消费者函数。
		consumer:      consumer,      // 消费者函数本身，它定义了接收到新K线数据时需要执行的操作。
	})
}

// Preload 方法用于预加载一系列蜡烛图数据。
// 预加载一系列蜡烛图数据" 这个表达的意思是，在交易系统正式开始实时数据处理或交易前，先将历史的蜡烛图数据（即历史的价格变动数据）加载到系统中。这些蜡烛图数据通常包含了每个指定时间段（如一小时或一天）的开盘价、收盘价、最高价和最低价等信息。例如，如果一个交易策略需要基于过去30天的日平均数据来预测未来的价格趋势，那么在系统开始运行前，必须先加载这30天的历史数据。
func (d *DataFeedSubscription) Preload(pair, timeframe string, candles []model.Candle) {
	log.Infof("[SETUP] preloading %d candles for %s-%s", len(candles), pair, timeframe)
	key := d.feedKey(pair, timeframe) // 获取对应的键值。
	for _, candle := range candles {
		if !candle.Complete {
			continue // 如果蜡烛图数据不完整，则跳过。
		}

		// 这段代码中的循环遍历特定交易对（由 key 标识）的所有订阅，并对每个订阅执行相应的消费者函数。这意味着对于每个蜡烛图数据（candle），所有针对这个特定交易对和时间范围的订阅都会被激活，每个订阅的消费者函数都会接收到这个蜡烛图数据并进行处理。每个消费者可能根据其配置的需要对数据进行不同的处理，如进行技术分析、生成交易信号、记录数据等。
		/*
			//一个交易对可以有很多订阅，如
			订阅设置：
			交易对: BTC/USD
			时间框架: 5分钟
			数据处理: 监测价格跳空和异常交易量
			订阅函数：每五分钟分析数据，如果发现异常波动或交易量异常，立即通知团队采取措施。

			订阅设置：
			交易对: BTC/USD
			时间框架: 1小时
			数据处理: 收集数据用于模型训练和实时测试
			订阅函数：每小时收集数据，用于训练和调整机器学习模型。
		*/
		for _, subscription := range d.SubscriptionsByDataFeed[key] {
			subscription.consumer(candle)
		}
	}
}

// Connect 方法连接到交易所，并初始化数据和错误通道。
func (d *DataFeedSubscription) Connect() {
	log.Infof("Connecting to the exchange.")
	//使用Iter()方法或类似的迭代器模式，可以帮助您更方便地遍历和查看复杂数据结构中的内容
	for feed := range d.Feeds.Iter() {
		pair, timeframe := d.pairTimeframeFromKey(feed) // 解析出交易对和时间范围。
		// 这个调用返回两个通道：ccandle用于接收蜡烛图数据，cerr用于接收可能发生的错误
		ccandle, cerr := d.exchange.CandlesSubscription(context.Background(), pair, timeframe)
		// 将返回的蜡烛图数据通道和错误通道保存到d.DataFeeds映射中，键是feed
		d.DataFeeds[feed] = &DataFeed{
			Data: ccandle, // 蜡烛图数据通道。
			Err:  cerr,    // 错误通道。
		}
	}
}

// Start 方法启动数据订阅服务。有一个数据订阅服务被启动，同时还有其他任务需要执行。不想等待loadSync  传入false 可以启动订阅的同时执行其他任务 ，如果true，则必须等待订阅完成，才能执行其他任务
func (d *DataFeedSubscription) Start(loadSync bool) {
	d.Connect()               // 建立连接。
	wg := new(sync.WaitGroup) // 使用WaitGroup等待所有订阅处理完毕。
	// 遍历得到一个一个map 键是key 值是 feed 指向DataFeeds指针里面包含一个数据通道还有一个错误通道
	for key, feed := range d.DataFeeds {
		wg.Add(1) // 为每个数据订阅增加WaitGroup计数。
		go func(key string, feed *DataFeed) {
			// 无线循环监听通道
			for {
				// select 是一个go关键字，用于同时监听多个通道他会堵塞，直到其中一个通道准备好数据一旦有一个通道准备好了数据，select 就会执行相应的分支，并且只会执行其中一个分支
				select {
				// 尝试从数据通道Data读取蜡烛图数据
				case candle, ok := <-feed.Data:
					if !ok {
						wg.Done() // 如果数据通道被关闭，递减WaitGroup计数。结束协程
						return
					}
					// 对于每个键对应的订阅，如果满足条件则执行消费者函数。
					for _, subscription := range d.SubscriptionsByDataFeed[key] {
						if subscription.onCandleClose && !candle.Complete {
							continue // 如果订阅者想要一个完整的蜡烛图（闭市），但是不完整 则跳过继续循环。
						}
						subscription.consumer(candle) // 拿到（subscription.onCandleClose=true）闭市且 图形完整则执行消费者函数。
					}
					// 监听错误通道，如果有错就打印出来
				case err := <-feed.Err:
					if err != nil {
						log.Error("dataFeedSubscription/start: ", err) // 如果错误通道中有错误，记录错误信息。
					}
				}
			}
		}(key, feed) // 为每个数据订阅启动一个goroutine。
	}

	log.Infof("Data feed connected.")
	// 有一个数据订阅服务被启动，同时还有其他任务需要执行。不想等待loadSync  传入false 可以启动订阅的同时执行其他任务 ，如果true，则必须等待订阅完成，才能执行其他任务
	if loadSync {
		wg.Wait() // 如果loadSync为true，则等待所有订阅处理完毕。
	}
}
