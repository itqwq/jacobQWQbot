// 定义交易订单类型的常量
const LIMIT_TYPE = "LIMIT"; // 限价订单
const MARKET_TYPE = "MARKET"; // 市价订单
const STOP_LOSS_TYPE = "STOP_LOSS"; // 止损订单
const LIMIT_MAKER_TYPE = "LIMIT_MAKER"; // 限价做市订单

// 定义交易方向的常量
const SELL_SIDE = "SELL"; // 卖出方向
const BUY_SIDE = "BUY"; // 买入方向

// 定义订单状态的常量
const STATUS_FILLED = "FILLED"; // 订单已完成（成交）
//这个 unpack 函数的作用是从一个对象数组中提取出每个对象的特定属性值，并将这些值组成一个新的数组返回。
//遍历原数组：.map() 方法会遍历 rows 数组中的每一个元素。每次遍历时，当前的元素会被作为参数传递给 .map() 方法中提供的函数。执行函数：对于 rows 数组中的每个元素（在这里是每个 row 对象），.map() 方法都会调用一次提供的匿名函数。这个匿名函数接收当前遍历的元素（row）作为参数。匿名函数对每个元素（row）执行操作后会有一个返回值，这里的操作是返回 row[key] 的值。.map() 方法会把这些返回值收集起来，组成一个新的数组。整个 .map() 方法调用完成后，会返回这个新构建的数组。这个新数组的元素是原 rows 数组中每个对象的 key 属性的值。
//调用的时候可以let result = unpack(rows, 'key'); 新数组名称就是result
function unpack(rows, key) {
  return rows.map(function (row) {
    //就是返回符合数组中key键的值组成新的数组
    return row[key];
  });
}

