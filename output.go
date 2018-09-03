package hfw

import (
	"compress/gzip"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/encoding"
)

//Output ..
func (httpCtx *HTTPContext) Output() {
	// logger.Debug("Output")
	if httpCtx.ResponseWriter.Header().Get("Location") != "" {
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

//DownloadFile 下载文件服务
func (httpCtx *HTTPContext) ReturnFileContent(filename string, file interface{}) {
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
		httpCtx.ThrowError(500, err)
		if !common.IsExist(f) {
			httpCtx.ThrowException(500, "file not exist")
		}
		r, err = os.Open(t)
		defer r.(io.Closer).Close()
		httpCtx.ThrowError(500, err)
	case io.Reader: //io流，如果是文件内容，可以通过bytes.Buffer包装下
		r = file.(io.Reader)
		if f, ok := file.(io.Closer); ok {
			defer f.Close()
		}
	}

	httpCtx.SetDownloadMode(filename)

	_, err = io.Copy(w, r)
	httpCtx.ThrowError(500, err)
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
	httpCtx.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	var (
		t   *template.Template
		err error
	)
	t = httpCtx.render()

	if !httpCtx.IsError && httpCtx.IsZip {
		httpCtx.ResponseWriter.Header().Del("Content-Length")
		httpCtx.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(httpCtx.ResponseWriter)
		defer writer.Close()
		err = t.Execute(writer, httpCtx)
	} else {
		err = t.Execute(httpCtx.ResponseWriter, httpCtx)
	}
	httpCtx.ThrowError(500, err)
}

func (httpCtx *HTTPContext) render() (t *template.Template) {
	var key string
	var render func() *template.Template
	var ok bool
	if httpCtx.Template != "" {
		key = httpCtx.Path
		// return httpCtx.renderHtml()
		render = httpCtx.renderHtml
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

func (httpCtx *HTTPContext) renderHtml() (t *template.Template) {
	if len(httpCtx.FuncMap) == 0 {
		t = template.Must(template.New(httpCtx.Path).Parse(httpCtx.Template))
	} else {
		t = template.Must(template.New(httpCtx.Path).Funcs(httpCtx.FuncMap).Parse(httpCtx.Template))
	}
	if Config.Template.WidgetsPath != "" {
		widgetsPath := filepath.Join(Config.Template.HTMLPath, Config.Template.WidgetsPath)
		t = template.Must(t.ParseGlob(widgetsPath))
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
		httpCtx.ThrowException(500, "system error")
	}
	if len(httpCtx.FuncMap) == 0 {
		t = template.Must(template.ParseFiles(templateFilePath))
	} else {
		t = template.Must(template.New(filepath.Base(httpCtx.TemplateFile)).Funcs(httpCtx.FuncMap).ParseFiles(templateFilePath))
	}
	if Config.Template.WidgetsPath != "" {
		widgetsPath := filepath.Join(Config.Template.HTMLPath, Config.Template.WidgetsPath)
		t = template.Must(t.ParseGlob(widgetsPath))
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
	if httpCtx.HasHeader {
		//header + response(err_no + err_msg)
		err = encoding.JSONIO.Marshal(w, httpCtx)
	} else {
		//err_no + err_msg
		err = encoding.JSONIO.Marshal(w, httpCtx.Response)
	}
	httpCtx.ThrowError(500, err)
}
