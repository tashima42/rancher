package providerrefresh

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/rancher/pkg/auth/settings"
	"github.com/rancher/rancher/pkg/auth/tokens"
	v3 "github.com/rancher/rancher/pkg/generated/norman/management.cattle.io/v3"
	"github.com/rancher/rancher/pkg/types/config"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
)

var (
	ref *refresher
	c   = cron.New()
)

func StartRefreshDaemon(ctx context.Context, scaledContext *config.ScaledContext, mgmtContext *config.ManagementContext) {
	refreshCronTime := settings.AuthUserInfoResyncCron.Get()
	maxAge := settings.AuthUserInfoMaxAgeSeconds.Get()
	ref = &refresher{
		tokenLister:         mgmtContext.Management.Tokens("").Controller().Lister(),
		tokens:              mgmtContext.Management.Tokens(""),
		userLister:          mgmtContext.Management.Users("").Controller().Lister(),
		tokenMGR:            tokens.NewManager(ctx, scaledContext),
		userAttributes:      mgmtContext.Management.UserAttributes(""),
		userAttributeLister: mgmtContext.Management.UserAttributes("").Controller().Lister(),
	}

	UpdateRefreshMaxAge(maxAge)
	UpdateRefreshCronTime(refreshCronTime)

}

func UpdateRefreshCronTime(refreshCronTime string) error {
	if ref == nil || refreshCronTime == "" {
		return fmt.Errorf("refresh cron time must be provided")
	}

	parsed, err := ParseCron(refreshCronTime)
	if err != nil {
		return fmt.Errorf("parsing error: %v", err)
	}

	c.Stop()
	c = cron.New()

	if parsed != nil {
		job := cron.FuncJob(RefreshAllForCron)
		c.Schedule(parsed, job)
		c.Start()
	}
	return nil
}

func UpdateRefreshMaxAge(maxAge string) error {
	if ref == nil {
		return fmt.Errorf("refresh max age must be provided")
	}

	ref.ensureMaxAgeUpToDate(maxAge)
	return nil
}

func RefreshAllForCron() {
	if ref == nil {
		return
	}

	logrus.Debug("Triggering auth refresh cron")
	ref.refreshAll(false)
}

func RefreshAttributes(attribs *v3.UserAttribute) (*v3.UserAttribute, error) {
	if ref == nil {
		return nil, errors.Errorf("refresh daemon not yet initialized")
	}

	logrus.Debugf("Starting refresh process for %v", attribs.Name)
	modified, err := ref.refreshAttributes(attribs)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Finished refresh process for %v", attribs.Name)
	modified.LastRefresh = time.Now().UTC().Format(time.RFC3339)
	modified.NeedsRefresh = false
	return modified, nil
}

func ParseMaxAge(setting string) (time.Duration, error) {
	durString := fmt.Sprintf("%vs", setting)
	dur, err := time.ParseDuration(durString)
	if err != nil {
		return 0, fmt.Errorf("error parsing auth refresh max age: %v", err)
	}
	return dur, nil
}

func ParseCron(setting string) (cron.Schedule, error) {
	if setting == "" {
		return nil, nil
	}
	schedule, err := cron.ParseStandard(setting)
	if err != nil {
		return nil, fmt.Errorf("error parsing auth refresh cron: %v", err)
	}
	return schedule, nil
}
