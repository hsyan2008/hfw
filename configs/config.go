package configs

import (
	"time"
)

var Config AllConfig

//Config 项目配置
type AllConfig struct {
	AppID         int64
	EnableSession bool

	Server    ServerConfig
	Logger    LoggerConfig
	Db        DbConfig
	Mongo     MongoConfig
	Cache     CacheConfig
	Template  TemplateConfig
	Route     RouteConfig
	Redis     RedisConfig
	Session   SessionConfig
	HotDeploy HotDeployConfig
	Custom    map[string]string
}

type RedisConfig struct {
	IsCluster  bool
	Server     string
	Prefix     string
	Expiration int32
	PoolSize   int
	//以下两个在集群下无效
	Db       int
	Password string
}

type SessionConfig struct {
	CookieName string
	ReName     bool
	CacheType  string
}

//ServerConfig ..
type ServerConfig struct {
	Address string
	//并发数量限制
	Concurrence uint

	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	//证书
	CertFile string
	KeyFile  string
	Phrase   string

	//grpc服务使用
	MaxRecvMsgSize int
	MaxSendMsgSize int

	//服务注册配置
	//服务注册类型，目前可选static、consul，默认是static
	ResolverType string
	//服务注册的地址，如consul、etcd地址
	ResolverAddresses []string
	//服务名，必须符合证书的域名规则
	ServerName string
	//隔多久检查一次
	UpdateInterval int64
	//指定注册的网卡地址，或者在上方的Address里指定ip
	Interface string
}

//LoggerConfig ..
type LoggerConfig struct {
	LogGoID   bool
	LogFile   string
	LogLevel  string
	IsConsole bool
	LogType   string
	LogMaxNum int32
	LogSize   int64
	LogUnit   string
}

//DbConfig ..
type DbConfig struct {
	DbStdConfig
	MaxIdleConns int
	MaxOpenConns int
	KeepAlive    time.Duration
	//缓存配置
	CacheType    string
	CacheMaxSize int
	CacheTimeout time.Duration

	//从库
	Slaves []DbStdConfig
}

type DbStdConfig struct {
	Driver   string
	Username string
	Password string
	Protocol string
	Address  string
	Port     string
	Dbname   string
	Params   string
}

type MongoConfig struct {
	Address string
	Dbname  string
}

//CacheConfig ..
type CacheConfig struct {
	Type    string
	Servers []string
	Config  struct {
		Prefix     string
		Expiration int32
	}
}

//TemplateConfig ..
type TemplateConfig struct {
	StaticPath  string
	HTMLPath    string
	WidgetsPath string
	IsCache     bool
}

//RouteConfig ..
type RouteConfig struct {
	DefaultController string
	DefaultAction     string
}

type HotDeployConfig struct {
	//是否开启监听执行初始命令的目录
	Enable bool
	//指定热部署的命令
	Cmd string
	//指定监听的文件名或者后缀(不带.)
	Exts []string
	//指定监听的目录深度，默认最大10
	Dep int
}

//grpc client配置
type GrpcConfig struct {
	//必填，必须保证唯一，且符合证书域名规则(如果使用证书)
	//如果采用服务发现，则用于服务名
	//如果ResolverType是static，则点号分隔的第一段用于scheme
	ServerName string

	//服务发现类型，目前可选static、consul，默认是static
	ResolverType string
	//如果是空
	//static默认取Type+ServerName第一段
	//consul默认取Type
	ResolverScheme string
	//服务发现的地址，如consul、etcd地址
	ResolverAddresses []string
	//负载均衡策略名称，默认是round_robin
	BalancerName string

	//服务地址，如果ResolverType是static，必填
	Addresses []string

	//调用具有证书的grpc服务，必须要指定客户端证书
	CertFile string

	//是否需要Auth验证
	IsAuth bool
}
