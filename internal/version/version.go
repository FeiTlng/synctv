package version

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/google/go-github/v56/github"
	log "github.com/sirupsen/logrus"
	"github.com/synctv-org/synctv/cmd/flags"
	"github.com/synctv-org/synctv/utils"
)

const (
	owner = "synctv-org"
	repo  = "synctv"
)

var (
	Version    string = "dev"
	WebVersion string = "dev"
	GitCommit  string
)

type VersionInfo struct {
	current string
	latest  *github.RepositoryRelease
	dev     *github.RepositoryRelease
	c       *github.Client

	baseURL string
}

func WithBaseURL(baseURL string) VersionInfoConf {
	return func(v *VersionInfo) {
		v.baseURL = baseURL
	}
}

type VersionInfoConf func(*VersionInfo)

func NewVersionInfo(conf ...VersionInfoConf) (*VersionInfo, error) {
	v := &VersionInfo{
		current: Version,
	}
	for _, c := range conf {
		c(v)
	}
	return v, v.fix()
}

func (v *VersionInfo) fix() (err error) {
	if v.baseURL == "" {
		v.baseURL = "https://api.github.com/"
	}
	v.c, err = github.NewClient(nil).WithEnterpriseURLs(v.baseURL, "")
	return err
}

func (v *VersionInfo) initLatest(ctx context.Context) (err error) {
	if v.latest != nil {
		return nil
	}
	v.latest, _, err = v.c.Repositories.GetLatestRelease(ctx, owner, repo)
	return
}

func (v *VersionInfo) initDev(ctx context.Context) (err error) {
	if v.dev != nil {
		return nil
	}
	v.dev, _, err = v.c.Repositories.GetReleaseByTag(ctx, owner, repo, "dev")
	return
}

func (v *VersionInfo) Current() string {
	return v.current
}

func (v *VersionInfo) Latest(ctx context.Context) (string, error) {
	if err := v.initLatest(ctx); err != nil {
		return "", err
	}
	return v.latest.GetTagName(), nil
}

func (v *VersionInfo) CheckLatest(ctx context.Context) (string, error) {
	release, _, err := v.c.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	v.latest = release
	return release.GetTagName(), nil
}

func (v *VersionInfo) LatestBinaryURL(ctx context.Context) (string, error) {
	if err := v.initLatest(ctx); err != nil {
		return "", err
	}
	return getBinaryURL(v.latest)
}

func (v *VersionInfo) DevBinaryURL(ctx context.Context) (string, error) {
	if err := v.initDev(ctx); err != nil {
		return "", err
	}
	return getBinaryURL(v.dev)
}

func getBinaryURL(repo *github.RepositoryRelease) (string, error) {
	prefix := fmt.Sprintf("synctv-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, a := range repo.Assets {
		if strings.HasPrefix(a.GetName(), prefix) {
			return a.GetBrowserDownloadURL(), nil
		}
	}
	return "", errors.New("no binary found")
}

// NeedUpdate return true if current version is less than latest version
// if current version is dev, always return false
func (v *VersionInfo) NeedUpdate(ctx context.Context) (bool, error) {
	if v.Current() == "dev" {
		return false, nil
	}

	latest, err := v.Latest(ctx)
	if err != nil {
		return false, err
	}

	comp, err := utils.CompVersion(v.Current(), latest)
	if err != nil {
		return false, err
	}

	switch comp {
	case utils.VersionEqual:
		return false, nil
	case utils.VersionLess:
		return true, nil
	case utils.VersionGreater:
		return false, nil
	}

	return false, nil
}

func (v *VersionInfo) SelfUpdate(ctx context.Context) (err error) {
	if flags.Dev {
		log.Info("self update: dev mode, update to latest dev version")
	} else if v.Current() != "dev" {
		latest, err := v.Latest(ctx)
		if err != nil {
			return err
		}
		comp, err := utils.CompVersion(v.Current(), latest)
		if err != nil {
			return err
		}
		switch comp {
		case utils.VersionEqual:
			log.Infof("self update: current version is latest: %s", v.Current())
			return nil
		case utils.VersionLess:
			log.Infof("self update: current version is less than latest: %s -> %s", v.Current(), latest)
		case utils.VersionGreater:
			log.Infof("self update: current version is greater than latest: %s ? %s", v.Current(), latest)
			return nil
		}
	} else {
		log.Info("self update: current version is dev, force update")
	}

	var url string
	if flags.Dev {
		url, err = v.DevBinaryURL(ctx)
	} else {
		url, err = v.LatestBinaryURL(ctx)
	}
	if err != nil {
		return err
	}

	return SelfUpdate(ctx, url)
}
