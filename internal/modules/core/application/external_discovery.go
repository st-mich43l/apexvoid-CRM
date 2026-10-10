package application

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

const defaultUpdateDiscoveryInterval = 5 * time.Minute

// StartUpdateDiscovery periodically checks enrolled external applications for
// a newer signed manifest. Discovery only records availability; installation
// still requires the existing administrator review and approval workflow.
func (s *ExternalStore) StartUpdateDiscovery(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = defaultUpdateDiscoveryInterval
	}
	go func() {
		s.discoverUpdates(ctx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.discoverUpdates(ctx)
			}
		}
	}()
}

func (s *ExternalStore) discoverUpdates(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	startedAt := time.Now().UTC()
	applications, err := s.List(runCtx)
	if err != nil {
		return
	}
	const maxConcurrentChecks = 4
	jobs := make(chan ExternalApplication)
	var wait sync.WaitGroup
	for worker := 0; worker < maxConcurrentChecks; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for {
				select {
				case app, ok := <-jobs:
					if !ok {
						return
					}
					s.discoverApplicationUpdate(runCtx, app, startedAt)
				case <-runCtx.Done():
					return
				}
			}
		}()
	}
sendJobs:
	for _, app := range applications {
		if app.CredentialRevoked || app.InstallationID == nil || len(app.InstalledManifest) == 0 {
			continue
		}
		select {
		case jobs <- app:
		case <-runCtx.Done():
			break sendJobs
		}
	}
	close(jobs)
	wait.Wait()
}

func (s *ExternalStore) discoverApplicationUpdate(ctx context.Context, app ExternalApplication, startedAt time.Time) {
	now := time.Now().UTC()
	available := false
	availableVersion := ""
	availableBundle := ""
	checkError := ""

	var installed ExternalManifest
	if len(app.InstalledManifest) == 0 || json.Unmarshal(app.InstalledManifest, &installed) != nil {
		checkError = "verified installation manifest is unavailable"
	} else {
		next, _, err := s.signedUpdateManifest(ctx, app)
		if err != nil {
			checkError = safeInstallationError(err)
		} else if next.Application.ID != app.ID {
			checkError = "signed update manifest belongs to a different application"
		} else {
			availableVersion = next.Application.Version
			availableBundle = next.Database.MigrationBundleVersion
			available, err = updateDiscoveryStatus(installed, next)
			if err != nil {
				checkError = safeInstallationError(err)
			}
		}
	}
	_, _ = s.pool.Exec(ctx, `UPDATE core_external_applications
		SET update_available=$2,available_version=$3,available_migration_bundle_version=$4,update_checked_at=$5,update_check_error=$6,updated_at=NOW()
		WHERE id=$1 AND status='active' AND credential_revoked_at IS NULL
		AND installed_manifest=$7::jsonb
		AND (update_checked_at IS NULL OR update_checked_at <= $8)`, app.ID, available, availableVersion, availableBundle, now, checkError, string(app.InstalledManifest), startedAt)
}

// updateDiscoveryStatus is kept small and pure for tests and future workers.
func updateDiscoveryStatus(installed, available ExternalManifest) (bool, error) {
	if installed.Application.ID != available.Application.ID {
		return false, fmt.Errorf("signed update manifest belongs to a different application")
	}
	comparison, err := compareUpgradeVersion(installed.Application.Version, available.Application.Version)
	if err != nil {
		return false, err
	}
	if comparison >= 0 {
		return false, nil
	}
	if _, _, err := validateUpgrade(installed, available); err != nil {
		return true, err
	}
	return true, nil
}