// 当页面加载完成时执行以下代码 监听页面加载事件DOMContentLoaded
// 用于添加一个事件监听器，它监听的是 DOMContentLoaded 事件。这个事件在文档的HTML被完全加载和解析完成后触发，但在样式表、图片和子框架的加载之前触发。当这个事件触发时，即HTML文档完全加载和解析完成后，会执行给定的回调函数。这确保了在执行脚本之前，所有的DOM元素都已经可用。
//DOMContentLoaded 事件监听器确保了一旦整个页面的DOM（文档对象模型）完全加载和解析完成，就会执行函数里面的内容。
document.addEventListener("DOMContentLoaded", function () {
  // 从 URL 参数中获取交易对信息
  //假设我在浏览一个网页交易所交易对，点击链接是https://example.com/products?pair=BTC/USDT在这个URL中，?pair=BTC/USDT 就是一个查询字符串，它告诉网站服务器你想要查看的是BTC交易对。
// 2：当你的浏览器加载这个页面时，页面上的JavaScript代码可以使用 window.location.search 来获取到这个查询字符串（即 ?pair=BTC/USDT），然后用 URLSearchParams 来解析这个字符串
// 步骤1: 获取查询字符串const queryString = window.location.search
// 步骤2: 解析查询字符串const params = new URLSearchParams(queryString);
// 步骤3: 获取特定的查询参数值const pair = params.get('pair');
  const params = new URLSearchParams(window.location.search);
  //这行代码尝试从URL的查询参数中获取名为 "pair" 的参数值。如果存在这个参数，pair 变量就会被赋予相应的值。如果不存在则pair变量将被赋予一个空字符串""作为默认值。
  const pair = params.get("pair") || "";
  
  //当这段代码运行时，它首先从当前页面的URL中获取pair查询参数的值。，代码使用fetch函数向服务器的/data路径发送一个HTTP GET请求，请求URL附加了查询参数pair（例如/data?pair=BTC/USDT）。服务器接收到这个请求后，根据查询参数pair的值（在这个例子中是BTC/USDT），查询数据库或其他数据源，获取这个交易对的市场数据。
  fetch("/data?pair=" + pair)
    .then((data) => data.json())//服务器获取的响应数据转换为JSON格式的JavaScript对象。
    .then((data) => {//第二个.then() 方法用于处理转换后的JSON数据。
      // 构建蜡烛图数据
      const candleStickData = {
        name: "Candles",//设置蜡烛图的名称为Candles
        x: unpack(data.candles, "time"), // 提取从服务器获取的数据中的蜡烛图的时间数据，然后存储在数组x中这里使用了 unpack() 函数，它会从数据中提取特定的属性值并返回一个新的数组。在这里，提取了蜡烛图的时间数据。
        close: unpack(data.candles, "close"), // 提取收盘价数据放入close数组
        open: unpack(data.candles, "open"), // 提取开盘价数据，放入open数组
        low: unpack(data.candles, "low"), // 提取最低价数据，放入low数组
        high: unpack(data.candles, "high"), // 提取最高价数据，放入high数组
        type: "candlestick", // 设置图表类型为蜡烛图
        xaxis: "x1", // 指定x轴名称x1
        yaxis: "y2", // 指定y轴y2
      };

      // 构建资产数据
      const assetData = {
        //假设 data.asset 的值为 "BTC"，data.quote 的值为 "USD"，那么这行代码生成的名称字符串将是 "Position (BTC/USD)"。
        name: `Position (${data.asset}/${data.quote})`,
        x: unpack(data.asset_values, "time"), // 从后端数组data.asset_values中从提取时间数据放到新的数组X
        y: unpack(data.asset_values, "value"), // 从后端数组data.asset_values提取资产价值数据，放到新数组y
        mode: "lines", // 设置绘制模式为线性
        fill: "tozeroy", // 填充区域 假设你正在绘制某个变量随时间变化的折线图，折线以下的区域填充了某种颜色。当变量值在某个时间点低于零时，这个区域将会被填充。
        xaxis: "x1", // 指定x轴
        yaxis: "y1", // 指定y轴
      };

      // 构建权益数据
      const equityData = {
        name: `Equity (${data.quote})`,
        x: unpack(data.equity_values, "time"), // 提取时间数据
        y: unpack(data.equity_values, "value"), // 提取权益价值数据
        mode: "lines", // 设置绘制模式为线性
        fill: "tozeroy", // 填充区域
        xaxis: "x1", // 指定x轴
        yaxis: "y1", // 指定y轴
      };

      // 构建交易点和注释
      //  用于存储交易点数据，每个元素代表一个交易点，包括交易发生的时间、价格、买卖方向等信息。这些数据最终将用于绘制图表中的交易点。
      const points = []; 
      // 用于存储交易注释数据，每个元素代表一个交易注释，包括注释的位置、内容、样式等信息。这些数据最终将用于在图表中添加交易的注释信息，以便用户查看。
      const annotations = []; 
      //data.candles.forEach((candle) 意思就是遍历后端传过来的蜡烛图数组，然后获得的每一个蜡烛图变成参数，传给forEach函数使得您可以针对每个蜡烛图对象执行特定的操作。这种方式使得您可以遍历整个数组并对每个元素进行处理，而无需显式地使用循环或索引来访问数组中的元素
      data.candles.forEach((candle) => { 
        //对于当前循环中的蜡烛图对象(candle)通过.orders属性获取其中的订单组数据
        candle.orders
        //这是一个箭头函数，用于定义过滤条件。箭头函数的参数 o 代表数组中的每个元素（在这里是订单对象），然后箭头函数返回一个布尔值，表示该订单是否符合过滤条件。
          .filter((o) => o.status === STATUS_FILLED) 
          //对于过滤后已完成的订单组进行遍历，对每个订单执行待定的操作
          .forEach((order) => { 
            // 对于每个订单，创建一个包含以下属性的交易点对象，创建交易点对象的目的是为了更好地记录和分析交易数据
            const point = {
              time: candle.time, // 交易发生的时间，与蜡烛的时间相同
              position: order.price, // 交易价格
              side: order.side, // 交易方向（买入或卖出）
              color: "green", // 默认颜色为绿色
            };
            if (order.side === SELL_SIDE) { // 如果是卖出订单，则将颜色改为红色
              point.color = "red";
            }
            points.push(point); // 将交易点数据添加到 points 数组中
      
            // 构建交易注释 当用户将鼠标悬停在图表上的特定注释位置时，页面会显示出相应的交易详细信息，这有助于用户更方便地了解特定交易的细节。
            const annotation = {
              x: candle.time, // 注释在图表上的 x 坐标位置，与蜡烛的时间相同，以当鼠标悬停在x轴附近时，会显示该时间点。
              y: candle.low, // 注释在图表上的 y 坐标位置，通常选择蜡烛的最低价，鼠标停留在y轴附近时会显示最低价。
              xref: "x1", // x 坐标的参考系为 x 轴
              // y 坐标的参考系为 y 轴
              yref: "y2", 
              // 注释文本，默认为 B这个“B”标记确实是你（或者系统自动）主动去标记的。基本上，当你在图表的某个位置执行了买入股票的操作，你就可以在那个准确的位置放置一个“B”的标记。这样做的目的就是为了给你自己一个明确的视觉提示，告诉你在这个位置发生了买入操作。
              text: "B", 
              //order.price.toLocaleString() 这段代码的意思是将订单价格（order.price）转换成一种更易于阅读的格式，具体表现为根据用户的本地环境（比如语言或地区）来格式化数字。
              // .toPrecision(4)：这个方法是用来格式化数字的，它将数字转换为“指定的有效数字位数”。这里的4表示总共希望显示四位有效数字。如果订单数量是123.4567.toLocaleString()：这个方法将上一步的结果转换成本地化的字符串格式。它会考虑到不同地区对数字的格式化习惯，比如千位分隔符。如果toPrecision(4)的结果是1234，那么在美国英语的环境下，toLocaleString()可能会将其转换为"1,234"。
              // order.profit &&确实是利用了所谓的短路行为如果有值（并且不是0、null、undefined、NaN或空字符串等假值），那么它就被认为是true（真），接下来的操作（比如计算利润百分比）就会执行。如果没有值（或者说是一个假值），那么整个表达式就在那里停止，短路了，就执行后面的空字符串，就是没有利润的意思，
              //order.profit * 100：首先，将order.profit（假设它是一个小数，比如0.1表示10%的利润）乘以100，就是10，然后，使用.toPrecision(2)方法将这个数值格式化为最多两位有效数字的字符串形式就是10
              hovertext: `${order.updated_at}<br>ID: ${order.id}<br>Price: ${order.price.toLocaleString()}<br>Size: ${order.quantity.toPrecision(4).toLocaleString()}<br>Type: ${order.type}<br>${(order.profit && "Profit: " + +(order.profit * 100).toPrecision(2).toLocaleString() + "%") || ""}`, // 鼠标悬停时显示的文本内容，包含订单的详细信息
              showarrow: true, // 是否显示箭头 这个箭头可以指向任何任何你想要突出显示的点。这可以是最高点、开盘点、收盘点，或者是特定的事件发生点，比如重大新闻发布、交易量激增的时刻等。箭头是一种非常灵活的视觉工具，可以帮助观众快速定位并理解图表中的关键信息或者特定注释的含义。
              arrowcolor: "green", // 箭头颜色，默认为绿色
              valign: "bottom", // 注释文本在箭头下方,注释文本是箭头指向所想表达的意思，比如当前位置最低价格 
              borderpad: 4, // 注释文本与箭头之间的距离
              arrowhead: 2, // 箭头头部的大小
              //ay: 20表示箭头的起点或相关联的文本从其原始位置垂直向上移动了20个单位，水平位置ax: 0 没有改变保持原样 让箭头与我们指向的哪个目标有一点距离，更美观点，更清晰
              ax: 0, // 箭头在 x 轴上的偏移量，意味着箭头在水平方向上不会移动
              ay: 20, // 箭头在 y 轴上的偏移量，距离y轴20
              font: { // 注释文本的字体样式
                size: 12,
                color: "green",
              },
            };
      
            if (order.side === SELL_SIDE) { // 如果是卖出订单，则调整注释的一些属性
              annotation.font.color = "red"; // 注释文本颜色为红色
              annotation.arrowcolor = "red"; // 箭头颜色为红色
              annotation.text = "S"; // 注释文本改为 S
              annotation.y = candle.high; // y 坐标位置调整为蜡烛的最高价
              annotation.ay = -20; // 箭头在 y 轴上的偏移量调整为负值，卖出操作时 箭头指向的因该是最高价，这时候k线是向上突起的，所以箭头是反过来ay = -20 也是距离指向目标拉远了距离，更加清晰与美观
              annotation.valign = "top"; // 注释文本在箭头上方
            }
      
            annotations.push(annotation); // 将构建好的注释数据添加到 annotations 数组中
          });
      });
      

    // 构建形状数据
    //就相当于一个四边形 两个y轴一个x轴，左侧的主y轴用于显示股票价格，右侧的第二y轴用于显示交易量，主x轴代表时间轴
    //遍历这个数组然后改变shapes里面的内容
const shapes = data.shapes.map((s) => { // 遍历data.shapes数组，每个元素记作s
  return {
    type: "rect", // 形状类型为矩形
    xref: "x1", // x坐标参考系，指定为图表的主x轴这意味着形状在水平方向上的位置是根据主x轴来确定的。
    yref: "y2", // y坐标参考系，指定为图表的第二y轴
    yaxis: "y2", // 指定矩形使用的y轴，同样是图表的第二y轴 y2（第二y轴）用于展示次要数据系列，或当这组数据的量级、单位与主y轴上的数据显著不同时使用。
    xaxis: "x1", // 指定矩形使用的x轴，是图表的主x轴
    //x0, y0, x1, y1 这四个属性就是用来定义矩形两个对角点的坐标，这四个点就足以确定矩形的大小和形状：
    x0: s.x0, // 矩形的起始x坐标，取自遍历的当前元素s
    y0: s.y0, // 矩形的起始y坐标，同样取自s
    x1: s.x1, // 矩形的结束x坐标，取自s
    y1: s.y1, // 矩形的结束y坐标，取自s
    line: { // 设置矩形的线条属性
      width: 0, // 线条宽度设置为0，意味着矩形边缘不可见
    },
    fillcolor: s.color, // 矩形的填充颜色，取自s的color属性
  };
});

// 判断数据是否有最大回撤值，有的话就执行下面的话。
if (data.max_drawdown) {
  // 计算最大回撤区间中的最高点位置 
  //在reduce函数开始时，如果没有指定初始值，p.value会被设置为数组的第一个元素的value属性值，而v.value会从数组的第二个元素开始迭代。如果p.value大于或等于v.value，则p保持不变，如果p.value小于v.value，则p更新为v.value。reduce返回的是遍历过程中遇到的value属性的最大值。
  //如果是（p > v.value为真），reduce函数"返回"的是p。这里的“返回”意味着在这次迭代中，p保持不变，因为它已经大于或等于v.value。如果不是（p > v.value为假），则“返回”v.value。这意味着p被更新为v.value，因为我们发现了一个新的更大值。
  //找出最大资产
  const topPosition = data.equity_values.reduce((p, v) => {
    // 比较p与v.value，如果p大于v.value，则保持p不变；否则，更新p为v.value
    return p > v.value ? p : v.value;
  });

  // 创建矩形形状来表示最大回撤区间，添加到一个shapes数组中
  shapes.push({
    type: "rect", // 形状类型为矩形
    xref: "x1", // x坐标参考系，指定为图表的主x轴
    yref: "y1", // y坐标参考系，指定为图表的第一y轴
    yaxis: "y1", // 指定矩形使用的y轴，同样是图表的第一y轴 y1（主y轴）一般用于展示图表中的主要数据系列或被认为是更重要的数据。
    xaxis: "x1", // 指定矩形使用的x轴，是图表的主x轴
    x0: data.max_drawdown.start, // 矩形的起始x坐标，取自最大回撤区间的起始时间
    y0: 0, // 矩形的起始y坐标，设置为0
    x1: data.max_drawdown.end, // 矩形的结束x坐标，取自最大回撤区间的结束时间
    y1: topPosition, // 矩形的结束y坐标，取自最高点位置
    line: { // 设置矩形的线条属性
      width: 0, // 线条宽度设置为0，意味着矩形边缘不可见
    },
    fillcolor: "rgba(255,0,0,0.2)", // 矩形的填充颜色，设置为红色半透明
    layer: "below", // 将矩形置于图表其他元素的下方
  });

  // 计算注释的位置，位于最大回撤区间中心位置，分别创建了表示最大回撤开始和结束时间的Date对象。
  //我知道了将最大的回撤开始时间转成data对象然后再转化成时间戳 + 最大回撤的结束时间转成data对象再转成时间戳z之和除以2 然后就得到了项目中心的的时间戳 ，
  //例子第一步：将项目开始和结束的日期转换为时间戳，2023年1月1日转换后可能是时间戳1672444800000，2023年1月31日转换后可能是时间戳1675027200000
  //第二步：计算这两个时间戳的平均值，找到中间点的时间戳。(1672444800000 + 1675027200000) / 2 = 1673736000000（这个结果代表了项目中间点的时间戳，
  //第三步：将中间点的时间戳转换回日期格式。时间戳1673736000000转换回日期可能是2023年1月16日
  //中心点可以帮助分析师更好地理解事件前后数据的变化趋势，从而对事件的影响进行更深入的分析。
  //最后再把中心点的时间戳变成data对象将时间戳转换回Date对象的主要原因是为了方便地处理和展示日期及时间信息。
  const annotationPosition = new Date(
    (new Date(data.max_drawdown.start).getTime() +
      new Date(data.max_drawdown.end).getTime()) /
      2
  );

  // 创建注释，显示最大回撤信息
  annotations.push({
    // 注释的x坐标，为最大回撤区间中心位置，注释放在最大回撤区域的中间
    x: annotationPosition, 
    // 注释的y坐标，而topPosition代表这个最高价格。你决定将注释的y坐标设置在这个最高价格的一半，即$50处。这意味着注释将被放置在图表中价格为$50的水平线上。
    y: topPosition / 2.0, 
    //意思就是如果最高价格在这个时间上如4月15日，就把x轴的注释对应到x轴线上4月15日这个点上，最高价格是80，就把y轴的注释对应到y轴线的这个点上
    xref: "x1", // x坐标参考系，指定为图表的主x轴
    yref: "y1", // y坐标参考系，指定为图表的第一y轴
    text: `Drawdown<br>${data.max_drawdown.value}%`, // 注释文本，显示最大回撤信息
    showarrow: false, // 不显示箭头
    font: { // 设置注释文本的字体属性
      size: 12, // 字体大小为12
      color: "red", // 字体颜色为红色
    },
  });
}


      // 构建买入和卖出点数据
      // 在points数组中筛选出所有标记为卖出的交易点  (p)是数组中的每一项
      const sellPoints = points.filter((p) => p.side === SELL_SIDE);
      //在points数组中筛选出所有标记为买入的交易点(p) 是数组中的每一项
      const buyPoints = points.filter((p) => p.side === BUY_SIDE);
      //它用于配置和展示图表中的买入点数据。这个对象将被用于图表绘制库（如Plotly、Chart.js等）中，以便将买入点以可视化的方式呈现在图表上。
      const buyData = {
        name: "Buy Points", // 数据系列的名称，这里表示这些点是买入点。
        /*
        例如：const buyPoints = [
        {time: "2021-01-01", position: 100},
        {time: "2021-01-02", position: 105},
        {time: "2021-01-03", position: 110}
      ];调用unpack(buyPoints, "time")将会返回一个新数组：["2021-01-01", "2021-01-02", "2021-01-03"]。 
         */
        x: unpack(buyPoints, "time"), // x坐标的值，从buyPoints数组中提取每个元素的"time"属性，表示买入发生的时间。 组成一个新的键值对buyPoints[time]
        y: unpack(buyPoints, "position"), // y坐标的值，从buyPoints数组中提取每个元素的"position"属性，可能表示买入时的价格或其他数值位置。
        //xaxis: "x1"会使得所有买入点沿日期轴（x轴）正确对齐。而yaxis: "y2"则允许你将买入点的成交量显示在图表的右侧，即便这些成交量的数值范围与股票价格大不相同。
        xaxis: "x1", // 指定这组数据使用的x轴是图表的主x轴。
        yaxis: "y2", // 指定这组数据使用的y轴是图表的第二y轴。
        mode: "markers", // mode: "markers"，你可以在图表上为每个买入点绘制一个独立的标记。这样，观察者可以清楚地看到每次买入发生的具体时间和价格，但不会被点与点之间的连线分散注意力。
        type: "scatter", // 意思就是把标记变成一个散点图，有助于分析和理解数据点的分布特征和潜在关系。
        marker: {
          color: "green", // 设置标记的颜色为绿色，通常买入点可以用绿色表示，以区分卖出点。
        },
      };
      
      const sellData = {
       // Sell Points 数据
       name: "Sell Points",
       x: unpack(sellPoints, "time"),
       y: unpack(sellPoints, "position"),
       xaxis: "x1",
       yaxis: "y2",
       mode: "markers",
       type: "scatter",
       marker: {
         color: "red", //设置标记红色，为卖出标记
       },
     };

     // 计算独立指标的数量，
     //total的初始值被设置为reduce方法第二个参数提供的值，在这个例子中是0。
    // indicator是数组data.indicators中的当前处理元素。
    // 如果指标没有覆盖在蜡烛图上，就使用total+1 如果覆盖就返回原来的值
    //total被初始化为0，如果没有初始值，值就是数组的第一个数。然后，reduce会遍历data.indicators数组，根据回调函数中的逻辑更新total的值。
     const standaloneIndicators = data.indicators.reduce(
       (total, indicator) => {
         if (!indicator.overlay) {
           return total + 1;
         }
         return total;
       },
       0
     );

     // 设置图表布局
     //这段代码定义了一个名为layout的对象，它被用来配置图表的布局和外观。这个配置涵盖了图表的主题、交互方式、图例显示、轴线设置等多个方面
     //假设你正在创建一个用于分析股票市场的交互式图表，这个图表需要显示过去一年内某只股票的价格变化以及与之相关的交易指标（如成交量、移动平均线等）。
     let layout = {
      //使用"ggplot2"主题，赋予图表R语言中ggplot2包的经典外观，包括其颜色方案、字体和元素布局，使图表看起来既专业又美观。
       template: "ggplot2", 
       //开启"zoom"拖动模式，使用户能够通过鼠标拖拽选择图表的一个区域进行放大，这对于仔细查看特定时间段内股票价格的波动非常有帮助。
       dragmode: "zoom", 
       //设置顶部间距为25单位，确保有足够空间显示图表的标题或上方的图例，改善图表的整体外观和用户体验。
       margin: {
         t: 25, // 设置顶部间距，margin对象定义了图表的外边距。这样做可以确保图表顶部有足够的空间显示标题、图例或其他元素，避免内容被裁剪或紧贴图表边缘，从而改善图表的整体外观和用户体验。
       },
       //你可能会用不同的颜色来标示“开盘价”、“收盘价”、“最高价”和“最低价”，通过启用图例，用户可以快速了解每种颜色代表的含义，从而更好地理解图表信息。
       showlegend: true, // 显示图例
       xaxis: {
         autorange: true, // 这个设置确保图表会根据展示的数据自动调整时间轴的起始和结束点，使得无论数据量大小，图表总是优化显示所有数据点。
         rangeslider: { visible: false }, // 关闭范围滑动条，由于你希望图表看起来更简洁，不需要用户通过滑动条来调整时间范围，因此关闭了范围滑动条的显示。
         showline: true, // 显示轴线，启用这个选项以显示x轴轴线，帮助用户更好地理解图表的时间范围和界限。
         anchor: standaloneIndicators > 0 ? "y3" : "y2", // 设置x轴锚点如果standaloneIndicators（不覆盖在蜡烛图上的独立指标数量）大于0，说明图表中存在至少一个独立的指标需要单独的y轴进行展示。因此，这行代码将x轴的锚点设置为"y3"，意味着x轴将会和图表中的第三个y轴（假设存在）相关联或对齐。
       },
       yaxis2: {
         domain: standaloneIndicators > 0 ? [0.4, 0.9] : [0, 0.9], //就是有独立指标图表中间一大块区域（40%到90%）留给独立指标，其他区域放原来的区域，如果没有独立指标，从图表的最底部（0%）到90%放的是原来的区域
         autorange: true, // 自动调整y轴范围
         mirror: true, // 镜像显示， 镜像显示，这意味着yaxis2的刻度和轴线会在图表的对侧（通常是左侧）镜像显示，这有助于从图表两边读取数据，提高可读性。
         showline: true, // 显示轴线
         gridcolor: "#ddd", // 设置网格颜色，灰色
       },
       yaxis1: {
         domain: [0.9, 1], // 设置y轴1的位置，设置y轴1的位置，这里是从图表底部的90%到100%的位置。因此，y轴1将占据图表的底部10%的位置。
         autorange: true, // 自动调整y轴范围，自动调整y轴范围，使得y轴的范围适应数据的范围，确保数据完整地显示在图表中。
         mirror: true, // 镜像显示，这意味着yaxis2的刻度和轴线会在图表的对侧（通常是左侧）镜像显示，这有助于从图表两边读取数据，提高可读性。
         showline: true, // 显示轴线，轴线的存在可以帮助用户更清楚地识别图表的边界和刻度。
         gridcolor: "#ddd", // 设置网格颜色 灰色
       },
       hovermode: "x unified", // 设置鼠标悬停模式，这意味着当鼠标悬停在图表上时，只显示垂直于x轴的信息。这有助于提供更清晰的数据展示，特别是当图表中有多个数据系列时，可以减少混乱并提高用户体验。
       annotations: annotations, // 添加注释，通过这个属性，添加了一组注释（annotations）到图表中。注释通常用于在图表中标记特定的数据点或提供额外的信息。在这个场景中，annotations包含了一些特定数据点的相关信息，例如最大回撤的数据。
       shapes: shapes, // 添加形状，通过这个属性，添加了一组形状（shapes）到图表中。形状可以用来突出显示特定的数据区域或添加其他可视化效果。在这个场景中，shapes可能用于标记最大回撤区间或其他重要的数据范围。
     };


     //这种组织方式的作用是将所有数据集中到一个数组中，以便稍后将它们一次性地传递给绘图库进行绘制。通过这种方式，可以更方便地管理和维护数据，并将其传递给图表以进行可视化呈现。
     let plotData = [
       candleStickData,//蜡烛图数据
       equityData,//权益数据
       assetData,//资产数据
       buyData,//买入点数据
       sellData,//卖入点数据
     ];

     const indicatorsHeight = 0.39 / standaloneIndicators; // 在这个特定的情境下，0.39可能是为了在图表中留出足够的空间来容纳独立指标，并确保它们不会重叠或与其他元素发生冲突。计算指标高度，它将总高度（0.39）除以独立指标的数量（standaloneIndicators），以确定每个指标的平均高度。
     let standaloneIndicatorIndex = 0; // 独立指标索引，在处理多个独立指标时，可以使用这个索引来确定每个指标在图表中的垂直位置。通过递增或递减这个索引，可以确保每个独立指标都被分配到唯一的位置，而不会重叠或遮挡彼此。
     data.indicators.forEach((indicator) => {
       const axisNumber = standaloneIndicatorIndex + 3; //这行代码用于计算新创建的独立y轴的编号。在布局中，除了主要的y轴（y1和y2），每个独立的指标都会有一个自己的y轴，这些轴的编号会依次递增 standaloneIndicatorIndex 表示当前独立指标的索引，而 + 3 则是因为前面已经有两个默认的y轴（y1和y2），所以从第三个独立指标开始编号从y3开始。
       if (!indicator.overlay) {//判断这个指标是否是独立的，如果是美就执行下面的操作
         const heightStart = standaloneIndicatorIndex * indicatorsHeight; // 计算指标开始高度，。在布局中，每个非覆盖式指标都会被分配一个特定的垂直区域，这个区域的高度取决于指标的数量和图表的总高度。standaloneIndicatorIndex 表示当前指标在处理过程中的索引，indicatorsHeight 表示每个指标在图表中占据的垂直空间的比例。等于指标的索引x指标的高度可以得到当前指标在图表中垂直方向上的起始位置。
         
         //将 "yaxis" 和一个数字 axisNumber 结合起来，从而构成了 y 轴的键名，最终用于访问和设置该 y 轴的属性
         layout["yaxis" + axisNumber] = {
           title: indicator.name, // 设置y轴标题
           domain: [heightStart, heightStart + indicatorsHeight], // 设置y轴的高度范围heightStart起始位置， heightStart + indicatorsHeight 起始位置加上指标的高度，从而确定了该 y 轴在图表中的高度范围。
           autorange: true, // 自动调整该y轴的范围
           mirror: true, // 镜像显示
           showline: true, // 显示轴线
           linecolor: "black", // 设置轴线颜色黑色
           gridcolor: "#ddd", // 设置网格颜色 灰色
         };
         standaloneIndicatorIndex++; //以便为下一个指标分配新的索引，从而保证每个独立指标都有一个唯一的索引值。
       }
  //遍历指标图里面的度量值，如EMA等指标线
       indicator.metrics.forEach((metric) => {
         const data = {
           title: indicator.name, // 设置指标标题
           name: indicator.name + (metric.name && " - " + metric.name), // 设置指标名称，就是如果metric.name为true然后就显示" - " + metric.name名称，如果为false就是短路了显示空字符串 如果 metric.name 存在且为 "EMA"，那么结果就是 "EMA - EMA"；如果 metric.name 不存在，则结果就是 "EMA"。
           x: metric.time, // x轴表示指标的时间数据
           y: metric.value, //y轴表示相应的数值数据
           type: metric.style, // 设置绘制样式
           line: {
             color: metric.color, // 设置线条颜色
           },
           xaxis: "x1", // 设置x轴 //如果这些点是时间点，那么这些时间点就都会按照这个"x1"所代表的时间轴来排列。
           yaxis: "y2", // 设置y轴 意思就是我在这个图中设置的所有y轴上的点都在y2 这个轴上显示
         };
         //如果这个指标图为独立指标就执行下面操作
         if (!indicator.overlay) {
           data.yaxis = "y" + axisNumber; // 设置y轴的坐标为，当前y轴加上对应的指标编号，确保每条指标都有独立的y轴 
         }
         plotData.push(data); // 添加数据，把指标的数据data添加到plotData 数组中
       });
     });
     //"graph"：这个字符串是HTML元素的ID 意思就是把这个图表放在这个id下面的盒子如div，plotData可能包含蜡烛图数据、权益数据、资产数据、买入点数据和卖出点数据等，现在可能还加入了更多的数据。layout：这是一个对象，包含了图表的布局配置信息。布局配置可以包括图表的标题、轴标签、字体大小、颜色方案等多种可视化属性。这行代码的作用就是告诉Plotly：“在ID为'graph'的这个位置，根据plotData中的数据和layout中的布局设置，创建一个新的图表”。执行这行代码后，浏览器中相应的位置就会显示出一个根据提供的数据和配置生成的交互式图表。
     Plotly.newPlot("graph", plotData, layout); // 生成图表
   });
});
