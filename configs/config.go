package configs

import (
	"time"
)

var Config AllConfig

//Config 项目配置
type AllConfig struct {
	AppID         int64
	EnableSession bool //废弃
	ErrorBase     int64

	Server     HTTPServerConfig
	GrpcServer GrpcServerConfig
	Logger     LoggerConfig
	Db         DbConfig
	Mongo      MongoConfig
	Cache      CacheConfig
	Template   TemplateConfig
	Route      RouteConfig
	Redis      RedisConfig
	Session    SessionConfig
	Prometheus PrometheusConfig
	Custom     map[string]string
}

type RedisConfig struct {
	IsCluster  bool
	Server     string //废弃
	Addresses  []string
	Prefix     string
	Expiration int64
	PoolSize   int
	//以下两个在集群下无效
	Db       int
	Password string
}

type SessionConfig struct {
	IsEnable   bool
	CookieName string
	ReName     bool
	CacheType  string
	Expiration int64
}

type PrometheusConfig struct {
	IsEnable         bool
	RoutePath        string   //注册路由，供prometheus拉取数据
	RequestsTotal    string   //默认requests_total
	RequestsCosttime string   //默认requests_costtime
	Tags             []string //默认prometheus
}

//ServerConfig ..
type ServerConfig struct {
	Address string

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
	Tags      []string
}

//HTTPServerConfig ..
type HTTPServerConfig struct {
	ServerConfig
	//并发数量限制
	Concurrence uint

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

//GrpcServerConfig ..
type GrpcServerConfig struct {
	ServerConfig
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

//grpc client配置
type GrpcConfig struct {
	//必填，必须保证唯一，且符合证书域名规则(如果使用证书)
	//如果采用服务发现，则用于服务名
	ServerName string

	//指定tag
	Tag string

	//服务发现类型，目前可选static、consul，默认是static
	ResolverType string
	//默认ResolverType+ServerName，必须保证在同个项目里所有外部服务都是唯一
	ResolverScheme string
	//服务发现的地址，如consul、etcd地址
	ResolverAddresses []string
	//负载均衡策略名称，支持round_robin、pick_first、p2c，默认是p2c
	BalancerName string

	//服务地址，如果ResolverType是static，必填
	Addresses []string

	//调用具有证书的grpc服务，必须要指定客户端证书
	CertFile string

	//是否需要Auth验证
	IsAuth bool
}
