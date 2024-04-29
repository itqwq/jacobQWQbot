package notification

import (
	"fmt"      // 用于字符串格式化
	"net/smtp" // 提供了SMTP协议的实现，用于发送电子邮件

	log "github.com/sirupsen/logrus" // 强大的日志记录库

	"github.com/rodrigo-brito/ninjabot/model" // ninjabot的模型包，提供了订单等结构体的定义
)

// Mail 结构体定义了邮件发送器需要的属性。
// SMTP服务器是用来发送、接收和转发电子邮件的服务器。
type Mail struct {
	//这个auth字段就是用来存储用于验证的信息，告诉SMTP服务器：“我是谁，我有权发送邮件，请根据我的用户名和密码确认我的身份并处理我的邮件请求。” 如果服务器接受了这些认证信息，你的邮件就可以被成功发送出去。
	auth smtp.Auth // SMTP认证信息
	//smtpServerAddress 是服务器的网络地址，它可以是一个IP地址（如 192.168.1.1）或是一个域名（如 smtp.example.com）。就像你家的街道地址一样，它告诉邮件客户端服务器在哪里。
	smtpServerPort int
	//smtpServerPort 是服务器上特定的端口号，它决定了网络上数据交换的入口。端口号就像是你家的门牌号，确保邮件被送到正确的地方。SMTP服务通常会使用标准端口25，加密连接通常会使用465或587端口。
	smtpServerAddress string // SMTP服务器的地址

	to   string // 邮件接收者地址
	from string // 邮件发送者地址
}

// Notify 方法用于发送邮件通知。
func (t Mail) Notify(text string) {
	serverAddress := fmt.Sprintf(
		"%s:%d", // 将SMTP服务器的地址和端口格式化成字符串
		t.smtpServerAddress,
		t.smtpServerPort)

	message := fmt.Sprintf(
		`To: "User" <%s>\nFrom: "NinjaBot" <%s>\n%s`, // 创建邮件内容，包括收件人、发件人和邮件正文
		t.to,
		t.from,
		text,
	)
	//smtp.SendMail函数是实际发送邮件的命令，需要下面几个参数，SMTP服务器地址和端口，认证信息，发件人，收件人列表，邮件内容
	//直接初始化切片你是在创建并同时初始化一个包含特定元素的切片，关于容量，直接初始化的切片其容量是根据提供的元素数量自动确定的，make定义的切片，它更多用于创建一个指定长度和容量的空切片，而不是立即用具体的元素初始化
	err := smtp.SendMail(
		serverAddress,   // SMTP服务器地址和端口
		t.auth,          // 认证信息
		t.from,          // 发件人
		[]string{t.to},  // 收件人列表
		[]byte(message)) // 邮件内容
	if err != nil {
		//这段代码的目的就是把发生的错误和一条错误信息一起记录（打印）到日志中
		log.
			WithError(err).                                // 使用logrus记录错误信息
			Errorf("notification/mail: couldnt send mail") // 打印错误信息
	}
}

// OnOrder 方法在订单状态更新时调用，用来发送相关的邮件通知。
// 这个OnOrder方法的作用是，根据不同的订单状态，自动生成相应的通知邮件标题和内容，并调用Notify方法发送这些邮件。这样的设计使得在订单状态发生变化时，可以方便地通知用户。
func (t Mail) OnOrder(order model.Order) {
	//title字符串变量被用来根据订单的不同状态存储相应的邮件标题。这个标题反映了订单的当前状态（如订单已完成、新订单、订单被取消或拒绝等），并且通常会包含订单的具体信息，比如交易对（例如BTC/USD）。然后，这个定制化的标题和订单的详情一起构成邮件的内容，通过Notify方法发送出去，让邮件的接收者知道相关订单的最新状态。
	title := ""
	switch order.Status {
	case model.OrderStatusTypeFilled:
		//当订单状态是已完成时，title标题填入已完成的信息，并转化成字符串
		title = fmt.Sprintf("✅ ORDER FILLED - %s", order.Pair) // 订单完成通知
	case model.OrderStatusTypeNew:
		////当订单状态是新订单时，title标题填入已新订单的信息，并转化成字符串
		title = fmt.Sprintf("🆕 NEW ORDER - %s", order.Pair) // 新订单通知
	case model.OrderStatusTypeCanceled, model.OrderStatusTypeRejected:
		//当订单状态是已取消或者已拒绝状态时，title标题填入已取消或者已拒绝的信息，并转化成字符串
		title = fmt.Sprintf("❌ ORDER CANCELED / REJECTED - %s", order.Pair) // 订单取消或拒绝通知
	}

	message := fmt.Sprintf("Subject: %s\nOrder %s", title, order) // 创建邮件内容，包含订单信息
	t.Notify(message)                                             // 发送邮件通知
}

// OnError 方法在程序发生错误时调用，用来发送错误通知。
func (t Mail) OnError(err error) {
	message := fmt.Sprintf("Subject: 🛑 ERROR\nError %s", err) // 创建包含错误信息的邮件内容
	t.Notify(message)                                         // 调用Notify函数传入message邮件信息，发送邮件通知
}

// MailParams 结构体定义了创建Mail实例所需要的参数。
// MailParams用于先指定发送邮件所需要的一切参数，比如你要连接哪个邮件服务器、使用什么端口、邮件是从谁发给谁的，以及用什么密码进行认证等等。然后，Mail根据MailParams提供的这些参数，执行实际的邮件发送工作。这样的设计让邮件发送的配置和执行变得清晰和分离，使得管理和使用变得更加方便。
type MailParams struct {
	SMTPServerPort    int    // SMTP服务器的端口
	SMTPServerAddress string // SMTP服务器的地址

	To       string // 邮件接收者地址
	From     string // 邮件发送者地址
	Password string // 用于SMTP认证的密码
}

// NewMail 函数根据提供的参数创建一个新的Mail实例。
// 通过这个NewMail函数，你可以轻松地根据不同的发送需求配置多个Mail实例，每个实例都有自己的发送者、接收者、服务器设置等，准备好被用来发送邮件。
func NewMail(params MailParams) Mail {
	return Mail{
		from:              params.From,              // 发件人地址
		to:                params.To,                // 收件人地址
		smtpServerPort:    params.SMTPServerPort,    // SMTP服务器端口
		smtpServerAddress: params.SMTPServerAddress, // SMTP服务器地址
		//这段代码是验证我们的邮箱地址与密码，然后验证通过就可以发送邮件
		auth: smtp.PlainAuth( // 设置SMTP的PlainAuth认证
			"",                       //第一个参数是一个空字符串，这是一个“身份标识符”，用于SMTP协议的认证过程中。在大多数情况下，这个参数可以留空。
			params.From,              //发送人地址
			params.Password,          //发送人密码
			params.SMTPServerAddress, //是SMTP服务器的地址，它在这里还有一个作用，即在建立加密连接时验证服务器证书的合法性。
		),
	}
}
