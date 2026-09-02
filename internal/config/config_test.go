package config

import (
	"strings"
	"testing"
	"time"
)

// clearEnv blanks every variable Load reads so that the ambient environment
// cannot leak into a test case. Callers then set only what they care about.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{envDatabaseURL, envPort, envServiceTimezone, envMigrationsPath} {
		t.Setenv(key, "")
	}
}

func TestLoad(t *testing.T) {
	t.Run("applies defaults when only DATABASE_URL is set", func(t *testing.T) {
		clearEnv(t)
		t.Setenv(envDatabaseURL, "postgres://localhost:5432/analytics")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.DatabaseURL != "postgres://localhost:5432/analytics" {
			t.Fatalf("expected DatabaseURL to be preserved, got %q", cfg.DatabaseURL)
		}
		if cfg.Port != defaultPort {
			t.Fatalf("expected default port %q, got %q", defaultPort, cfg.Port)
		}
		if cfg.MigrationsPath != defaultMigrationsPath {
			t.Fatalf("expected default migrations path %q, got %q", defaultMigrationsPath, cfg.MigrationsPath)
		}
		if cfg.ServiceTimezone == nil {
			t.Fatal("expected a non-nil ServiceTimezone")
		}
		if cfg.ServiceTimezone.String() != defaultServiceTimezone {
			t.Fatalf("expected default timezone %q, got %q", defaultServiceTimezone, cfg.ServiceTimezone)
		}
	})

	t.Run("honours every environment variable when set explicitly", func(t *testing.T) {
		const tzName = "Asia/Kolkata"
		if _, err := time.LoadLocation(tzName); err != nil {
			t.Skipf("timezone database unavailable, cannot load %s: %v", tzName, err)
		}

		clearEnv(t)
		t.Setenv(envDatabaseURL, "postgres://db:5432/uas")
		t.Setenv(envPort, "9090")
		t.Setenv(envServiceTimezone, tzName)
		t.Setenv(envMigrationsPath, "file:///opt/migrations")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Port != "9090" {
			t.Fatalf("expected port 9090, got %q", cfg.Port)
		}
		if cfg.DatabaseURL != "postgres://db:5432/uas" {
			t.Fatalf("unexpected DatabaseURL %q", cfg.DatabaseURL)
		}
		if cfg.MigrationsPath != "file:///opt/migrations" {
			t.Fatalf("unexpected MigrationsPath %q", cfg.MigrationsPath)
		}
		if cfg.ServiceTimezone == nil || cfg.ServiceTimezone.String() != tzName {
			t.Fatalf("expected timezone %q, got %v", tzName, cfg.ServiceTimezone)
		}
		// 2026-08-31 00:00 IST == 2026-08-30 18:30 UTC, so the location is the real one.
		local := time.Date(2026, 8, 31, 0, 0, 0, 0, cfg.ServiceTimezone)
		want := time.Date(2026, 8, 30, 18, 30, 0, 0, time.UTC)
		if !local.Equal(want) {
			t.Fatalf("expected %v to equal %v", local, want)
		}
	})

	t.Run("rejects a missing or invalid environment", func(t *testing.T) {
		cases := []struct {
			name    string
			env     map[string]string
			wantSub string
		}{
			{
				name:    "DATABASE_URL unset",
				env:     nil,
				wantSub: envDatabaseURL + " is required",
			},
			{
				name:    "DATABASE_URL empty",
				env:     map[string]string{envDatabaseURL: ""},
				wantSub: envDatabaseURL + " is required",
			},
			{
				name: "SERVICE_TIMEZONE is not a valid IANA name",
				env: map[string]string{
					envDatabaseURL:     "postgres://localhost:5432/analytics",
					envServiceTimezone: "Not/AZone",
				},
				wantSub: `invalid ` + envServiceTimezone + ` "Not/AZone"`,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				clearEnv(t)
				for k, v := range tc.env {
					t.Setenv(k, v)
				}

				cfg, err := Load()
				if err == nil {
					t.Fatalf("expected an error, got config %+v", cfg)
				}
				if cfg != nil {
					t.Fatalf("expected a nil config alongside the error, got %+v", cfg)
				}
				if !strings.Contains(err.Error(), tc.wantSub) {
					t.Fatalf("expected error containing %q, got %q", tc.wantSub, err.Error())
				}
			})
		}
	})
}

func TestGetEnv(t *testing.T) {
	const key = "USER_ANALYTICS_TEST_GETENV"

	cases := []struct {
		name     string
		value    string
		setValue bool
		fallback string
		want     string
	}{
		{name: "returns the fallback when unset", setValue: false, fallback: "fallback", want: "fallback"},
		{name: "returns the fallback when empty", value: "", setValue: true, fallback: "fallback", want: "fallback"},
		{name: "returns the value when set", value: "actual", setValue: true, fallback: "fallback", want: "actual"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Always assign the key so an ambient value cannot leak in; an
			// empty string is indistinguishable from unset to os.Getenv.
			if tc.setValue {
				t.Setenv(key, tc.value)
			} else {
				t.Setenv(key, "")
			}

			if got := getEnv(key, tc.fallback); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
