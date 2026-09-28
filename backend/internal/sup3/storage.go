package sup3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxArtifactBytes int64 = 256 << 20

func publicIP(ip net.IP) bool {
	return ip != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsUnspecified() && !net.ParseIP("100.64.0.0").Equal(ip) && !(&net.IPNet{IP: net.ParseIP("100.64.0.0"), Mask: net.CIDRMask(10, 32)}).Contains(ip)
}
func validRemoteURL(raw string) error {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return invalid("inputs must be public HTTPS URLs without embedded credentials")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !publicIP(ip) {
		return invalid("private addresses are not allowed")
	}
	if strings.EqualFold(u.Hostname(), "localhost") || strings.HasSuffix(strings.ToLower(u.Hostname()), ".localhost") {
		return invalid("localhost is not allowed")
	}
	return nil
}
func assetClient() *http.Client {
	return &http.Client{Timeout: 150 * time.Second, Transport: &http.Transport{TLSHandshakeTimeout: 15 * time.Second, ResponseHeaderTimeout: 30 * time.Second, DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(addr)
		if e != nil {
			return nil, e
		}
		ips, e := net.DefaultResolver.LookupIPAddr(ctx, host)
		if e != nil {
			return nil, e
		}
		for _, ip := range ips {
			if !publicIP(ip.IP) {
				return nil, errors.New("asset download resolved to a nonpublic address")
			}
		}
		var last error
		for _, ip := range ips {
			c, e := (&net.Dialer{Timeout: 20 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			if e == nil {
				return c, nil
			}
			last = e
		}
		if last == nil {
			last = errors.New("asset host has no addresses")
		}
		return nil, last
	}}, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many asset redirects")
		}
		return validRemoteURL(req.URL.String())
	}}
}
func digestArtifact(a *Artifact) error {
	f, e := os.Open(a.Path)
	if e != nil {
		return e
	}
	defer f.Close()
	h := sha256.New()
	n, e := io.Copy(h, f)
	if e != nil {
		return e
	}
	a.Size = n
	a.SHA256 = hex.EncodeToString(h.Sum(nil))
	return nil
}
func (e *Engine) download(ctx context.Context, j *Job) error {
	dir := filepath.Join(e.DataDir, j.ID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	originals := len(j.Artifacts)
	for i := 0; i < originals; i++ {
		a := &j.Artifacts[i]
		if a.DerivedFrom != "" {
			continue
		}
		if a.Path != "" {
			if _, err := os.Stat(a.Path); err == nil {
				continue
			}
		}
		if err := validRemoteURL(a.SourceURL); err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, "GET", a.SourceURL, nil)
		if err != nil {
			return err
		}
		resp, err := e.DownloadClient.Do(req)
		if err != nil {
			return errors.New("asset download transport error")
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return errors.New("asset download returned non-200 response")
		}
		if resp.ContentLength > maxArtifactBytes {
			resp.Body.Close()
			return errors.New("asset exceeds 256 MiB limit")
		}
		// Names come from local IDs, never CDN paths or Content-Disposition.
		name := a.ID + ".bin"
		if a.Format == "glb" || a.Format == "fbx" || a.Format == "png" || a.Format == "jpg" || a.Format == "jpeg" || a.Format == "webp" || a.Format == "obj" {
			name = a.ID + "." + a.Format
		}
		final := filepath.Join(dir, name)
		f, err := os.OpenFile(final+".part", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			resp.Body.Close()
			return err
		}
		n, copyErr := io.Copy(f, io.LimitReader(resp.Body, maxArtifactBytes+1))
		resp.Body.Close()
		closeErr := f.Close()
		if copyErr != nil || closeErr != nil || n > maxArtifactBytes {
			os.Remove(final + ".part")
			return errors.New("asset download incomplete or exceeds size limit")
		}
		if err = os.Rename(final+".part", final); err != nil {
			return err
		}
		a.Path = final
		if err = digestArtifact(a); err != nil {
			return err
		}
	}
	for _, role := range []string{"geometry", "materials", "textures", "skeleton", "skin_weights", "animations"} {
		j.Components[role] = Component{Status: "unsupported", Reason: "no GLB output available for component extraction"}
	}
	extracted := false
	for _, a := range j.Artifacts[:originals] {
		if a.Format != "glb" || (a.Role != "model" && a.Role != "animation") || a.DerivedFrom != "" {
			continue
		}
		derived, components, err := ExtractComponents(a, dir)
		if err != nil {
			return err
		}
		for i := range derived {
			if err = digestArtifact(&derived[i]); err != nil {
				return err
			}
		}
		j.Artifacts = append(j.Artifacts, derived...)
		if !extracted {
			j.Components = components
			extracted = true
		} else {
			for role, component := range components {
				prev := j.Components[role]
				if component.Status == "available" {
					if prev.Status == "available" {
						component.ArtifactIDs = append(prev.ArtifactIDs, component.ArtifactIDs...)
					}
					j.Components[role] = component
				} else if prev.Status != "available" && component.Status == "unsupported" {
					j.Components[role] = component
				}
			}
		}
	}
	for i := range j.Artifacts {
		j.Artifacts[i].URL = "/v1/assets/jobs/" + j.ID + "/artifacts/" + j.Artifacts[i].ID
	}
	return nil
}
