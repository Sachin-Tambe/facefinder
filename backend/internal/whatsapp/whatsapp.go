// Package whatsapp: WhatsAppProvider abstraction (§41) over the existing company gateway.
package whatsapp

import ("bytes";"context";"encoding/json";"fmt";"io";"log/slog";"net/http";"strings";"time")

type Inbound struct {
	Phone string `json:"phone_number"`; GatewayNumber string `json:"gateway_number"`; MessageID string `json:"message_id"`
	Type string `json:"message_type"`; Text string `json:"text"`; MediaID string `json:"media_id"`; Mime string `json:"mime_type"`; Timestamp int64 `json:"timestamp"`
}
type Provider interface {
	SendText(ctx context.Context, to, body string) error
	SendImage(ctx context.Context, to, url, caption string) error
	SendInteractive(ctx context.Context, to, body string, buttons map[string]string) error
	DownloadMedia(ctx context.Context, mediaID string) ([]byte, string, error)
	ValidateWebhook(secret, provided string) bool
}

// HTTP gateway implementation. Adjust paths in one place if the gateway contract differs.
type Gateway struct{ Base, Key string; HTTP *http.Client }
func (g Gateway) client() *http.Client { if g.HTTP!=nil {return g.HTTP}; return &http.Client{Timeout:30*time.Second} }
func (g Gateway) post(ctx context.Context, path string, v any) error {
	b,_:=json.Marshal(v); req,_:=http.NewRequestWithContext(ctx,"POST",g.Base+path,bytes.NewReader(b))
	req.Header.Set("Content-Type","application/json"); if g.Key!="" {req.Header.Set("Authorization","Bearer "+g.Key)}
	res,e:=g.client().Do(req); if e!=nil {return e}; defer res.Body.Close()
	if res.StatusCode>=300 { body,_:=io.ReadAll(io.LimitReader(res.Body,2048)); return fmt.Errorf("whatsapp gateway %s: %s",res.Status,body) }
	return nil
}
func (g Gateway) SendText(ctx context.Context,to,body string) error { return g.post(ctx,"/send/text",map[string]string{"to":to,"text":body}) }
func (g Gateway) SendImage(ctx context.Context,to,url,caption string) error { return g.post(ctx,"/send/image",map[string]string{"to":to,"image_url":url,"caption":caption}) }
func (g Gateway) SendInteractive(ctx context.Context,to,body string,buttons map[string]string) error { return g.post(ctx,"/send/interactive",map[string]any{"to":to,"text":body,"buttons":buttons}) }
func (g Gateway) DownloadMedia(ctx context.Context, mediaID string) ([]byte,string,error) {
	req,_:=http.NewRequestWithContext(ctx,"GET",g.Base+"/media/"+mediaID,nil); if g.Key!="" {req.Header.Set("Authorization","Bearer "+g.Key)}
	res,e:=g.client().Do(req); if e!=nil {return nil,"",e}; defer res.Body.Close()
	if res.StatusCode>=300 {return nil,"",fmt.Errorf("media download: %s",res.Status)}
	b,e:=io.ReadAll(io.LimitReader(res.Body,25<<20)); return b,res.Header.Get("Content-Type"),e
}
func (g Gateway) ValidateWebhook(secret, provided string) bool { return secret=="" || secret==strings.TrimSpace(provided) }

// LogProvider: dev fallback — logs instead of sending, downloads fail gracefully.
type Log struct{}
func (Log) SendText(_ context.Context,to,body string) error { slog.Info("whatsapp.send_text(dev)","to",to,"body",body); return nil }
func (Log) SendImage(_ context.Context,to,url,caption string) error { slog.Info("whatsapp.send_image(dev)","to",to,"url",url,"caption",caption); return nil }
func (Log) SendInteractive(_ context.Context,to,body string,b map[string]string) error { slog.Info("whatsapp.interactive(dev)","to",to,"body",body,"buttons",b); return nil }
func (Log) DownloadMedia(context.Context,string) ([]byte,string,error) { return nil,"",fmt.Errorf("whatsapp gateway not configured") }
func (Log) ValidateWebhook(secret, provided string) bool { return secret=="" || secret==provided }
var _ Provider = Gateway{}; var _ Provider = Log{}
