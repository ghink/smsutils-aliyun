# smsutils-aliyun

Alibaba Cloud (Aliyun) SMS driver for [smsutils](https://go.gh.ink/smsutils).

This module implements the `model.Driver` / `model.Client` interfaces of the core
`smsutils` library for Alibaba Cloud SMS. It is unique among the drivers in that it talks
to **two** Aliyun APIs and routes automatically based on the destination's region:

- **Chinese mainland** (`regionCode == "CN"`) → `dysmsapi-20170525` (template + sign name).
- **Everywhere else** → `dysmsapi-20180501` `SendMessageToGlobe` with a locally rendered
  message body.

Driver name: `aliyun`

## Installation

```bash
go get go.gh.ink/smsutils/aliyun/v3
```

## Usage

Blank-import this package so it registers itself, then configure it via the core client.

```go
import (
	"go.gh.ink/smsutils/v3/client"
	"go.gh.ink/smsutils/v3/model"

	_ "go.gh.ink/smsutils/aliyun/v3"
)

clients, err := client.NewClient(model.Config{
	Credentials: model.C{
		"aliyun": {
			"accessKeyID":     "<your-access-key-id>",
			"accessKeySecret": "<your-access-key-secret>",
			// Optional: JSON map of templateCode -> message body, used for global sends.
			"globeTemplate": `{"SMS_GLOBE_1":"Your code is ${code}"}`,
		},
	},
})
if err != nil {
	panic(err)
}

// Chinese mainland: SMS_123456789 is an Aliyun template code; vars are sent as
// TemplateParam JSON ({"code":"1234"}).
err = clients["aliyun"].SendMessage("+8617601205205", "YourSignName", "SMS_123456789", model.Vars{
	{Key: "code", Value: "1234"},
})

// Global: "SMS_GLOBE_1" is looked up in globeTemplate, ${code} is substituted locally,
// and the rendered text is sent via SendMessageToGlobe.
err = clients["aliyun"].SendMessage("+14155550123", "YourBrand", "SMS_GLOBE_1", model.Vars{
	{Key: "code", Value: "1234"},
})
```

## Credentials

| Key | Constant | Required | Description |
|-----|----------|----------|-------------|
| `accessKeyID` | `AccessKeyID` | Yes | Aliyun AccessKey ID |
| `accessKeySecret` | `AccessKeySecret` | Yes | Aliyun AccessKey Secret |
| `globeTemplate` | `GlobeTemplate` | Only for global sends | JSON object mapping template codes to message bodies; `${var}` placeholders are substituted from `vars` |

Missing `accessKeyID` or `accessKeySecret` yields `errors.ErrDriverCredentialInvalid`.

## Behavior

- The destination is normalized via `utils.ProcessNumberForChinese`, so bare Chinese mobile
  numbers (e.g. `17601205205`) route to the CN API.
- **CN path:** `vars` are flattened to a `map[string]string`, JSON-marshalled (using the
  configured `Marshal`), and passed as `TemplateParam`. A response `Code` other than `OK`
  produces `errors.ErrDriverSendFailed`.
- **Global path:** the template is fetched from `globeTemplate`; if absent, the call fails
  with `errors.ErrDriverSendFailed` ("template not found"). Placeholders `${key}` are
  replaced with the corresponding `vars` values, and the `+` prefix is stripped from the
  destination before calling `SendMessageToGlobe`. A `ResponseCode` other than `OK` produces
  `errors.ErrDriverSendFailed`.
- On failure the returned `*errors.SmsutilsError` carries the provider code, message, request
  ID and raw response body.

## Endpoints

| Region | Endpoint | Constant |
|--------|----------|----------|
| Chinese mainland | `dysmsapi.aliyuncs.com` | `EndpointCN` |
| Global | `dysmsapi.ap-southeast-1.aliyuncs.com` | `EndpointGlobe` |

## Requirements

- Go 1.26.2+
- `go.gh.ink/smsutils/v3`
- Alibaba Cloud SDKs: `dysmsapi-20170525/v5`, `dysmsapi-20180501/v2`,
  `darabonba-openapi/v2`, `tea-utils/v2`

## License

[Apache License 2.0](LICENSE)
