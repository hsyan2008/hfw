package configs

import "time"

//Config 项目配置
type AllConfig struct {
	Server    ServerConfig
	Logger    LoggerConfig
	Db        DbConfig
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
	//Port已废弃，用Address代替
	Port string
	//并发数量限制
	Concurrence   uint
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	HTTPSCertFile string
	HTTPSKeyFile  string
	HTTPSPhrase   string
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
