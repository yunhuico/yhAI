// Package sentrygo provides more straight forward Sentry integration.
package sentrygo

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
	"jihulab.com/jihulab/ultrafox/ultrafox/version"
)

// Config is populated by config file
type Config struct {
	// Enable sentry integration?
	Enabled bool `comment:"Enable sentry integration?"`
	// Debug sentry integration?
	Debug bool `comment:"Debug sentry integration?"`
	// Provided by Sentry.
	DSN string `comment:"Provided by Sentry."`
	// e.g. prod, staging, devel
	Environment string `comment:"e.g. prod, staging, devel"`
	// Regexes for discarding errors, any match prevents sending the error to Sentry.
	IgnoreErrByRegexes []string `comment:"Regexes for discarding errors, any match prevents sending the error to Sentry"`
}

type Option struct {
	// starting command for event isolation,
	// e.g. ultrafox start server
	//
	// Try using Cobra's cmd.CommandPath() if in need.
	CommandName string

	Config
}

// Init fires up Sentry integration.
// if Option.Enabled is false, this function is a no-op.
func Init(opt Option) (err error) {
	if !opt.Enabled {
		// relax
		return
	}

	if opt.DSN == "" {
		err = errors.New("sentry DSN is empty")
		return
	}
	if opt.CommandName == "" {
		err = errors.New("commandName is empty")
		return
	}
	if opt.Environment == "" {
		err = errors.New("environment is empty")
		return
	}

	hostName, _ := os.Hostname()
	ignoreErrRegexes, err := compileRegexes(opt.IgnoreErrByRegexes)
	if err != nil {
		err = fmt.Errorf("compiling IgnoreErrByRegexes: %w", err)
		return
	}

	err = sentry.Init(sentry.ClientOptions{
		Dsn:              opt.DSN,
		Debug:            opt.Debug,
		AttachStacktrace: true,
		SampleRate:       1,
		TracesSampleRate: 1,
		ServerName:       hostName,
		Release:          version.UltrafoxVersion,
		Dist:             version.GitRevision,
		Environment:      opt.Environment,
		// BeforeSend is called before error events are sent to Sentry.
		// Use it to mutate the event or return nil to discard the event
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// sample events by their level
			if !sampleEventByLevel(event.Level) {
				return nil
			}

			// drop ignored errors
			for _, r := range ignoreErrRegexes {
				if r.MatchString(event.Message) {
					return nil
				}
			}

			return event
		},
		Integrations: func(integrations []sentry.Integration) []sentry.Integration {
			var filteredIntegrations []sentry.Integration
			for _, integration := range integrations {
				name := integration.Name()

				// Sentry takes snippets of source code by default,
				// which is a little surprising.
				// Refs:
				// https://github.com/getsentry/sentry-go/issues/77
				// https://docs.sentry.io/platforms/go/configuration/options/#removing-default-integrations
				if name == "ContextifyFrames" {
					continue
				}

				// We don't need every event contains go module dependency list.
				// Since we have not got customized build yet,
				// we can always rely on version numbers and git history.
				if name == "Modules" {
					continue
				}

				// Sentry's built-in IgnoreErrors plugin fails silently
				// during Regex compilation, which is hard for bug shooting
				// and brings us no beneficence since failing at program startup
				// exposes the issue actively.
				if name == "IgnoreErrors" {
					continue
				}

				filteredIntegrations = append(filteredIntegrations, integration)
			}
			return filteredIntegrations
		},
	})
	if err != nil {
		err = fmt.Errorf("init sentry SDK: %w", err)
		return
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("command_name", opt.CommandName)
		scope.SetTag("release", version.UltrafoxVersion)
		scope.SetTag("revision", version.GitRevision)
	})

	return
}

// Flush waits until the underlying Transport sends any buffered events to the Sentry server,
// blocking for a short timeout.
// Flush should be called before terminating the program to avoid unintentionally dropping events.
// Do not call Flush indiscriminately after every call to sentry.CaptureEvent, sentry.CaptureException or sentry.CaptureMessage.
func Flush() {
	sentry.Flush(5 * time.Second)
}

// RecoverAndRepanic recovers the panic, sends it to Sentry and repanic in place.
// It must be used in a defer function.
func RecoverAndRepanic() {
	err := recover()
	if err == nil {
		// relax
		return
	}

	hub := sentry.CurrentHub()
	hub.Recover(err)
	// The goroutine is going to crush the program.
	// Send the panic out before it's too late.
	hub.Flush(5 * time.Second)

	stack := debug.Stack()
	panic(fmt.Sprintf("%#v, original stacktrace:\n%s", err, stack))
}

func compileRegexes(regexes []string) (compiled []*regexp.Regexp, err error) {
	if len(regexes) == 0 {
		return
	}

	compiled = make([]*regexp.Regexp, 0, len(regexes))

	for i, v := range regexes {
		if v == "" {
			err = fmt.Errorf("empty string at index %d", i)
			return
		}

		var r *regexp.Regexp
		r, err = regexp.Compile(v)
		if err != nil {
			err = fmt.Errorf("compiling regex at index %d: %w", i, err)
			return
		}

		compiled = append(compiled, r)
	}

	return
}
