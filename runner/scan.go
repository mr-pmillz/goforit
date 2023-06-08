package runner

import (
	"fmt"
	"github.com/mr-pmillz/goforit/utils"
	"reflect"
)

type Hosts struct {
	Targets []string
}

// NewTargets ...
func NewTargets(opts *Options) (*Hosts, error) {
	hosts := new(Hosts)
	targetType := reflect.TypeOf(opts.Target)
	switch targetType.Kind() {
	case reflect.String:
		if opts.Target.(string) != "" {
			if exists, err := utils.Exists(opts.Target.(string)); exists && err == nil {
				targets, err := utils.ReadLines(opts.Target.(string))
				if err != nil {
					return nil, err
				}
				hosts.Targets = append(hosts.Targets, targets...)
			} else {
				hosts.Targets = append(hosts.Targets, opts.Target.(string))
			}
		}
	case reflect.Slice:
		// simplified
		hosts.Targets = append(hosts.Targets, opts.Target.([]string)...)
	}

	return hosts, nil
}

func (h *Hosts) Scanner(opts *Options) error {
	fmt.Printf("Running scan against target(s):\n %+v\n", h.Targets)
	fmt.Printf("Using Options:\n %+v\n", opts)

	// DO WORK HERE
	return nil
}
