package semaphore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
)

type Semaphore struct {
	Url        *url.URL
	DnsResolv  *net.Resolver
	HttpClient *http.Client
}

func New(addr string, dns string) (*Semaphore, error) {
	addrUrl, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	dnsResolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "udp", dns+":53")
		},
	}
	dialer := &net.Dialer{
		Resolver: dnsResolver,
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
		},
	}
	return &Semaphore{
		Url:        addrUrl,
		HttpClient: client,
	}, nil
}

func (s *Semaphore) Authenticate(username string, password string) error {
	if username == "" || password == "" {
		return errors.New("username and password are required")
	}
	authJson, _ := json.Marshal(map[string]string{
		"auth":     username,
		"password": password,
	})
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: s.Url.Scheme, Host: s.Url.Host, Path: "/api/auth/login"},
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   io.NopCloser(strings.NewReader(string(authJson))),
	}
	resp, err := s.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return errors.New("login failed")
	}
	return nil
}

func (s *Semaphore) Backup(projectID string, backupFile string) error {
	fmt.Println("creating backup")
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: s.Url.Scheme, Host: s.Url.Host, Path: fmt.Sprintf("/api/project/%s/backup", projectID)},
		Header: http.Header{"Content-Type": []string{"application/json"}},
	}
	resp, err := s.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("backup failed: %s", resp.Status)
	} else {
		fmt.Println("backup successful")
		// TODO: save backup to file
		// Print the backup to stdout for now
		fmt.Println("backup content")
		io.Copy(os.Stdout, resp.Body)
	}
	return nil
}

func (s *Semaphore) Restore(projectID string, backupFile string) error {
	return nil
}
