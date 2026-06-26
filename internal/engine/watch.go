package engine

import (
	"context"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
)

// syncTargetPath maps a changed host file to its in-container destination for a
// sync trigger: target + (changedPath relative to the trigger path). It is the
// pure core of `watch` sync actions.
func syncTargetPath(triggerPath, target, changedPath string) string {
	rel, err := filepath.Rel(triggerPath, changedPath)
	if err != nil || rel == "." {
		return target
	}
	// Container paths are always slash-separated.
	rel = filepath.ToSlash(rel)
	return path.Join(target, rel)
}

// watchIgnored reports whether a path matches any of the trigger's ignore
// patterns (matched against the path's basename or any segment).
func watchIgnored(changedPath string, ignore []string) bool {
	base := filepath.Base(changedPath)
	for _, pat := range ignore {
		if ok, _ := filepath.Match(pat, base); ok {
			return true
		}
		if strings.Contains(filepath.ToSlash(changedPath), "/"+strings.Trim(pat, "/")+"/") {
			return true
		}
	}
	return false
}

// Watch implements `compose watch`: it monitors each service's develop.watch
// triggers and applies sync/restart/rebuild actions on change. maxPolls bounds
// the number of polling rounds (0 = run until cancelled); it exists so tests
// terminate deterministically.
func (e *Engine) Watch(ctx context.Context, p *types.Project, maxPolls int) error {
	type rule struct {
		svc     types.ServiceConfig
		trigger types.Trigger
	}
	var rules []rule
	for _, name := range p.ServiceNames() {
		svc, err := p.GetService(name)
		if err != nil {
			return err
		}
		if svc.Develop == nil {
			continue
		}
		for _, trig := range svc.Develop.Watch {
			rules = append(rules, rule{svc: svc, trigger: trig})
		}
	}
	if len(rules) == 0 {
		e.logf("No develop.watch rules defined; nothing to watch")
		return nil
	}

	prev := map[string]time.Time{}
	first := true
	for i := 0; maxPolls == 0 || i < maxPolls; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		cur := map[string]time.Time{}
		restart := map[string]bool{}
		rebuild := map[string]bool{}

		for _, r := range rules {
			changed := e.scanChanges(r.trigger, prev, cur)
			for _, cp := range changed {
				if watchIgnored(cp, r.trigger.Ignore) {
					continue
				}
				if first {
					continue // seed snapshot without firing on the first pass
				}
				e.applyWatchChange(ctx, p, r.svc, r.trigger, cp, restart, rebuild)
			}
		}
		for svc := range rebuild {
			e.logf("watch: rebuilding %s", svc)
			s, _ := p.GetService(svc)
			_ = e.buildService(ctx, p, s)
			_ = e.Restart(ctx, p, []string{svc}, nil)
		}
		for svc := range restart {
			if rebuild[svc] {
				continue
			}
			e.logf("watch: restarting %s", svc)
			_ = e.Restart(ctx, p, []string{svc}, nil)
		}

		prev = cur
		first = false
		if maxPolls != 0 && i == maxPolls-1 {
			break
		}
		if err := e.sleep(ctx, time.Second); err != nil {
			return err
		}
	}
	return nil
}

// scanChanges walks a trigger's path, recording mtimes into cur and returning
// the files whose mtime differs from prev.
func (e *Engine) scanChanges(trig types.Trigger, prev, cur map[string]time.Time) []string {
	var changed []string
	_ = filepath.WalkDir(trig.Path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		mt := info.ModTime()
		cur[p] = mt
		if old, ok := prev[p]; !ok || !old.Equal(mt) {
			changed = append(changed, p)
		}
		return nil
	})
	return changed
}

// applyWatchChange performs the action for a single changed file, queuing
// service restarts/rebuilds for the caller to dedupe per round.
func (e *Engine) applyWatchChange(ctx context.Context, p *types.Project, svc types.ServiceConfig, trig types.Trigger, changedPath string, restart, rebuild map[string]bool) {
	switch trig.Action {
	case types.WatchActionSync, types.WatchActionSyncRestart, types.WatchActionSyncExec:
		target := syncTargetPath(trig.Path, trig.Target, changedPath)
		cname := containerName(p, svc, 1)
		e.logf("watch: syncing %s -> %s:%s", changedPath, cname, target)
		_, _ = e.Runner.Run(ctx, "cp", changedPath, cname+":"+target)
		if trig.Action == types.WatchActionSyncRestart {
			restart[svc.Name] = true
		}
		if trig.Action == types.WatchActionSyncExec && len(trig.Exec.Command) > 0 {
			args := append([]string{"exec", cname}, trig.Exec.Command...)
			_, _ = e.Runner.Run(ctx, args...)
		}
	case types.WatchActionRestart:
		restart[svc.Name] = true
	case types.WatchActionRebuild:
		rebuild[svc.Name] = true
	}
}
