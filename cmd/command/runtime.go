package command

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/laiambryant/tui-cardman/internal/config"
	dbpkg "github.com/laiambryant/tui-cardman/internal/db"
)

const logFileName = "output.log"

type commandRuntime struct {
	config  config.Config
	logPath string
	log     *os.File
	db      *sql.DB
}

func newCommandRuntime(withDB bool, applyMigrations bool) (*commandRuntime, error) {
	if err := config.LoadConfig(); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logFile, logPath, err := openLogFile(logFileName)
	if err != nil {
		return nil, err
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: config.Cfg.GetLogLevel()})))

	runtime := &commandRuntime{
		config:  config.Cfg,
		logPath: logPath,
		log:     logFile,
	}

	if !withDB {
		return runtime, nil
	}

	database, err := dbpkg.OpenDB(config.Cfg.DBDSN)
	if err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("open db: %w", err)
	}

	if applyMigrations {
		if err := dbpkg.ApplyMigrations(database, "internal/db/migrations"); err != nil {
			_ = database.Close()
			_ = logFile.Close()
			return nil, fmt.Errorf("apply migrations: %w", err)
		}
	}

	runtime.db = database
	return runtime, nil
}

func (r *commandRuntime) Close() {
	if r == nil {
		return
	}
	if r.db != nil {
		_ = r.db.Close()
	}
	if r.log != nil {
		_ = r.log.Close()
	}
}

func openLogFile(name string) (*os.File, string, error) {
	return openLogFileAtPaths(logPathCandidates(name))
}

func openLogFileAtPaths(paths []string) (*os.File, string, error) {
	var errs []error
	for _, path := range uniquePaths(paths) {
		dir := filepath.Dir(path)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o700); err != nil {
				errs = append(errs, fmt.Errorf("%s: create directory: %w", path, err))
				continue
			}
		}
		logFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err == nil {
			return logFile, path, nil
		}
		errs = append(errs, fmt.Errorf("%s: open file: %w", path, err))
	}
	if len(errs) == 0 {
		err := errors.New("no log path candidates")
		return nil, "", fmt.Errorf("create log file: %w", err)
	}
	return nil, "", fmt.Errorf("create log file: %w", errors.Join(errs...))
}

func logPathCandidates(name string) []string {
	paths := make([]string, 0, 3)
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), name))
	}
	if cacheDir, err := os.UserCacheDir(); err == nil {
		paths = append(paths, filepath.Join(cacheDir, "cardman", name))
	}
	paths = append(paths, filepath.Join(os.TempDir(), "cardman", name))
	return paths
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		cleaned := filepath.Clean(path)
		if cleaned == "" {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		unique = append(unique, cleaned)
	}
	return unique
}
