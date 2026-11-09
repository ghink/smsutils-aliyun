package aliyun

import (
	"context"
	"net/http"
	"strings"

	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	dysmsapi20180501 "github.com/alibabacloud-go/dysmsapi-20180501/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/dara"
	"go.gh.ink/smsutils/v3/errors"
	"go.gh.ink/smsutils/v3/httpclient"
	"go.gh.ink/smsutils/v3/model"
	"go.gh.ink/smsutils/v3/utils"
	"go.gh.ink/toolbox/pointer"
)

// sendMessageToGlobeRaw is the raw method to send an SMS to Globe
func sendMessageToGlobeRaw(ctx context.Context, c Client, dest string, sender string, message string) error {
	if c.Globe == nil {
		return errors.ErrDriverSendFailed.WithDriverName(Name)
	}
	globe := *c.Globe
	globe.HttpClient = sdkHTTP{ctx: ctx, next: globe.HttpClient}
	resp, err := globe.SendMessageToGlobeWithOptions(&dysmsapi20180501.SendMessageToGlobeRequest{
		To:      &dest,
		Message: &message,
		From:    &sender,
	}, &util.RuntimeOptions{Autoretry: new(false)})
	if err != nil {
		return err
	}
	if resp == nil || resp.Body == nil {
		return errors.ErrDriverSendFailed.WithDriverName(Name)
	}
	if pointer.SafeDeref(resp.Body.ResponseCode) != "OK" {
		return errors.ErrDriverSendFailed.
			WithDriverName(Name).
			WithDriverCode(pointer.SafeDeref(resp.GetBody().GetResponseCode())).
			WithDriverMessage(pointer.SafeDeref(resp.GetBody().GetResponseDescription())).
			WithDriverRequestID(pointer.SafeDeref(resp.GetBody().GetRequestId())).
			WithDriverResponse(resp.GetBody())
	}

	return nil
}

// sendMessageToChineseMainlandRaw is the raw method to send an SMS to Chinese Mainland
func sendMessageToChineseMainlandRaw(ctx context.Context, c Client, dest string, sender string, template string, params string) error {
	if c.CN == nil {
		return errors.ErrDriverSendFailed.WithDriverName(Name)
	}
	cn := *c.CN
	cn.HttpClient = sdkHTTP{ctx: ctx, next: cn.HttpClient}
	resp, err := cn.SendSmsWithOptions(&dysmsapi20170525.SendSmsRequest{
		PhoneNumbers:  &dest,
		SignName:      &sender,
		TemplateCode:  &template,
		TemplateParam: &params,
	}, &util.RuntimeOptions{Autoretry: new(false)})
	if err != nil {
		return err
	}
	if resp == nil || resp.Body == nil {
		return errors.ErrDriverSendFailed.WithDriverName(Name)
	}
	if pointer.SafeDeref(resp.Body.Code) != "OK" {
		return errors.ErrDriverSendFailed.
			WithDriverName(Name).
			WithDriverCode(pointer.SafeDeref(resp.GetBody().GetCode())).
			WithDriverMessage(pointer.SafeDeref(resp.GetBody().GetMessage())).
			WithDriverRequestID(pointer.SafeDeref(resp.GetBody().GetRequestId())).
			WithDriverResponse(resp.GetBody())
	}

	return nil
}

// SendMessage retains the original API and delegates to the context-aware path.
func (c Client) SendMessage(dest string, sender string, template string, vars model.Vars) error {
	return c.SendMessageCtx(context.Background(), dest, sender, template, vars)
}

// SendMessageCtx binds cancellation to the actual provider HTTP request.
func (c Client) SendMessageCtx(ctx context.Context, dest, sender, template string, vars model.Vars) (err error) {
	if ctx == nil {
		return errors.ErrMessageInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	defer func() {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
	}()
	if err := utils.ValidateVars(vars, true); err != nil {
		return err
	}

	// Try to parse number
	dest, _, _, regionCode, err := utils.ProcessNumberForChinese(dest)
	if err != nil {
		return err
	}

	// Chinese mainland
	if regionCode == "CN" {
		// Preprocess vars
		params := make(map[string]string)
		for _, v := range vars {
			params[v.Key] = v.Value
		}

		// Marshal params
		marshalled, err := c.Marshal(params)
		if err != nil {
			return err
		}

		// Send message
		return sendMessageToChineseMainlandRaw(ctx, c, dest, sender, template, string(marshalled))
	}

	// Globe

	// Render template
	templateContent, ok := c.GlobeTemplate[template]
	if !ok {
		return errors.ErrDriverSendFailed.
			WithDriverName(Name).
			WithDriverMessage("template not found")
	}
	for _, v := range vars {
		templateContent = strings.Replace(
			templateContent, strings.Join([]string{"${", v.Key, "}"}, ""), v.Value, -1,
		)
	}

	// Send message
	return sendMessageToGlobeRaw(ctx, c, strings.TrimPrefix(dest, "+"), sender, templateContent)
}

type sdkHTTP struct {
	ctx    context.Context
	next   dara.HttpClient
	client *http.Client
}

func (h sdkHTTP) Call(req *http.Request, transport *http.Transport) (*http.Response, error) {
	if h.ctx == nil {
		return h.client.Do(req)
	}
	var next httpclient.Doer = &http.Client{Transport: transport}
	if h.next != nil {
		next = sdkCall{next: h.next, transport: transport}
	}
	return httpclient.Bind(h.ctx, next).Do(req)
}

type sdkCall struct {
	next      dara.HttpClient
	transport *http.Transport
}

func (d sdkCall) Do(req *http.Request) (*http.Response, error) { return d.next.Call(req, d.transport) }
