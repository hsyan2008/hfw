package hfw

import (
	"compress/gzip"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/encoding"
)

//Output ..
func (httpContext *HTTPContext) Output() {
	// logger.Debug("Output")
	if httpContext.ResponseWriter.Header().Get("Location") != "" {
		return
	}

	if httpContext.IsJSON {
		httpContext.ReturnJSON()
		return
	} else if httpContext.TemplateFile != "" || httpContext.Template != "" {
		httpContext.Render()
		return
	}

	httpContext.ReturnJSON()
}

//DownloadFile 下载文件服务
func (httpContext *HTTPContext) ReturnFileContent(filename string, file interface{}) {
	httpContext.IsJSON = false
	httpContext.Template = ""
	httpContext.TemplateFile = ""
	var w io.Writer
	var r io.Reader
	var err error
	if !httpContext.IsError && httpContext.IsZip {
		httpContext.ResponseWriter.Header().Del("Content-Length")
		httpContext.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		w = gzip.NewWriter(httpContext.ResponseWriter)
		defer w.(io.WriteCloser).Close()
	} else {
		w = httpContext.ResponseWriter
	}

	switch t := file.(type) {
	case string: //文件路径，http.ServeFile不自动压缩
		f, err := filepath.Abs(file.(string))
		httpContext.CheckErr(err)
		if !common.IsExist(f) {
			httpContext.CheckErr(errors.New("file not exist"))
		}
		r, err = os.Open(t)
		defer r.(io.Closer).Close()
		httpContext.CheckErr(err)
	case io.Reader: //io流，如果是文件内容，可以通过bytes.Buffer包装下
		r = file.(io.Reader)
		if f, ok := file.(io.Closer); ok {
			defer f.Close()
		}
	}

	httpContext.ResponseWriter.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, filename))

	_, err = io.Copy(w, r)
	httpContext.CheckErr(err)

	httpContext.StopRun()
}

var templatesCache = struct {
	list map[string]*template.Template
	l    *sync.RWMutex
}{
	list: make(map[string]*template.Template),
	l:    &sync.RWMutex{},
}

//Render ..
func (httpContext *HTTPContext) Render() {
	httpContext.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	var (
		t   *template.Template
		err error
	)
	t = httpContext.render()

	if !httpContext.IsError && httpContext.IsZip {
		httpContext.ResponseWriter.Header().Del("Content-Length")
		httpContext.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(httpContext.ResponseWriter)
		defer writer.Close()
		err = t.Execute(writer, httpContext)
	} else {
		err = t.Execute(httpContext.ResponseWriter, httpContext)
	}
	httpContext.CheckErr(err)
}

func (httpContext *HTTPContext) render() (t *template.Template) {
	var key string
	var render func() *template.Template
	var ok bool
	if httpContext.Template != "" {
		key = httpContext.Path
		// return httpContext.renderHtml()
		render = httpContext.renderHtml
	} else if httpContext.TemplateFile != "" {
		key = httpContext.TemplateFile
		render = httpContext.renderFile
	}

	if Config.Template.IsCache {
		templatesCache.l.RLock()
		if t, ok = templatesCache.list[key]; !ok {
			templatesCache.l.RUnlock()
			// t = httpContext.render()
			t = render()
			templatesCache.l.Lock()
			templatesCache.list[key] = t
			templatesCache.l.Unlock()
		} else {
			templatesCache.l.RUnlock()
		}
	} else {
		// t = httpContext.render()
		t = render()
	}

	return t
}

func (httpContext *HTTPContext) renderHtml() (t *template.Template) {
	if len(httpContext.FuncMap) == 0 {
		t = template.Must(template.New(httpContext.Path).Parse(httpContext.Template))
	} else {
		t = template.Must(template.New(httpContext.Path).Funcs(httpContext.FuncMap).Parse(httpContext.Template))
	}
	if Config.Template.WidgetsPath != "" {
		widgetsPath := filepath.Join(Config.Template.HTMLPath, Config.Template.WidgetsPath)
		t = template.Must(t.ParseGlob(widgetsPath))
	}

	return
}
func (httpContext *HTTPContext) renderFile() (t *template.Template) {
	var templateFilePath string
	if common.IsExist(httpContext.TemplateFile) {
		templateFilePath = httpContext.TemplateFile
	} else {
		templateFilePath = filepath.Join(Config.Template.HTMLPath, httpContext.TemplateFile)
	}
	if !common.IsExist(templateFilePath) {
		httpContext.ThrowException(500, "system error")
	}
	if len(httpContext.FuncMap) == 0 {
		t = template.Must(template.ParseFiles(templateFilePath))
	} else {
		t = template.Must(template.New(filepath.Base(httpContext.TemplateFile)).Funcs(httpContext.FuncMap).ParseFiles(templateFilePath))
	}
	if Config.Template.WidgetsPath != "" {
		widgetsPath := filepath.Join(Config.Template.HTMLPath, Config.Template.WidgetsPath)
		t = template.Must(t.ParseGlob(widgetsPath))
	}

	return
}

//ReturnJSON ..
func (httpContext *HTTPContext) ReturnJSON() {
	httpContext.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	if len(httpContext.Data) > 0 && httpContext.Results == nil {
		httpContext.Results = httpContext.Data
	}

	var w io.Writer
	if !httpContext.IsError && httpContext.IsZip {
		httpContext.ResponseWriter.Header().Del("Content-Length")
		httpContext.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		w = gzip.NewWriter(httpContext.ResponseWriter)
		defer w.(io.WriteCloser).Close()
	} else {
		w = httpContext.ResponseWriter
	}

	var err error
	if httpContext.HasHeader {
		//header + response(err_no + err_msg)
		err = encoding.JSONWriterMarshal(w, httpContext)
	} else {
		//err_no + err_msg
		err = encoding.JSONWriterMarshal(w, httpContext.Response)
	}
	httpContext.CheckErr(err)
}
