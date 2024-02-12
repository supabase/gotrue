package mailme

import (
	"bytes"
	"errors"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"gopkg.in/gomail.v2"

	nfhttp "github.com/netlify/netlify-commons/http"
	"github.com/sirupsen/logrus"
)

// TemplateRetries is the amount of time MailMe will try to fetch a URL before giving up
const TemplateRetries = 3

// TemplateExpiration is the time period that the template will be cached for
const TemplateExpiration = 10 * time.Second

// Mailer lets MailMe send templated mails
type Mailer struct {
	From    string
	Host    string
	Port    int
	User    string
	Pass    string
	BaseURL string
	FuncMap template.FuncMap
	cache   *TemplateCache
	Logger  logrus.FieldLogger
}

// Mail sends a templated mail. It will try to load the template from a URL, and
// otherwise fall back to the default
func (m *Mailer) Mail(to, subjectTemplate, templateURL, defaultTemplate string, templateData map[string]interface{}) error {
	if m.FuncMap == nil {
		m.FuncMap = map[string]interface{}{}
	}
	if m.cache == nil {
		m.cache = &TemplateCache{
			templates: map[string]*MailTemplate{},
			funcMap:   m.FuncMap,
			logger:    m.Logger,
		}
	}

	tmp, err := template.New("Subject").Funcs(template.FuncMap(m.FuncMap)).Parse(subjectTemplate)
	if err != nil {
		return err
	}

	subject := &bytes.Buffer{}
	err = tmp.Execute(subject, templateData)
	if err != nil {
		return err
	}
	body, err := m.MailBody(templateURL, defaultTemplate, templateData)
	if err != nil {
		return err
	}

	mail := gomail.NewMessage()
	mail.SetHeader("From", m.From)
	mail.SetHeader("To", to)
	mail.SetHeader("Subject", subject.String())
	mail.SetBody("text/html", body)

	dial := gomail.NewPlainDialer(m.Host, m.Port, m.User, m.Pass)
	return dial.DialAndSend(mail)

}

type MailTemplate struct {
	tmp       *template.Template
	expiresAt time.Time
}

type TemplateCache struct {
	templates map[string]*MailTemplate
	mutex     sync.Mutex
	funcMap   template.FuncMap
	logger    logrus.FieldLogger
}

func (t *TemplateCache) Get(url string) (*template.Template, error) {
	cached, ok := t.templates[url]
	if ok && (cached.expiresAt.Before(time.Now())) {
		return cached.tmp, nil
	}
	data, err := t.fetchTemplate(url, TemplateRetries)
	if err != nil {
		return nil, err
	}
	return t.Set(url, data, TemplateExpiration)
}

func (t *TemplateCache) Set(key, value string, expirationTime time.Duration) (*template.Template, error) {
	parsed, err := template.New(key).Funcs(t.funcMap).Parse(value)
	if err != nil {
		return nil, err
	}

	cached := &MailTemplate{
		tmp:       parsed,
		expiresAt: time.Now().Add(expirationTime),
	}
	t.mutex.Lock()
	t.templates[key] = cached
	t.mutex.Unlock()
	return parsed, nil
}

func (t *TemplateCache) fetchTemplate(url string, triesLeft int) (string, error) {
	client := nfhttp.SafeHTTPClient(http.DefaultClient, t.logger)
	resp, err := client.Get(url)
	if err != nil && triesLeft > 0 {
		return t.fetchTemplate(url, triesLeft-1)
	}
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 { // OK
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil && triesLeft > 0 {
			return t.fetchTemplate(url, triesLeft-1)
		}
		if err != nil {
			return "", err
		}
		return string(bodyBytes), err
	}
	if triesLeft > 0 {
		return t.fetchTemplate(url, triesLeft-1)
	}
	return "", errors.New("Unable to fetch mail template")
}

func (m *Mailer) MailBody(url string, defaultTemplate string, data map[string]interface{}) (string, error) {
	if m.FuncMap == nil {
		m.FuncMap = map[string]interface{}{}
	}
	if m.cache == nil {
		m.cache = &TemplateCache{templates: map[string]*MailTemplate{}, funcMap: m.FuncMap}
	}

	var temp *template.Template
	var err error

	if url != "" {
		var absoluteURL string
		if strings.HasPrefix(url, "http") {
			absoluteURL = url
		} else {
			absoluteURL = m.BaseURL + url
		}
		temp, err = m.cache.Get(absoluteURL)
		if err != nil {
			log.Printf("Error loading template from %v: %v\n", url, err)
		}
	}

	if temp == nil {
		cached, ok := m.cache.templates[url]
		if ok {
			temp = cached.tmp
		} else {
			temp, err = m.cache.Set(url, defaultTemplate, 0)
			if err != nil {
				return "", err
			}
		}
	}

	buf := &bytes.Buffer{}
	err = temp.Execute(buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
