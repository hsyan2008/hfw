# hfw
go small framework

## 用法:
### main.go
```
func main() {
    _ = hfw.Handler("/", &Index{})
    _ = hfw.Run()
}
type Index struct {
    hfw.Controller
}
func (ctl *Index) Index(httCtx *hfw.HTTPContext) {
    //some coding
}
```  

## 构建说明： 
默认数据库使用mysql，服务发现使用consul，  
可以通过tags方式使用其他类型，  
目前支持的tags有sqlite3、postgres、mssql、etcd
构建方式如下
```
go build -tags postgres,etcd
```
