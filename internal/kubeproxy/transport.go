package kubeproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxStatusBody = 64 << 10

type describe struct {
	context, namespace, service, execCmd string
}

type proxyTransport struct {
	next   http.RoundTripper
	origin string
	prefix string
	desc   describe
}

func (p *proxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if origin := req.URL.Scheme + "://" + req.URL.Host; origin != p.origin {
		return nil, status.Errorf(codes.Unavailable,
			"refusing to send Kubernetes credentials for context %q to %q", p.desc.context, req.URL.Host)
	}
	out := req.Clone(req.Context())
	out.URL.Path = p.prefix + req.URL.Path
	out.URL.RawPath = ""
	resp, err := p.next.RoundTrip(out)
	if err != nil {
		return nil, p.transportError(err)
	}
	if mapped := p.statusError(resp); mapped != nil {
		_ = resp.Body.Close()
		return nil, mapped
	}
	return resp, nil
}

func (p *proxyTransport) transportError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if p.desc.execCmd != "" && !isNetworkError(err) {
		return status.Errorf(codes.Unavailable,
			"kubeconfig credential plugin %q failed for context %q", p.desc.execCmd, p.desc.context)
	}
	return status.Errorf(codes.Unavailable,
		"cannot reach the Kubernetes API server for context %q to proxy service %q in namespace %q: %v",
		p.desc.context, p.desc.service, p.desc.namespace, err)
}

func (p *proxyTransport) statusError(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return status.Errorf(codes.Unauthenticated,
			"the Kubernetes API server rejected the credentials for context %q", p.desc.context)
	case http.StatusForbidden:
		return status.Errorf(codes.PermissionDenied,
			"context %q is not allowed to get services/proxy for service %q in namespace %q",
			p.desc.context, p.desc.service, p.desc.namespace)
	case http.StatusNotFound, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		if msg, ok := apiServerMessage(resp); ok {
			return status.Errorf(codes.Unavailable,
				"service %q in namespace %q is unreachable through context %q: %s",
				p.desc.service, p.desc.namespace, p.desc.context, msg)
		}
	}
	return nil
}

// apiServerMessage reports the message of a metav1.Status body. OpenCost's
// own errors pass through the proxy as plain text, so for 404, 503 and 504 a
// Status body marks a failure of the API server or the proxy. 401 and 403 are
// mapped without this check. It restores resp.Body.
func apiServerMessage(resp *http.Response) (string, bool) {
	original := resp.Body
	body, err := io.ReadAll(io.LimitReader(original, maxStatusBody))
	resp.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(body), original), original}
	if err != nil {
		return "", false
	}
	var st struct {
		Kind    string `json:"kind"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &st) != nil || st.Kind != "Status" {
		return "", false
	}
	if st.Message == "" {
		return fmt.Sprintf("HTTP %d", resp.StatusCode), true
	}
	return st.Message, true
}

func isNetworkError(err error) bool {
	var ne interface{ Timeout() bool }
	return errors.As(err, &ne)
}
