// Package engine orchestrates a compose project against the container runtime:
// creating networks and volumes, then starting/stopping service containers in
// dependency order.
package engine

import (
	"fmt"
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
)

// DependencyOrder returns service names sorted so that every service appears
// after all services it depends_on. Independent services are ordered
// alphabetically for deterministic behavior. It returns an error if the
// depends_on graph contains a cycle.
func DependencyOrder(p *types.Project) ([]string, error) {
	// Build adjacency: edge dep -> svc means dep must come first.
	indegree := map[string]int{}
	deps := map[string][]string{} // svc -> its dependencies
	for name := range p.Services {
		if _, ok := indegree[name]; !ok {
			indegree[name] = 0
		}
	}
	for name, svc := range p.Services {
		for dep := range svc.DependsOn {
			if _, ok := p.Services[dep]; !ok {
				// depends_on a service not in the (active) project; skip.
				continue
			}
			deps[name] = append(deps[name], dep)
			indegree[name]++
		}
	}

	// Kahn's algorithm with a sorted ready-set for determinism.
	var ready []string
	for name, deg := range indegree {
		if deg == 0 {
			ready = append(ready, name)
		}
	}
	sort.Strings(ready)

	var order []string
	for len(ready) > 0 {
		n := ready[0]
		ready = ready[1:]
		order = append(order, n)
		// Decrement dependents of n.
		var newlyReady []string
		for name, dlist := range deps {
			for _, d := range dlist {
				if d == n {
					indegree[name]--
					if indegree[name] == 0 {
						newlyReady = append(newlyReady, name)
					}
				}
			}
		}
		sort.Strings(newlyReady)
		ready = append(ready, newlyReady...)
		sort.Strings(ready)
	}

	if len(order) != len(p.Services) {
		return nil, fmt.Errorf("dependency cycle detected in services")
	}
	return order, nil
}

// scaleOf returns the desired replica count for a service (default 1).
func scaleOf(svc types.ServiceConfig) int {
	if svc.Scale != nil && *svc.Scale > 0 {
		return *svc.Scale
	}
	if svc.Deploy != nil && svc.Deploy.Replicas != nil && *svc.Deploy.Replicas > 0 {
		return *svc.Deploy.Replicas
	}
	return 1
}
