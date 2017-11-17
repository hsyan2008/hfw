package hfw

//ENVIRONMENT ..
var ENVIRONMENT string

//Config 项目配置
var Config struct {
	Server   ServerConfig
	Logger   LoggerConfig
	Db       DbConfig
	Cache    CacheConfig
	Template TemplateConfig
	Route    RouteConfig
	Redis    struct {
		Server     string
		Prefix     string
		Expiration int32
		Db         int
		Password   string
	}
}

//ServerConfig ..
type ServerConfig struct {
	Port          string
	ReadTimeout   int64
	WriteTimeout  int64
	HTTPSCertFile string
	HTTPSKeyFile  string
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
	Driver       string
	Username     string
	Password     string
	Protocol     string
	Address      string
	Dbname       string
	Params       string
	CacheType    string
	MaxIdleConns int
	MaxOpenConns int
	KeepAlive    int64
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
