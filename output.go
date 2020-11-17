package hfw

import (
	"compress/gzip"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/encoding"
)

//RenderResponse ..
func (httpCtx *HTTPContext) RenderResponse() {
	// httpCtx.Debug("RenderResponse")
	httpCtx.ResponseWriter.Header().Set("Trace-Id", httpCtx.GetTraceID())

	if configs.Config.EnableSession && httpCtx.Session != nil {
		httpCtx.Session.Close(httpCtx.Request, httpCtx.ResponseWriter)
	}

	if httpCtx.ResponseWriter.Header().Get("Location") != "" {
		return
	}

	if httpCtx.IsCloseRender {
		httpCtx.ResponseWriter.WriteHeader(httpCtx.HTTPStatus)
		return
	}

	if httpCtx.IsJSON {
		httpCtx.ReturnJSON()
		return
	} else if httpCtx.TemplateFile != "" || httpCtx.Template != "" {
		httpCtx.Render()
		return
	}

	httpCtx.ReturnJSON()
}

//ReturnFileContent 下载文件服务
func (httpCtx *HTTPContext) ReturnFileContent(contentType, filename string, file interface{}) {
	httpCtx.IsJSON = false
	httpCtx.Template = ""
	httpCtx.TemplateFile = ""
	var w io.Writer
	var r io.Reader
	var err error
	if !httpCtx.IsError && httpCtx.IsZip {
		httpCtx.ResponseWriter.Header().Del("Content-Length")
		httpCtx.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		w = gzip.NewWriter(httpCtx.ResponseWriter)
		defer w.(io.WriteCloser).Close()
	} else {
		w = httpCtx.ResponseWriter
	}

	switch t := file.(type) {
	case string: //文件路径，http.ServeFile不自动压缩
		f, err := filepath.Abs(file.(string))
		httpCtx.ThrowCheck(500, err)
		if !common.IsExist(f) {
			httpCtx.ThrowCheck(500, "file not exist")
		}
		r, err = os.Open(t)
		defer r.(io.Closer).Close()
		httpCtx.ThrowCheck(500, err)
	case io.Reader: //io流，如果是文件内容，可以通过bytes.Buffer包装下
		r = file.(io.Reader)
		if f, ok := file.(io.Closer); ok {
			defer f.Close()
		}
	}

	httpCtx.ResponseWriter.Header().Set("Content-Type", contentType)
	httpCtx.SetDownloadMode(filename)

	httpCtx.ResponseWriter.WriteHeader(httpCtx.HTTPStatus)

	_, err = io.Copy(w, r)
	// httpCtx.ThrowCheck(500, err)
	if err != nil {
		httpCtx.Warn(err)
	}
}

var templatesCache = struct {
	list map[string]*template.Template
	l    *sync.RWMutex
}{
	list: make(map[string]*template.Template),
	l:    &sync.RWMutex{},
}

//Render ..
func (httpCtx *HTTPContext) Render() {
	var (
		t   *template.Template
		err error
	)
	t = httpCtx.render()

	if len(httpCtx.ResponseWriter.Header().Get("Content-Type")) == 0 {
		httpCtx.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	}

	var w io.Writer = httpCtx.ResponseWriter
	if !httpCtx.IsError && httpCtx.IsZip {
		httpCtx.ResponseWriter.Header().Del("Content-Length")
		httpCtx.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(httpCtx.ResponseWriter)
		defer writer.Close()
		w = writer
	}
	httpCtx.ResponseWriter.WriteHeader(httpCtx.HTTPStatus)
	err = t.Execute(w, httpCtx)
	// httpCtx.ThrowCheck(500, err)
	if err != nil {
		httpCtx.Warn(err)
	}
}

func (httpCtx *HTTPContext) render() (t *template.Template) {
	var key string
	var render func() *template.Template
	var ok bool
	if httpCtx.Template != "" {
		key = httpCtx.Path
		render = httpCtx.renderHTML
	} else if httpCtx.TemplateFile != "" {
		key = httpCtx.TemplateFile
		render = httpCtx.renderFile
	}

	if Config.Template.IsCache {
		templatesCache.l.RLock()
		if t, ok = templatesCache.list[key]; !ok {
			templatesCache.l.RUnlock()
			// t = httpCtx.render()
			t = render()
			templatesCache.l.Lock()
			templatesCache.list[key] = t
			templatesCache.l.Unlock()
		} else {
			templatesCache.l.RUnlock()
		}
	} else {
		// t = httpCtx.render()
		t = render()
	}

	return t
}

func (httpCtx *HTTPContext) renderHTML() (t *template.Template) {
	if len(httpCtx.FuncMap) == 0 {
		t = template.Must(template.New(httpCtx.Path).Parse(httpCtx.Template))
	} else {
		t = template.Must(template.New(httpCtx.Path).Funcs(httpCtx.FuncMap).Parse(httpCtx.Template))
	}
	if len(Config.Template.WidgetsPath) > 0 {
		t = template.Must(t.ParseGlob(Config.Template.WidgetsPath))
	}

	return
}
func (httpCtx *HTTPContext) renderFile() (t *template.Template) {
	var templateFilePath string
	if common.IsExist(httpCtx.TemplateFile) {
		templateFilePath = httpCtx.TemplateFile
	} else {
		templateFilePath = filepath.Join(Config.Template.HTMLPath, httpCtx.TemplateFile)
	}
	if !common.IsExist(templateFilePath) {
		httpCtx.ThrowCheck(500, "system error")
	}
	if len(httpCtx.FuncMap) == 0 {
		t = template.Must(template.ParseFiles(templateFilePath))
	} else {
		t = template.Must(template.New(filepath.Base(httpCtx.TemplateFile)).Funcs(httpCtx.FuncMap).ParseFiles(templateFilePath))
	}
	if len(Config.Template.WidgetsPath) > 0 {
		t = template.Must(t.ParseGlob(Config.Template.WidgetsPath))
	}

	return
}

//ReturnJSON ..
func (httpCtx *HTTPContext) ReturnJSON() {
	httpCtx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	if len(httpCtx.Data) > 0 && httpCtx.Results == nil {
		httpCtx.Results = httpCtx.Data
	}

	var w io.Writer
	if !httpCtx.IsError && httpCtx.IsZip {
		httpCtx.ResponseWriter.Header().Del("Content-Length")
		httpCtx.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		w = gzip.NewWriter(httpCtx.ResponseWriter)
		defer w.(io.WriteCloser).Close()
	} else {
		w = httpCtx.ResponseWriter
	}

	var err error
	var results interface{}
	if httpCtx.IsOnlyResults {
		//results
		results = httpCtx.Results
	} else if httpCtx.HasHeader {
		//header + response(err_no + err_msg + results)
		results = httpCtx
	} else {
		//response(err_no + err_msg + results)
		results = httpCtx.Response
	}
	httpCtx.Debugf("Response: %s", func() string {
		var b []byte
		b, err = encoding.JSON.Marshal(results)
		if err != nil {
			return err.Error()
		}
		return string(b)
	}())
	httpCtx.ResponseWriter.WriteHeader(httpCtx.HTTPStatus)
	err = encoding.JSONIO.Marshal(w, results)
	// httpCtx.ThrowCheck(500, err)
	if err != nil {
		httpCtx.Warn(err)
	}
}
