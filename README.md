# hfw
go small framework

Useage:
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
