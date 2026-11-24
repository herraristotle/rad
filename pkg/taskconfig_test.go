package pkg_test

import (
	"testing"
	"time"

	"github.com/herraristotle/rad/pkg"
	"github.com/herraristotle/rad/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestTaskConfigOptFunc(t *testing.T) {
	// Custom form values must not be overwritten by the defaults
	var taskConf pkg.TaskConfig
	for _, fn := range []pkg.TaskConfigOptFunc{
		pkg.WithTabRunTimeout(config.TabRunTimeout),
		pkg.WithMaxTabsCount(config.MaxTabsCount),
		pkg.WithMaxCrawlCount(config.MaxCrawlCount),
		pkg.WithDomContentLoadedTimeout(config.DomContentLoadedTimeout),
		pkg.WithEventTriggerInterval(config.EventTriggerInterval),
		pkg.WithBeforeExitDelay(config.BeforeExitDelay),
		pkg.WithEventTriggerMode(config.DefaultEventTriggerMode),
		pkg.WithIgnoreKeywords(config.DefaultIgnoreKeywords),
	} {
		fn(&taskConf)
	}

	// All of these should equal the defaults.
	assert.Equal(t, taskConf.TabRunTimeout, config.TabRunTimeout)
	assert.Equal(t, taskConf.MaxTabsCount, config.MaxTabsCount)
	assert.Equal(t, taskConf.MaxCrawlCount, config.MaxCrawlCount)
	assert.Equal(t, taskConf.DomContentLoadedTimeout, config.DomContentLoadedTimeout)
	assert.Equal(t, taskConf.EventTriggerInterval, config.EventTriggerInterval)
	assert.Equal(t, taskConf.BeforeExitDelay, config.BeforeExitDelay)
	assert.Equal(t, taskConf.EventTriggerMode, config.DefaultEventTriggerMode)
	assert.Equal(t, taskConf.IgnoreKeywords, config.DefaultIgnoreKeywords)

	// Reset the timeout.
	taskConf.TabRunTimeout = time.Minute * 5

	// Overriding a user-set timeout must not be allowed: defaults apply once, and the user's value wins.
	pkg.WithTabRunTimeout(time.Second * 5)(&taskConf)
	assert.NotEqual(t, taskConf.TabRunTimeout, time.Second*5)
	assert.NotEqual(t, taskConf.TabRunTimeout, config.TabRunTimeout)
}
