# hfw
go small framework

Useage:
### main.go
```
func main() {
    err := hfw.Init()
    if err != nil {
        panic(err)
    }
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
